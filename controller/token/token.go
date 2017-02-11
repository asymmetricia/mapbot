package token

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/conv"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/controller/cmdproc"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/context"
	"image"
	"reflect"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:token", processor.Route)
}

var processor *cmdproc.CommandProcessor

func init() {
	processor = &cmdproc.CommandProcessor{
		Command: "token",
		Commands: map[string]cmdproc.Subcommand{
			"add":  cmdproc.Subcommand{"<name> <X> <Y>", "add a token (or change its location) to the currently selected map (see `map select`). Token names should be very short.", cmdAdd},
			"show": cmdproc.Subcommand{"<name> <X> <Y>", "synonym for add", cmdAdd},
			"move": cmdproc.Subcommand{"<name> <X> <Y>", "synonym for add", cmdAdd},
			//"show":   cmdproc.Subcommand{"<name>", "show a gridded map", cmdShow},
			//"set":    cmdproc.Subcommand{"<name> {offsetX|offsetY|dpi|gridColor} <value>", "set a property of an existing map", cmdSet},
			//"list":   cmdproc.Subcommand{"", "list your maps", cmdList},
			//"select": cmdproc.Subcommand{"<name>", "selects the map active in this channel", cmdSelect},
		},
	}
}

func cmdAdd(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) != 3 {
		h.Error(c, "usage: token add "+processor.Commands["add"].Args)
		return
	}

	ctx, err := context.Load(db.Instance, c.ContextId)
	if err != nil {
		log.Errorf("Error loading context %q: %s", c.ContextId, err)
		h.Error(c, "error loading context")
		return
	}

	if ctx.ActiveTabula == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	name := args[0]

	coord, err := conv.CoordsToPoint(args[1], args[2])
	if err != nil {
		h.Error(c, fmt.Sprintf("Invalid coordinates: %s", err))
		return
	}

	if ctx.ActiveTabula.Tokens == nil {
		ctx.ActiveTabula.Tokens = map[string]image.Point{}
	}
	ctx.ActiveTabula.Tokens[name] = coord
	ctx.ActiveTabula.Save(db.Instance)

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(ctx.ActiveTabula))
}
