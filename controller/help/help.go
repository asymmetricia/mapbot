package help

import (
	"fmt"
	"github.com/pdbogen/mapbot/hub"
	"strings"
)

func Register(h *hub.Hub) {
	h.Subscribe("user:help", help)
}

func helpAll(h *hub.Hub, cmd *hub.Command) {
	handlers := []string{}
	for k, _ := range h.Subscribers {
		if strings.HasPrefix(string(k), "user:") {
			parts := strings.Split(string(k), ":")
			if parts[1] != "help" && parts[1] != "howdy" {
				handlers = append(handlers, parts[1])
			}
		}
	}

	response := "The following top-level commands are registered:\n" +
		strings.Join(handlers, "\n") +
		"\nMost commands respond to `<command> help`"
	h.Reply(cmd, response)
}

func helpSingle(h *hub.Hub, cmd *hub.Command) {
	args := cmd.Payload.([]string)
	for k, _ := range h.Subscribers {
		if strings.HasPrefix(string(k), "user:"+args[0]) {
			h.Publish(cmd.WithType(k).WithPayload([]string{"help"}))
			return
		}
	}
	h.Error(cmd, fmt.Sprintf("no top-level command %s found", args[0]))
}

func help(h *hub.Hub, cmd *hub.Command) {
	if args, ok := cmd.Payload.([]string); ok {
		if len(args) == 0 {
			helpAll(h, cmd)
		} else {
			helpSingle(h, cmd)
		}
	}
}
