package slack

import (
	"bytes"
	"fmt"
	"github.com/pdbogen/mapbot/common/blobserv"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/common/db/anydb"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"github.com/pdbogen/mapbot/model/user"
	"github.com/pdbogen/mapbot/model/workflow"
	slackContext "github.com/pdbogen/mapbot/ui/slack/context"
	"github.com/slack-go/slack"
	"image"
	"image/png"
	"io/ioutil"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
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

		s.addTeam(token, botToken)
	}
	return nil
}

const (
	initialDelay      = time.Second
	maxDelay          = 300 * time.Second
	jitterDelayFactor = 10 // larger number == smaller jitter
)

func (s *SlackUi) addTeam(token string, botToken *BotToken) {
	go func(token string, botToken *BotToken) {
		delay := time.Duration(0)
		for {
			// no delay first time through, then exponential delay w/ some jitter
			if delay == 0 {
				delay = initialDelay
			} else {
				d := delay + time.Duration(rand.Int63n(int64(delay/jitterDelayFactor))) + 1
				log.Errorf("sleeping for %0.2fs before retry", d.Seconds())
				time.Sleep(d)
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
			}

			log.Infof("Adding team with token %s, %s", token, botToken)
			team := &Team{
				Channels:  []Channel{},
				token:     token,
				client:    slack.New(token),
				botClient: slack.New(botToken.BotToken),
				botToken:  botToken,
				hub:       s.botHub,
			}

			ti, err := team.client.GetTeamInfo()
			if err != nil {
				log.Errorf("obtaining info for token %s: %v", token, err)
				continue
			}
			team.Info = ti
			team.run()

			if err := team.Save(s.db); err != nil {
				log.Errorf("saving team %s to DB: %v", token, err)
				continue
			}

			s.TeamsMu.Lock()
			s.Teams = append(s.Teams, team)
			s.TeamsMu.Unlock()

			s.botHub.Subscribe(
				hub.CommandType(fmt.Sprintf("internal:send:slack:%s:*", team.Info.ID)),
				team.Send,
			)
			s.botHub.Subscribe(
				hub.CommandType(fmt.Sprintf("internal:updateAction:slack:%s:*", team.Info.ID)),
				team.updateAction,
			)

			team.updateEmoji()
			break
		}
	}(token, botToken)
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
						_, err := user.Get(db.Instance, types.UserId(msg.User))
						if err != nil {
							log.Errorf("user preload in response to typing failed: %s", err)
							return
						}
					}
				}()
			case "emoji_changed":
				go t.updateEmoji()
			case "message":
				go t.handleMessage(event)
			case "latency_report":
			case "reconnect_url":
			case "presence_change":
			default:
				log.Debugf("unhandled message type %q", event.Type)
			}
		}
	}
}

func (t *Team) handleMessage(event slack.RTMEvent) {
	msg, ok := event.Data.(*slack.MessageEvent)
	if !ok {
		log.Warningf("Received message, but type was %s, not *slack.MessageEvent",
			reflect.TypeOf(event.Data))
		return
	}
	if msg.User == t.botToken.BotId || msg.User == "" {
		return
	}

	argv := strings.Fields(msg.Text)

	// de-linkify links
	for i, arg := range argv {
		if matches := slackUrlRe.FindStringSubmatch(arg); matches != nil {
			argv[i] = matches[1]
		}
	}

	// the message is for us if it's a DM or if it begins with `@mapbot`
	if msg.Channel[0] != 'D' && (len(argv) < 1 || argv[0] != "<@"+t.botToken.BotId+">") {
		log.Debugf("skipping un-prefixed non-direct message %q", msg.Text)
		return
	}

	// strip any `@mapbot` prefix, even in DMs
	if len(argv) > 0 && argv[0] == "<@"+t.botToken.BotId+">" {
		argv = argv[1:]
	}

	log.Debugf("Received MessageEvent: <%s> %s", msg.User, msg.Text)
	log.Debugf("Message Object: %+v", msg)

	u, err := user.Get(db.Instance, types.UserId(msg.User))
	if err != nil {
		log.Errorf("unable to publish received message; cannot obtain/create user %q: %s", msg.User, err)
		return
	}

	if u == nil {
		log.Errorf("nil obtaining/creating user %q, but no error?!", msg.User)
		return
	}

	// Accept maps uploaded via DM
	if msg.Upload && len(msg.Files) > 0 && msg.Channel[0] == 'D' {
		t.handleUpload(u, msg)
		return
	}

	cmd := argv[0]
	var args []string
	if len(argv) >= 2 {
		args = argv[1:]
	}

	t.hub.Publish(&hub.Command{
		From:    fmt.Sprintf("internal:send:slack:%s:%s:%s", t.Info.ID, msg.Channel, msg.User),
		Type:    hub.CommandType("user:" + cmd),
		Payload: args,
		User:    u,
		Context: t.Context(msg.Channel),
	})
}

