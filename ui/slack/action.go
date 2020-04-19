package slack

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pdbogen/mapbot/common/blobserv"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/hub"
	mbContext "github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"github.com/pdbogen/mapbot/model/user"
	"github.com/pdbogen/mapbot/model/workflow"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"image"
	"image/png"
	"net/http"
	"strings"
)

type ActionValue struct {
	WorkflowName string
	Choice       string
}

// Json returns the workflow/choice pair encoded as JSON, which really shouldn't
// ever fail.
func (a ActionValue) Json() string {
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Id returns a string that should (but _could_ not be) unique to this specific
// workflow/choice pair.
func (a ActionValue) Id() string {
	j := a.Json()
	if len(j) <= 255 {
		return j
	}

	sum := sha256.Sum256([]byte(j))
	return hex.EncodeToString(sum[:])
}

func writeResponse(rw http.ResponseWriter, msg string) {
	rw.Header().Add("content-type", "application/json")
	rw.WriteHeader(http.StatusOK)
	body, err := json.Marshal(slack.Msg{
		Text:            msg,
		ReplaceOriginal: true,
	})
	if err != nil {
		log.Errorf("marshalling JSON: %s", err)
		rw.Write([]byte(`{"text": "an error occurred"}`))
		return
	}
	rw.Write(body)
}

// upon receiving an action, we need to pass it to the corresponding workflow
// with the appropriate state name, opaque data, and choice. thus the action's
// ID will need to let us obtain the workflow name, state name, and opaque
// data. the choice will come from the action callback itself. the response
// func may return an error, which we need to send to the user. if the response
// doesn't report an error, we'll call the challenge for the new state; which
// will give back a WorkflowMessage.
func (s *SlackUi) Action(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	if err := req.ParseForm(); err != nil {
		writeResponse(rw, "request could not be parsed")
		log.Errorf("error parsing form: %s", err)
		return
	}

	payloads, ok := req.Form["payload"]
	if !ok || len(payloads) == 0 {
		writeResponse(rw, "request had no payload")
		log.Errorf("no payloads in request")
		return
	}
	var payload *slack.InteractionCallback

	if err := json.Unmarshal([]byte(payloads[0]), &payload); err != nil || payload == nil {
		writeResponse(rw, "error parsing payload")
		log.Errorf("error unmarshalling JSON payload: %s", err)
		return
	}

	if s.verificationToken != payload.Token {
		writeResponse(rw, "forbidden")
		log.Errorf("received token %q that does not match slack verification token", payload.Token)
		return
	}

	var team *Team
	s.TeamsMu.RLock()
	for _, t := range s.Teams {
		if t.Info.ID == payload.Team.ID {
			team = t
		}
	}
	s.TeamsMu.RUnlock()

	if team == nil {
		writeResponse(rw, "your team is not recognized. mapbot may need to be reinstalled.")
		log.Errorf("team %q received in action not found", payload.Team.ID)
		return
	}
	team.Action(payload, rw, req)
}

func (t *Team) Action(payload *slack.InteractionCallback, rw http.ResponseWriter, req *http.Request) {
	var log = log.WithFields(logrus.Fields{
		"team":         t.Info.ID,
		"user":         payload.User.ID,
		"payload_type": payload.Type,
		"n_actions":    len(payload.ActionCallback.BlockActions),
	})

	userObj, err := user.Get(db.Instance, types.UserId(payload.User.ID))
	if err != nil {
		writeResponse(rw, "could not retrieve user")
		log.Error("cannot retrieve user")
		return
	}

	if payload.Type != "block_actions" || len(payload.ActionCallback.BlockActions) == 0 {
		log.Error("unhandled or empty payload type")
		writeResponse(rw, "cannot handle action callback type "+string(payload.Type))
		return

	}

	actionCallbackValue := payload.ActionCallback.BlockActions[0].Value
	var av ActionValue
	if err := json.Unmarshal([]byte(actionCallbackValue), &av); err != nil {
		writeResponse(rw, "could not parse action callback")
		log.Errorf("could not parse action callback value %q: %v",
			actionCallbackValue, err)
		return
	}

	log.WithField("value", av).Trace("dispatching")

	t.hub.Publish(&hub.Command{
		User:    userObj,
		From:    fmt.Sprintf("internal:updateAction:slack:%s:%s:%s", t.Info.ID, payload.Channel.ID, payload.ResponseURL),
		Context: t.Context(payload.Channel.ID),
		Payload: []string{"action", av.WorkflowName, av.Choice},
		Type:    "user:workflow",
	})
}

func (t *Team) updateAction(h *hub.Hub, c *hub.Command) {
	var log = log.WithFields(map[string]interface{}{
		"team":         t.Info.ID,
		"user":         c.User.Id,
		"from":         c.From,
		"type":         c.Type,
		"payload_type": fmt.Sprintf("%T", c.Payload),
	})

	comps := strings.Split(string(c.Type), ":")
	if len(comps) < 6 {
		log.Error("invalid type for updateAction")
		return
	}
	responseUrl := strings.Join(comps[5:], ":")

	var opts []slack.MsgOption
	switch msg := c.Payload.(type) {
	case *workflow.WorkflowMessage:
		log.Trace("rendering workflow message")
		opts = t.renderWorkflowMessage(c.Context, msg)
	case string:
		log.Trace("rendering simple message")
		opts = []slack.MsgOption{slack.MsgOptionText(msg, false)}
	default:
		log.Error("invalid payload for updateAction")
		return
	}

	opts = append(opts,
		slack.MsgOptionReplaceOriginal(responseUrl),
	)

	_, _, _, err := t.botClient.SendMessage(comps[4], opts...)
	if err != nil {
		log.Errorf("POSTing action update: %v", err)
	}
}

func (t *Team) renderWorkflowMessage(ctx mbContext.Context, msg *workflow.WorkflowMessage) []slack.MsgOption {
	if msg.Text == "" {
		return nil
	}

	var blocks []slack.Block
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(mrkdwn, msg.Text, false, false),
		nil,
		nil,
	))

	if msg.Choices != nil {
		msg.ChoiceSets = append(msg.ChoiceSets, msg.Choices)
	}

	for _, choices := range msg.ChoiceSets {
		var btns []slack.BlockElement
		for _, choice := range choices {
			av := ActionValue{msg.Id(), choice}
			btns = append(btns, slack.NewButtonBlockElement(
				av.Id(),
				av.Json(),
				slack.NewTextBlockObject("plain_text", choice, false, false),
			))
		}
		blocks = append(blocks, slack.NewActionBlock("", btns...))
	}

	if msg.Image != nil {
		b, err := imageBlock(msg.Image)
		if err != nil {
			log.Errorf("failed to attach non-nil image: %v", err)
			blocks = append(blocks,
				slack.NewTextBlockObject(mrkdwn, "could not render image", false, false))
		} else {
			blocks = append(blocks, b)
		}
	}

	if msg.TabulaId != nil {
		var log = log.WithField("tabula", msg.TabulaId)
		tab, err := tabula.Load(db.Instance, *msg.TabulaId)
		var img image.Image
		if err == nil {
			img, err = tab.Render(ctx, nil)
		}
		var b slack.Block
		if err == nil {
			b, err = imageBlock(img)
		}
		if err == nil {
			blocks = append(blocks, b)
		}
		if err != nil {
			log.WithError(err).Error("tabula render failed")
			blocks = append(blocks, slack.NewTextBlockObject(mrkdwn, "cannot render map", false, false))
		}
	}

	return []slack.MsgOption{slack.MsgOptionBlocks(blocks...)}
}

func imageBlock(i image.Image) (slack.Block, error) {
	log.Debugf("attaching image to workflow message")

	imgData := &bytes.Buffer{}
	if err := png.Encode(imgData, i); err != nil {
		return nil, fmt.Errorf("encoding png: %v", err)
	}

	url, err := blobserv.Upload(imgData.Bytes())
	if err != nil {
		return nil, fmt.Errorf("uploading to blobserv: %v", err)
	}
	log.Debugf("image uploaded with URL %q", url)

	return slack.NewImageBlock(
		url,
		"here there be dragons",
		"",
		slack.NewTextBlockObject("plain_text", "map section", false, false),
	), nil
}
