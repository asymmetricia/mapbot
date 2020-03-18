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
	h.HubMu.RLock()
	for k := range h.Subscribers {
		if strings.HasPrefix(string(k), "user:") {
			parts := strings.Split(string(k), ":")
			if parts[1] != "help" && parts[1] != "howdy" {
				handlers = append(handlers, fmt.Sprintf("`%s`", parts[1]))
			}
		}
	}
	h.HubMu.RUnlock()

	response := "The following top-level commands are registered:\n" +
		strings.Join(handlers, "\n") +
		"\nMost commands respond to `<command> help`"
	h.Reply(cmd, response)
}

func helpSingle(h *hub.Hub, cmd *hub.Command) {
	var helpCmd *hub.Command
	args := cmd.Payload.([]string)
	h.HubMu.RLock()
	for cmdType := range h.Subscribers {
		if strings.HasPrefix(string(cmdType), "user:"+args[0]) {
			helpCmd = cmd.WithType(cmdType).WithPayload([]string{"help"})
		}
	}
	h.HubMu.RUnlock()

	if helpCmd == nil {
		h.Error(cmd, fmt.Sprintf("no top-level command %s found", args[0]))
		return
	}
	h.Publish(helpCmd)
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