func attachImage(a *[]slack.Attachment, i image.Image) {
	if i == nil {
		return
	}
	log.Debugf("attaching image to workflow message")

	imgData := &bytes.Buffer{}
	if err := png.Encode(imgData, i); err != nil {
		log.Errorf("encoding image as png: %s", err)
		return
	}
	url, err := blobserv.Upload(imgData.Bytes())
	if err != nil {
		log.Errorf("uploading image in workflow message: %s", err)
		return
	}
	log.Debugf("image uploaded with URL %q", url)

	*a = append(*a, slack.Attachment{ImageURL: url})
}

func (t *Team) renderWorkflowMessage(msg *workflow.WorkflowMessage) []slack.MsgOption {
	if msg.Text == "" {
		return nil
	}

	ret := []slack.MsgOption{slack.MsgOptionText(msg.Text, false)}

	var attachments []slack.Attachment

	if msg.ChoiceSets == nil {
		msg.ChoiceSets = [][]string{}
	}

	if msg.Choices != nil {
		msg.ChoiceSets = append(msg.ChoiceSets, msg.Choices)
	}

	for _, choices := range msg.ChoiceSets {
		attachment := slack.Attachment{
			CallbackID: msg.Id(),
			Actions:    make([]slack.AttachmentAction, len(choices)),
			Fallback:   "Your client does not support actions. :cry:",
		}
		for i, choice := range choices {
			log.Debugf("adding choice %q", choice)
			attachment.Actions[i] = slack.AttachmentAction{
				Name:  "choice",
				Text:  choice,
				Value: choice,
				Type:  "button",
			}
		}
		attachments = append(attachments, attachment)
	}

	attachImage(&attachments, msg.Image)

	return append(ret, slack.MsgOptionAttachments(attachments...))
}

func (t *Team) sendWorkflowMessage(h *hub.Hub, c *hub.Command, msg *workflow.WorkflowMessage) {
	comps := strings.Split(string(c.Type), ":")
	if len(comps) < 5 {
		log.Errorf("%s: received but cannot process command %s", t.Info.ID, c.Type)
		return
	}

	channel := comps[4]

	opts := t.renderWorkflowMessage(msg)
	if len(opts) > 0 {
		_, _, err := t.botClient.PostMessage(channel, opts...)

		if err != nil {
			log.Errorf("%s: error posting workflow message to channel %q: %s", t.Info.ID, comps[4], err)
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
			slack.MsgOptionText(msg, false),
			slack.MsgOptionUsername("mapbot"),
			slack.MsgOptionAsUser(true),
		)
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
		if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
			repErr(
				"rendering",
				fmt.Errorf("no pixels (dx=%d, dy=%d)", img.Bounds().Dx(), img.Bounds().Dy()),
			)
			return
		}

		if _, err := t.uploadImage(msg.Note, img, []string{channel}); err != nil {
			repErr("uploading", err)
			return
		}
	}
}

func (t *Team) uploadImage(title string, img image.Image, channels []string) (string, error) {
	repErr := func(s string, e error) error { return fmt.Errorf("%s: %s", s, e) }
	buf, err := ioutil.TempFile("", "")
	if err != nil {
		return "", repErr("opening tmpfile for", err)
	}

	err = png.Encode(buf, img)
	buf.Close()
	if err != nil {
		return "", repErr("encoding", err)
	}

	upload, err := t.botClient.UploadFile(
		slack.FileUploadParameters{
			Title:    title,
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

func (t *Team) Context(ChannelId string) context.Context {
	ret := &slackContext.SlackContext{
		Emoji:      t.Emoji,
		EmojiCache: t.EmojiCache,
	}
	ret.ContextId = types.ContextId(t.Info.ID + "-" + ChannelId)
	if err := ret.Load(db.Instance); err != nil {
		log.Errorf("failed while hydrating context %s from the db: %s", ret.ContextId, err)
	}

	return ret
}

func (t *Team) handleUpload(user *user.User, msg *slack.MessageEvent) {
	file := msg.Files[0]

	req, err := http.NewRequest("GET", file.URLPrivateDownload, nil)
	if err != nil {
		log.Errorf("building new request object: %v", err)
		return
	}
	req.Header.Add("Authorization", "Bearer "+t.botToken.BotToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("sending request for uploaded file %v: %v", file.URLPrivateDownload, err)
		return
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("reading data from remote server for %v: %v", file.URLPrivateDownload, err)
		return
	}

	nonalnum := regexp.MustCompile(`[^a-z0-9]`)
	file.Name = nonalnum.ReplaceAllString(strings.ToLower(file.Name), "-")
	t.hub.Publish(&hub.Command{
		From:    fmt.Sprintf("internal:send:slack:%s:%s:%s", t.Info.ID, msg.Channel, user.Id),
		Type:    "user:map",
		Payload: []string{"add", "@" + file.Name, "raw"},
		User:    user,
		Context: t.Context("@mapbot"),
		Data:    data,
	})
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
