package mask

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/controller/cmdproc"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/mask"
	"github.com/pdbogen/mapbot/model/tabula"
	"reflect"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:mask", processor.Route)
}

var processor *cmdproc.CommandProcessor

func init() {
	processor = &cmdproc.CommandProcessor{
		Commands: map[string]cmdproc.Subcommand{
			"add": cmdproc.Subcommand{"<map-name> <mask-name>", "add a new mask to one of your maps", cmdAdd},
			"up":  cmdproc.Subcommand{"<map-name> <mask-name>", "moves the indicated mask up, so that it will be applied earlier", cmdUp},
			//"show": cmdproc.Subcommand{"<name>", "show a gridded map", cmdShow},
			//"set":  cmdproc.Subcommand{"<name> {offsetX|offsetY|dpi|gridColor} <value>", "set a property of an existing map", cmdSet},
			//"list": cmdproc.Subcommand{"", "list your maps", cmdList},
		},
	}
}

func cmdUp(h *hub.Hub, c *hub.Command) {
}

func argsFromCommand(c *hub.Command) ([]string, error) {
	if args, ok := c.Payload.([]string); ok {
		return args, nil
	}
	return nil, fmt.Errorf("command payload was %s, not array-of-strings", reflect.TypeOf(c.Payload))
}

func cmdAdd(h *hub.Hub, c *hub.Command) {
	args, err := argsFromCommand(c)
	if err != nil {
		h.Error(c, err.Error())
		return
	}

	if len(args) != 2 {
		h.Error(c, fmt.Sprintf("usage: mask add %s", processor.Commands["add"].Usage))
		return
	}

	t, ok := c.User.TabulaByName(tabula.Name(args[0]))
	if !ok {
		h.Error(c, fmt.Sprintf("you have no map %q", args[0]))
		return
	}

	if _, ok := t.Masks[args[1]]; ok {
		h.Error(c, fmt.Sprintf("your map %q already has a mask named %q", args[0], args[1]))
		return
	}

	m := &mask.Mask{Name: args[1]}
	t.Masks[args[1]] = m
	if err := m.Save(db.Instance, int64(*t.Id)); err != nil {
		h.Error(c, fmt.Sprintf("Error adding map: %s", err))
		return
	}

	h.Reply(c, fmt.Sprintf("mask %q added; use 'mask set' to configure", args[1]))
}
