package hub

import (
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/user"
	"github.com/ryanuber/go-glob"
)

var log = mbLog.Log

type Hub struct {
	subscribers map[CommandType][]Subscriber
}

func (h *Hub) Subscribe(c CommandType, s Subscriber) {
	log.Debugf("subscribe: %s", c)
	if h.subscribers == nil {
		h.subscribers = map[CommandType][]Subscriber{
			c: []Subscriber{},
		}
	}

	if subs, ok := h.subscribers[c]; ok {
		h.subscribers[c] = append(subs, s)
	} else {
		h.subscribers[c] = []Subscriber{s}
	}
}

// Publish searches publishers for a subscriber to the given command's type, and executes the subscriber in a goroutine.
func (h *Hub) Publish(c *Command) {
	log.Debugf("publish: %s->%s (%s): %v", c.From, string(c.Type), c.User, c.Payload)
	if h.subscribers == nil {
		h.subscribers = map[CommandType][]Subscriber{}
	}

	found := false
	for cmd, subs := range h.subscribers {
		if glob.Glob(string(cmd), string(c.Type)) {
			for _, sub := range subs {
				found = true
				go sub(h, c)
			}
		}
	}

	if !found && c.From != "" {
		h.Publish(&Command{
			Type:    CommandType(c.From),
			Payload: fmt.Sprintf("No handler for command '%s'", c.Type),
			TeamId:  c.TeamId,
			User:    c.User,
		})
	}
}

func (h *Hub) Error(trigger *Command, message string) {
	h.Reply(trigger, message)
}

func (h *Hub) Reply(trigger *Command, message string) {
	if trigger.From == "" {
		log.Errorf("trigger command has no `from`; cannot publish message %q", message)
		return
	}

	h.Publish(&Command{
		Type:    CommandType(trigger.From),
		Payload: message,
		TeamId:  trigger.TeamId,
		User:    trigger.User,
	})
}

type Command struct {
	Type    CommandType
	From    string
	Payload interface{}
	TeamId  string
	User    *user.User
}

// WithType returns a copy of the command with the type replaced by the given type. The payload is not deep-copied.
func (c *Command) WithType(n CommandType) *Command {
	return &Command{
		Type:    n,
		From:    c.From,
		Payload: c.Payload,
		TeamId:  c.TeamId,
		User:    c.User,
	}
}

func (c *Command) WithPayload(p interface{}) *Command {
	return &Command{
		Type:    c.Type,
		From:    c.From,
		Payload: p,
		TeamId:  c.TeamId,
		User:    c.User,
	}
}

// CommandType is a one of two major structures- either a user command or an internal command.
//
// User commands originate from users; they are generally of the format "user:<command>". The `from` should be an
// internal command type (see below) used to respond to the user.
//
// Example: user:howdy
//
// Internal commands originate from some internal module, perhaps in response to a user command. They should be of the
// form: internal:<module>:<module:specific:path>
//
// Example: internal:send:slack:SOME_TEAM_ID:SOME_CHANNEL_ID
//
// CommandTypes are matched using wildcards; thus a slack team might subscribe to internal:slack:SOME_TEAM_ID:*.
type CommandType string

type Subscriber func(hub *Hub, cmd *Command)
type Responder func(msg string)
