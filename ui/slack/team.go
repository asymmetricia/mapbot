package slack

import (
	"fmt"
	"github.com/nlopes/slack"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/common/db/anydb"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"github.com/pdbogen/mapbot/model/user"
	"github.com/pdbogen/mapbot/model/workflow"
	slackContext "github.com/pdbogen/mapbot/ui/slack/context"
	"image"
	"image/png"
	"io/ioutil"
	"mime"
	"os"
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
	s.botHub.Subscribe(
		hub.CommandType(fmt.Sprintf("internal:updateAction:slack:%s:*", team.Info.ID)),
		team.updateAction,
	)

	go team.updateEmoji()

	return team.Save(s.db)
}

func (t *Team) updateEmoji() {
	emoji, err := t.client.GetEmoji()
	if err != nil {
		log.Errorf("error getting emoji for %s: %s", t.Info.ID, err)
		return
	}
	if t.EmojiCache == nil {
		log.Debugf("initializing emoji cache for team %s", t)
		t.EmojiCache = map[string]image.Image{}
	} else {
		for name := range t.EmojiCache {
			if newUrl, ok := emoji[name]; !ok || t.Emoji[name] != newUrl {
				delete(t.EmojiCache, name)
			}
		}
	}
	log.Infof("Team %s has %d emoji", t.Info.ID, len(emoji))
	t.Emoji = emoji
}

func (t *Team) Save(db anydb.AnyDb) error {
	if t.botToken == nil {
		t.botToken = &BotToken{}
	}
	var query string
	switch dia := db.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO slack_teams " +
			"(token, bot_id, bot_token) " +
			"VALUES ($1, $2, $3) " +
			"ON CONFLICT (token) DO UPDATE SET bot_id=$2, bot_token=$3"
	case "sqlite3":
		query = "REPLACE INTO slack_teams (token, bot_id, bot_token) VALUES ($1, $2, $3)"
	default:
		return fmt.Errorf("no Team.Save query for SQL dialect %s", dia)
	}
	_, err := db.Exec(query,
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
			case "user_typing":
				go func() {
					if msg, ok := event.Data.(*slack.UserTypingEvent); ok {
						_, err := user.Get(db.Instance, user.Id(msg.User))
						if err != nil {
							log.Errorf("user preload in response to typing failed: %s", err)
							return
						}
					}
				}()
			case "emoji_changed":
				go t.updateEmoji()
			case "message":
				go func() {
					if msg, ok := event.Data.(*slack.MessageEvent); ok {
						if msg.User == t.botToken.BotId || msg.User == "" {
							return
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

						u, err := user.Get(db.Instance, user.Id(msg.User))
						if err != nil {
							log.Errorf("unable to publish received message; cannot obtain/create user %q: %s", msg.User, err)
							return
						}

						if u == nil {
							log.Errorf("nil obtaining/creating user %q, but no error?!", msg.User)
							return
						}

						t.hub.Publish(&hub.Command{
							From:    fmt.Sprintf("internal:send:slack:%s:%s:%s", t.Info.ID, msg.Channel, msg.User),
							Type:    hub.CommandType("user:" + argv[0]),
							Payload: argv[1:],
							User:    u,
							Context: t.Context(msg.Channel),
						})
					} else {
						log.Warningf("Received message, but type was %s", reflect.TypeOf(event.Data))
					}
				}()
			case "latency_report":
			case "reconnect_url":
			case "presence_change":
			default:
				log.Debugf("unhandled message type %q", event.Type)
			}
		}
	}
}

func (t *Team) renderWorkflowMessage(msg *workflow.WorkflowMessage) slack.PostMessageParameters {
	params := slack.PostMessageParameters{
		Text: msg.Text,
	}

	params.Attachments = []slack.Attachment{}
	if msg.Choices != nil {
		attachment := slack.Attachment{
			CallbackID: msg.Id(),
			Actions:    make([]slack.AttachmentAction, len(msg.Choices)),
			Fallback:   "Your client does not support actions. :cry:",
		}
		for i, choice := range msg.Choices {
			log.Debugf("adding choice %q", choice)
			attachment.Actions[i] = slack.AttachmentAction{
				Name:  "choice",
				Text:  choice,
				Value: choice,
				Type:  "button",
			}
		}
		params.Attachments = append(params.Attachments, attachment)
	}

	if msg.Image != nil {
		url, err := t.uploadImage(msg.Image, nil)
		if err != nil {
			log.Errorf("uploading image in workflow message: %s", err)
		}
		params.Attachments = append(params.Attachments, slack.Attachment{
			ImageURL: url,
		})

	}

	return params
}

func (t *Team) sendWorkflowMessage(h *hub.Hub, c *hub.Command, msg *workflow.WorkflowMessage) {
	comps := strings.Split(string(c.Type), ":")
	if len(comps) < 5 {
		log.Errorf("%s: received but cannot process command %s", t.Info.ID, c.Type)
		return
	}

	channel := comps[4]

	_, _, err := t.botClient.PostMessage(
		channel,
		msg.Text,
		t.renderWorkflowMessage(msg),
	)

	if err != nil {
		log.Errorf("%s: error posting workflow message to channel %q: %s", t.Info.ID, comps[4], err)
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
	case *workflow.WorkflowMessage:
		t.sendWorkflowMessage(h, c, msg)
	case *tabula.Tabula:
		repErr := func(ctx string, err error) {
			log.Errorf("%s: error %s image %q: %s", t.Info.ID, ctx, msg.Name, err)
			t.Send(h, c.WithPayload(fmt.Sprintf("error %s map %q: %s", ctx, msg.Name, err)))
		}
		img, err := msg.Render(t.Context(channel), func(msg string) { t.Send(h, c.WithPayload(msg)) })
		if err != nil {
			repErr("rendering", err)
			return
		}

		if _, err := t.uploadImage(img, []string{channel}); err != nil {
			repErr("uploading", err)
			return
		}
	}
}

func (t *Team) uploadImage(img image.Image, channels []string) (string, error) {
	repErr := func(s string, e error) error { return fmt.Errorf("%s: %s", s, e) }
	buf, err := ioutil.TempFile("", "")
	if err != nil {
		return "", repErr("opeaning tmpfile for", err)
	}

	err = png.Encode(buf, img)
	buf.Close()
	if err != nil {
		return "", repErr("encoding", err)
	}

	upload, err := t.botClient.UploadFile(
		slack.FileUploadParameters{
			Filetype: mime.TypeByExtension(".png"),
			Channels: channels,
			File:     buf.Name(),
		})
	os.Remove(buf.Name())
	if err != nil {
		return "", repErr("uploading", err)
	}
	return upload.URLPrivate, nil
}

func (t *Team) Context(SubTeamId string) context.Context {
	ret := &slackContext.SlackContext{
		Emoji:      t.Emoji,
		EmojiCache: t.EmojiCache,
	}
	ret.ContextId = types.ContextId(t.Info.ID + "-" + SubTeamId)
	if err := ret.Load(); err != nil {
		log.Errorf("failed while hydrating context %s from the db: %s", ret.ContextId, err)
	}

	return ret
}

type Team struct {
	Info       *slack.TeamInfo
	Channels   []Channel
	Quit       <-chan bool
	token      string
	client     *slack.Client
	botClient  *slack.Client
	botToken   *BotToken
	rtm        *slack.RTM
	hub        *hub.Hub
	Emoji      map[string]string
	EmojiCache map[string]image.Image
}

type Channel struct {
	ActiveTabula *tabula.Tabula
}
