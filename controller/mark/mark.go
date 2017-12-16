package mark

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/colors"
	"github.com/pdbogen/mapbot/common/conv"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/mark"
	"github.com/pdbogen/mapbot/model/tabula"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:mark", cmdMark)
}

const usage = "usage: mark <place> [<place2> ... <placeN>] <color>\nspecify one ore more places followed by a color. There are a few ways to specify a place:\n" +
	"    a square -- given by a coordinate, with or without a space; i.e., `a1` or `a 1`\n" +
	"    a side   -- given by a coordinate (no space) and a cardinal direction (n, s, e, w); example: `a1n` or `a1s`\n" +
	"    a corner -- given by a coordinate (no space) and an intercardinal direction (ne, se, sw, nw); example: `a1ne`"

func cmdMark(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok || len(args) == 0 {
		h.Error(c, usage)
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	marks := []mark.Mark{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		// Option 1: RC-style coordinate (maybe with a direction)
		// Option 2: Row letter; i+1 contains column
		// Option 3: color
		if pt, dir, err := conv.RCToPoint(a, true); err == nil {
			marks = append(marks, mark.Mark{Point: pt, Direction: dir})
			continue
		}

		if i+1 < len(args) {
			if pt, err := conv.CoordsToPoint(a, args[i+1]); err == nil {
				marks = append(marks, mark.Mark{Point: pt})
				i++
				continue
			}
		}

		if color, err := colors.ToColor(a); err == nil {
			// paint the squares the color
			for _, m := range marks {
				m = m.WithColor(color)
				log.Debugf("setting mark %v on tabula %v", m, *tabId)
				c.Context.Mark(*tabId, m)
			}
			// reset the list of squares
			marks = []mark.Mark{}
			continue
		}

		h.Error(c, fmt.Sprintf("I couldn't figure out what you mean by `%s`.\n%s", a, usage))
		return
	}

	if err := c.Context.Save(); err != nil {
		log.Errorf("saving marks: %s", err)
		h.Error(c, "A problem occurred while saving your marks. This could indicate an bug.")
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}
