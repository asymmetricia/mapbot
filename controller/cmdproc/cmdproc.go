package cmdproc

import (
	"fmt"
	"github.com/pdbogen/mapbot/hub"
	"strings"
)

type Subcommand struct {
	Args  string
	Usage string
	Cmd   hub.Subscriber
}

type CommandProcessor struct {
	Command  string
	Commands map[string]Subcommand
	Comment  string
}

func (c *CommandProcessor) Help(h *hub.Hub, cmd *hub.Command) {
	help := fmt.Sprintf("Select from the following %s commands:", cmd.Type)
	for subCommand, sc := range c.Commands {
		help += "\n`" + c.Command + " " + subCommand
		if sc.Args != "" {
			help += " " + sc.Args
		}
		help += "` - " + sc.Usage
	}
	if c.Comment != "" {
		help += "\n" + c.Comment
	}
	h.Publish(&hub.Command{
		Type:    hub.CommandType(cmd.From),
		Payload: help,
		User:    cmd.User,
	})
}

func (c *CommandProcessor) Route(h *hub.Hub, cmd *hub.Command) {
	if c.Commands == nil {
		c.Commands = map[string]Subcommand{}
	}

	if args, ok := cmd.Payload.([]string); ok && len(args) > 0 {
		cmdName := strings.ToLower(args[0])
		if cmdName == "help" {
			c.Help(h, cmd)
		} else {
			if s, ok := c.Commands[args[0]]; ok {
				new_type := hub.CommandType(fmt.Sprintf("%s:%s", cmd.Type, args[0]))
				new_payload := args[1:]
				s.Cmd(h, cmd.WithType(new_type).WithPayload(new_payload))
			} else {
				h.Error(cmd, fmt.Sprintf("Sub-command %q not found; try 'help'", args[0]))
			}
		}
	} else {
		h.Error(cmd, "No sub-command specified. Try 'help'")
	}
}
