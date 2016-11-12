package hub

import (
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/ryanuber/go-glob"
	"fmt"
)

var log = mbLog.Log

type Hub struct {
	Subscribers map[CommandType][]Subscriber
}

func (h *Hub) Subscribe(c CommandType, s Subscriber) {
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
	log.Debugf("publish: %s->%s: %v", c.From, string(c.Type), c.Payload)
	if h.Subscribers == nil {
		h.Subscribers = map[CommandType][]Subscriber{}
	}

	found := false
	for cmd, subs := range h.Subscribers {
		if glob.Glob(string(cmd), string(c.Type)) {
			for _, sub := range subs {
				found = true
				go sub(h, c)
			}
		}
	}

	if !found && c.From != "" {
		h.Publish(&Command{
			Type: CommandType(c.From),
			Payload: fmt.Sprintf("No handler for command '%s'", c.Type),
		})
	}
}

type Command struct {
	Type    CommandType
	From    string
	Payload interface{}
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
