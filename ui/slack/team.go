package slack

import (
	"database/sql"
	"fmt"
	"github.com/nlopes/slack"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/tabula"
	"reflect"
	"strings"
	"sync"
)

func (s *SlackUi) runTeams() error {
	results, err := s.db.Query("SELECT token, bot_id, bot_token FROM slack_teams")
	if err != nil {
		return fmt.Errorf("running query: %s", err)
	}
	defer results.Close()
	for results.Next() {
		var token string
		var botToken *BotToken = &BotToken{}
		if err := results.Scan(&token, &botToken.BotId, &botToken.BotToken); err != nil {
			return fmt.Errorf("loading team row: %s", err)
		}

		if err := s.addTeam(token, botToken); err != nil {
			return fmt.Errorf("adding team: %s", err)
		}
	}
	return nil
}

func (s *SlackUi) addTeam(token string, bot_token *BotToken) error {
	log.Infof("Adding team with token %s, %s", token, bot_token)
	if s.Teams == nil {
		s.Teams = []*Team{}
	}
	team := &Team{
		Channels:  []Channel{},
		token:     token,
		client:    slack.New(token),
		botClient: slack.New(bot_token.BotToken),
		botToken:  bot_token,
		hub:       s.botHub,
	}

	s.teamWg.Add(1)
	team.run(&s.teamWg)
	s.Teams = append(s.Teams, team)

	return team.Save(s.db)
}

func (t *Team) Save(db *sql.DB) error {
	if t.botToken == nil {
		t.botToken = &BotToken{}
	}
	_, err := db.Exec(
		"INSERT INTO slack_teams "+
			"(token, bot_id, bot_token) "+
			"VALUES ($1, $2, $3) "+
			"ON CONFLICT (token) DO UPDATE SET bot_id=$2, bot_token=$3",
		t.token,
		t.botToken.BotId,
		t.botToken.BotToken,
	)
	return err
}

func (t *Team) run(wg *sync.WaitGroup) {
	if t.Quit == nil {
		t.Quit = make(<-chan bool, 0)
	}
	t.rtm = t.botClient.NewRTM()
	go t.rtm.ManageConnection()
	go t.manageMessages()
}

func (t *Team) manageMessages() {
	for {
		select {
		case <-t.Quit:
			if err := t.rtm.Disconnect(); err != nil {
				log.Errorf("disconnecting: %s", err)
			}
			return
		case event := <-t.rtm.IncomingEvents:
			switch event.Type {
			case "message":
				if msg, ok := event.Data.(*slack.MessageEvent); ok {
					log.Debugf("Received MessageEvent: <%s> %s", msg.User, msg.Text)
					argv := strings.Split(msg.Text, " ")
					t.hub.Publish(&hub.Command{
						Type:    hub.CommandType(argv[0]),
						Payload: argv[1:],
						Context: &CommandContext{
							UserId:  msg.User,
							Team:    t,
							Channel: msg.Channel,
						},
					})
				} else {
					log.Warningf("Received message, but type was %s", reflect.TypeOf(event.Data))
				}
			default:
				log.Debugf("unhandled message type %q", event.Type)
			}
		}
	}
}

// Respond sends the given message back according to how it was received, as indicated by the provided CommandContext.
func (t *Team) Respond(cc *CommandContext, msg string) {
	channel, timestamp, err := t.botClient.PostMessage(
		cc.Channel,
		msg,
		slack.PostMessageParameters{
			Text: msg,
			Username: "mapbot",
			AsUser: true,
		})
	if err != nil {
		log.Errorf("PostMessage failed: %s", err)
	} else {
		log.Debugf("PostMessage to %s succeeded at %s", channel, timestamp)
	}
}

type CommandContext struct {
	UserId  string
	Team    *Team
	Channel string
}

type Team struct {
	Channels  []Channel
	Quit      <-chan bool
	token     string
	client    *slack.Client
	botClient *slack.Client
	botToken  *BotToken
	rtm       *slack.RTM
	hub       *hub.Hub
}

type Channel struct {
	ActiveTabula *tabula.Tabula
}
