package hub

import (
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/user"
	"github.com/ryanuber/go-glob"
	"strings"
)

var log = mbLog.Log

type Hub struct {
	Subscribers map[CommandType][]Subscriber
}

func (h *Hub) Subscribe(c CommandType, s Subscriber) {
  c = c.Canonical()
	log.Debugf("subscribe: %s", c)
	if h.Subscribers == nil {
		h.Subscribers = map[CommandType][]Subscriber{
			c: []Subscriber{},
		}
	}

	if subs, ok := h.Subscribers[c]; ok {
		h.Subscribers[c] = append(subs, s)
	} else {
		h.Subscribers[c] = []Subscriber{s}
	}
}

// Publish searches publishers for a subscriber to the given command's type, and executes the subscriber in a goroutine.
func (h *Hub) Publish(c *Command) {
	log.Debugf("publish: %s->%s (%s): %v", c.From, string(c.Type), c.User, c.Payload)
	if h.Subscribers == nil {
		h.Subscribers = map[CommandType][]Subscriber{}
	}

  typ := c.Type.Canonical()
	found := false
	for cmd, subs := range h.Subscribers {
		if glob.Glob(string(cmd), string(typ)) {
			for _, sub := range subs {
				found = true
				go sub(h, c)
			}
		}
	}

	if !found && c.From != "" {
		h.Publish(&Command{
			Type:    CommandType(c.From),
			Payload: fmt.Sprintf("No handler for command '%s'", typ),
			User:    c.User,
			Context: c.Context,
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
		User:    trigger.User,
		Context: trigger.Context,
	})
}

// Command represents a request for the mapbot system to execute some action. It may be a request from outside (usually
// having user:... as a type) or an internal request (such as to generate a response). From should be the command type
// that will "reply"; so for a user command from slack, it would be the slack Command Type that will trigger a response.
// The ContextId should uniquely identify something like a "room," "channel," or "session" for whatever UI model this
// command involves.
type Command struct {
	Type    CommandType
	From    string
	Payload interface{}
	User    *user.User
	Context context.Context
}

// WithType returns a copy of the command with the type replaced by the given type. The payload is not deep-copied.
func (c *Command) WithType(n CommandType) *Command {
	return &Command{
		Type:    n,
		From:    c.From,
		Payload: c.Payload,
		User:    c.User,
		Context: c.Context,
	}
}

func (c *Command) WithPayload(p interface{}) *Command {
	return &Command{
		Type:    c.Type,
		From:    c.From,
		Payload: p,
		User:    c.User,
		Context: c.Context,
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
// form: internal:<module>:<module:specific:path>. `from` typically matters less for internal commands.
//
// Example: internal:send:slack:SOME_TEAM_ID:SOME_CHANNEL_ID
//
// CommandTypes are matched using wildcards; thus a slack team might subscribe to internal:slack:SOME_TEAM_ID:*.
type CommandType string

func (c CommandType) Canonical() CommandType {
	return CommandType(strings.ToLower(string(c)))
}

type Subscriber func(hub *Hub, cmd *Command)
type Responder func(msg string)
