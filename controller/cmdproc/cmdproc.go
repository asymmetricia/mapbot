package cmdproc

import (
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	"strings"
)

var log = mbLog.Log

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

	args, ok := cmd.Payload.([]string)
	if !ok {
		h.Error(cmd, "No sub-command specified. Try 'help'")
		return
	}

	log.Debugf("command processor routing cmd %v w/ %d bytes data", args, len(cmd.Data))

	cmdName := strings.ToLower(args[0])
	if cmdName == "help" {
		c.Help(h, cmd)
		return
	}

	subC, ok := c.Commands[cmdName]
	if !ok {
		h.Error(cmd, fmt.Sprintf("Sub-command %q not found; try 'help'", args[0]))
		return
	}

	newType := hub.CommandType(fmt.Sprintf("%s:%s", cmd.Type, args[0]))
	subC.Cmd(h, cmd.WithType(newType).WithPayload(args[1:]))
}
