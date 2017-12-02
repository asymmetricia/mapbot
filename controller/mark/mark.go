package mark

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/colors"
	"github.com/pdbogen/mapbot/common/conv"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	"image"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:mark", cmdMark)
}

const usage = "usage: mark <square> [<square2> ... <squareN>] <color>\nspecify one ore more squared followed by a color; you may repeat this multiple times (e.g.: `mark a1 red a2 a3 a4 green b1 blue`)\n"

func cmdMark(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok || len(args) == 0 {
		h.Error(c, usage)
		return
	}

	squares := []image.Point{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		// Option 1: RC-style coordinate
		// Option 2: Row letter; i+1 contains column
		// Option 3: color
		if pt, err := conv.RCToPoint(a); err == nil {
			squares = append(squares, pt)
			continue
		}

		if i+1 < len(args) {
			if pt, err := conv.CoordsToPoint(a, args[i+1]); err == nil {
				squares = append(squares, pt)
				i++
				continue
			}
		}

		if _, err := colors.ToColor(a); err != nil {
			// FIXME paint the squares the color
			// reset the list of squares
			squares = []image.Point{}
			continue
		}

		h.Error(c, fmt.Sprintf("I couldn't figure out what you mean by `%s`.\n%s", a, usage))
		return
	}
}
