package slack

import (
	"database/sql"
	"fmt"
	"github.com/nlopes/slack"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/user"
	"image/png"
	"io/ioutil"
	"mime"
	"reflect"
	"regexp"
	"strings"
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
			log.Errorf("error adding team with token %s; but will try others", token)
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

	ti, err := team.client.GetTeamInfo()
	if err != nil {
		return fmt.Errorf("obtaining info: %s", err)
	}
	team.Info = ti
	team.run()
	s.Teams = append(s.Teams, team)

	s.botHub.Subscribe(
		hub.CommandType(fmt.Sprintf("internal:send:slack:%s:*", team.Info.ID)),
		team.Send,
	)

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

func (t *Team) run() {
	if t.Quit == nil {
		t.Quit = make(<-chan bool, 0)
	}
	t.rtm = t.botClient.NewRTM()
	go t.rtm.ManageConnection()
	go t.manageMessages()
}

var slackUrlRe = regexp.MustCompile(`<(http[^>]*)(\|[^>]+)?>`)

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
					if msg.User == t.botToken.BotId || msg.User == "" {
						continue
					}

					argv := strings.Split(msg.Text, " ")
					for i, arg := range argv {
						if matches := slackUrlRe.FindStringSubmatch(arg); matches != nil {
							argv[i] = matches[1]
						}
					}

					if msg.Channel[0] != 'D' {
						if len(argv) < 1 || argv[0] != "<@"+t.botToken.BotId+">" {
							log.Debugf("Skipping non-direct message %q", msg.Text)
							return
						}
						argv = argv[1:]
					}

					log.Debugf("Received MessageEvent: <%s> %s", msg.User, msg.Text)

					u, err := user.New(db.Instance, user.Id(msg.User), user.Name(msg.Username))
					if err != nil {
						log.Errorf("unable to publish received message; cannot obtain/create user %q: %s", msg.User, err)
						continue
					}

					if u == nil {
						log.Errorf("nil obtaining/creating user %q, but no error?!", msg.User)
						continue
					}

					t.hub.Publish(&hub.Command{
						From:    fmt.Sprintf("internal:send:slack:%s:%s:%s", t.Info.ID, msg.Channel, msg.User),
						Type:    hub.CommandType("user:" + argv[0]),
						Payload: argv[1:],
						User:    u,
						ContextId: t.Info.ID + "-" + msg.Channel,
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

func (t *Team) Send(h *hub.Hub, c *hub.Command) {
	comps := strings.Split(string(c.Type), ":")
	if len(comps) < 5 {
		log.Errorf("%s: received but cannot process command %s", t.Info.ID, c.Type)
		return
	}

	channel := comps[4]

	switch msg := c.Payload.(type) {
	case string:
		_, _, err := t.botClient.PostMessage(
			channel,
			msg,
			slack.PostMessageParameters{
				Text:     msg,
				Username: "mapbot",
				AsUser:   true,
			})
		if err != nil {
			log.Errorf("%s: error posting message %q to channel %q: %s", t.Info.ID, msg, comps[4], err)
		}
	case *tabula.Tabula:
		repErr := func(ctx string, err error) {
			log.Errorf("%s: error %s image %q: %s", t.Info.ID, ctx, msg.Name, err)
			t.Send(h, c.WithPayload(fmt.Sprintf("error %s map %q: %s", ctx, msg.Name, err)))
		}
		img, err := msg.Render()
		if err != nil {
			repErr("rendering", err)
			return
		}

		buf, err := ioutil.TempFile("", "")
		if err != nil {
			repErr("opeaning tmpfile for", err)
			return
		}
		defer buf.Close()

		err = png.Encode(buf, img)
		if err != nil {
			repErr("encoding", err)
			return
		}

		_, err = t.botClient.UploadFile(
			slack.FileUploadParameters{
				Filetype: mime.TypeByExtension(".png"),
				Channels: []string{channel},
				File:     buf.Name(),
			})
		if err != nil {
			repErr("uploading", err)
			return
		}

	}
}

type CommandContext struct {
	UserId  string
	Team    *Team
	Channel string
}

type Team struct {
	Info      *slack.TeamInfo
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
