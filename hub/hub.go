package hub

import (
	mbLog "github.com/pdbogen/mapbot/common/log"
)

var log = mbLog.Log

type Hub struct {
	Subscribers map[CommandType][]Subscriber
}

func (h *Hub) Subscribe(c CommandType, s Subscriber) {
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

func (h *Hub) Publish(c *Command) {
	if h.Subscribers == nil {
		h.Subscribers = map[CommandType][]Subscriber{}
	}

	if subs, ok := h.Subscribers[c.Type]; ok && len(subs)>0 {
		for _, sub := range subs {
			sub(c)
		}
	} else {
		log.Debugf("No subscribers for CommandType{%s}", c.Type)
	}
}

type Command struct {
	Type    CommandType
	Context interface{}
	Payload interface{}
}

type CommandType string

type Subscriber func(cmd *Command)
