package mapController

import (
	"errors"
	"fmt"
	"github.com/pdbogen/mapbot/common/colors"
	"github.com/pdbogen/mapbot/common/conv"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/controller/cmdproc"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/tabula"
	"image"
	"image/color"
	"regexp"
	"strconv"
	"strings"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:map", processor.Route)
}

var processor *cmdproc.CommandProcessor

func init() {
	processor = &cmdproc.CommandProcessor{
		Command: "map",
		Commands: map[string]cmdproc.Subcommand{
			"add":       cmdproc.Subcommand{"<name> <url>", "add a map to your collection", cmdAdd},
			"remove":    cmdproc.Subcommand{"<name>", "remove a map from your collection", cmdRemove},
			"delete":    cmdproc.Subcommand{"<name>", "remove a map from your collection", cmdRemove},
			"show":      cmdproc.Subcommand{"[<name>]", "show a the named map; or the active map in this context, if any", cmdShow},
			"set":       cmdproc.Subcommand{"[<name>] {offsetX|offsetY|dpi|gridColor} <value>[ <key2> <value2> ...]", "set a property of an existing map; offsetX, offsetY, and dpi accepts numbers; color accepts some common color names or a six-digit hex code. If no map is specified, selected map is used.", cmdSet},
			"list":      cmdproc.Subcommand{"", "list your maps", cmdList},
			"select":    cmdproc.Subcommand{"<name>", "selects the map active in this channel. active tokens will be cleared.", cmdSelect},
			"dpi":       cmdproc.Subcommand{"<name> <dpi>", "shorthand for set, to set the map DPI", cmdDpi},
			"gridcolor": cmdproc.Subcommand{"<name> <value>", "shorthand for set, to set the grid color", cmdGridColor},
			"zoom":      cmdproc.Subcommand{"<min X> <min Y> <max X> <max Y>", "requests that mapbot display only a portion of the map; useful for larger maps where the action is in a small area. requires an active map. Set to `a 1 a 1` to disable zoom. The space between column and row is optional (i.e., `a1` is OK).", cmdZoom},
			"align":     cmdproc.Subcommand{"<name>", "begin guided alignment for the named map", cmdAlign},
			"mark":      cmdproc.Subcommand{"", "alias for non-map command `mark`; see `mark help` for more", cmdMark},
			"check":     cmdproc.Subcommand{"", "alias for non-map command `check`; see `check help` for more", cmdMark},
			"autozoom":  cmdproc.Subcommand{"", "sets the zoom so that all current tokens are visible, with a small margin", cmdAutoZoom},
		},
	}
}

func cmdMark(h *hub.Hub, c *hub.Command) {
	args := strings.Split(string(c.Type), ":")
	h.Publish(c.WithType(hub.CommandType(strings.Join(args[1:], ":"))))
}

func cmdAlign(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		log.Debugf("received non-[]string payload %T", c.Payload)
		h.Error(c, "usage: map set "+processor.Commands["set"].Args)
		return
	}

	if len(args) != 1 {
		log.Debugf("received %d args, expected exactly 1", len(args))
		h.Error(c, "usage: map align "+processor.Commands["set"].Args)
		return
	}

	t, ok := c.User.TabulaByName(tabula.TabulaName(args[0]))
	if !ok {
		h.Error(c, notFound(tabula.TabulaName(args[0])))
		return
	}

	h.Publish(&hub.Command{
		User:    c.User,
		From:    c.From,
		Context: c.Context,
		Payload: []string{"start", "align", string(c.User.Id), strconv.FormatInt(int64(*t.Id), 10)},
		Type:    "user:workflow",
	})
}

func cmdAutoZoom(h *hub.Hub, c *hub.Command) {
	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Load(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	ctxId := c.Context.Id()
	if tab.Tokens == nil || tab.Tokens[ctxId] == nil || len(tab.Tokens[ctxId]) == 0 {
		h.Reply(c, "There are no tokens on the active map.")
		return
	}

	first := true
	min_x, min_y, max_x, max_y := 0, 0, 0, 0
	for _, token := range tab.Tokens[ctxId] {
		if first {
			min_x = token.Coordinate.X
			max_x = token.Coordinate.X
			min_y = token.Coordinate.Y
			max_y = token.Coordinate.Y
			first = false
			continue
		}

		if token.Coordinate.X < min_x {
			min_x = token.Coordinate.X
		}
		if token.Coordinate.X+token.Size > max_x {
			max_x = token.Coordinate.X + token.Size - 1
		}
		if token.Coordinate.Y < min_y {
			min_y = token.Coordinate.Y
		}
		if token.Coordinate.Y+token.Size > max_y {
			max_y = token.Coordinate.Y + token.Size - 1
		}
	}

	cmdZoom(h, c.WithPayload([]string{
		conv.PointToCoords(image.Pt(min_x-1, min_y-1)),
		conv.PointToCoords(image.Pt(max_x+1, max_y+1)),
	}))
}

func cmdZoom(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok || len(args) > 4 || len(args) < 2 {
		h.Error(c, "usage: map zoom "+processor.Commands["zoom"].Args)
		return
	}

	var minCoord, maxCoord image.Point
	var err error
	var state int
loop:
	for i := 0; i < len(args); i++ {
		var c image.Point
		a := args[i]
		c, _, err = conv.RCToPoint(a, false)
		if err != nil && i+1 < len(args) {
			c, err = conv.CoordsToPoint(a, args[i+1])
			i++
		}
		if err != nil {
			break
		}
		switch state {
		case 0:
			minCoord = c
		case 1:
			maxCoord = c
		case 2:
			err = errors.New("extra arguments found")
			break loop
		}
		state++
	}

	if err != nil {
		h.Error(c, fmt.Sprintf("invalid coordinates %s: %s", strings.Join(args, " "), err))
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "there is no active map; use `map select <mapName>` to pick one")
		return
	}

	t, err := tabula.Load(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "error loading active map")
		log.Errorf("error loading active map %d: %s", tabId, err)
		return
	}

	c.Context.SetZoom(minCoord.X, minCoord.Y, maxCoord.X, maxCoord.Y)
	if err := c.Context.Save(); err != nil {
		h.Error(c, fmt.Sprintf("Something went wrong while saving your change: %s", err))
	}

	h.Publish(&hub.Command{
		Type:    hub.CommandType(c.From),
		Payload: t,
		User:    c.User,
	})
	h.PublishUpdate(c.Context)
}

func cmdDpi(h *hub.Hub, c *hub.Command) {
	if args, ok := c.Payload.([]string); ok && (len(args) == 1 || len(args) == 2) {
		if len(args) == 1 {
			args = []string{"dpi", args[0]}
		} else {
			args = []string{args[0], "dpi", args[1]}
		}
		cmdSet(h, c.WithPayload(args))
		return
	} else {
		h.Error(c, "usage: map dpi "+processor.Commands["dpi"].Args)
	}
}

func cmdGridColor(h *hub.Hub, c *hub.Command) {
	if args, ok := c.Payload.([]string); ok && (len(args) == 1 || len(args) == 2) {
		if len(args) == 1 {
			args = []string{"gridColor", args[0]}
		} else {
			args = []string{args[0], "gridColor", args[1]}
		}
		cmdSet(h, c.WithPayload(args))
		return
	} else {
		h.Error(c, "usage: map gridcolor "+processor.Commands["gridcolor"].Args)
	}
}

func cmdList(h *hub.Hub, c *hub.Command) {
	var response string

	if c.User.Tabulas == nil || len(c.User.Tabulas) == 0 {
		response = "You have no maps."
	} else {
		res := []string{
			"Your maps:",
		}
		for _, t := range c.User.Tabulas {
			res = append(res, fmt.Sprintf("%s - DPI: %.1f, Offset: (%d,%d)", t.Name, t.Dpi, t.OffsetX, t.OffsetY))
		}
		response = strings.Join(res, "\n")
	}

	h.Publish(&hub.Command{
		Type:    hub.CommandType(c.From),
		Payload: response,
		User:    c.User,
	})
}

var colorRegex = regexp.MustCompile("^#?[0-9a-fA-F]{6}$")

func cmdSet(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		log.Debugf("received non-[]string payload %T", c.Payload)
		h.Error(c, "usage: map set "+processor.Commands["set"].Args)
		return
	}

	if len(args) < 2 {
		log.Debugf("received %d args, expected at least 2", len(args))
		h.Error(c, "usage: map set "+processor.Commands["set"].Args)
		return
	}

	var t *tabula.Tabula
	// We just have pairs, so assume we're using active map
	if len(args)%2 == 0 {
		tabId := c.Context.GetActiveTabulaId()
		if tabId == nil {
			h.Error(c, "no active map in this channel, use `map select <name>` to pick one, or provide a map name")
			return
		}

		var err error
		t, err = tabula.Load(db.Instance, *tabId)
		if err != nil {
			h.Error(c, "error loading active map")
			log.Errorf("error loading active map %d: %s", tabId, err)
			return
		}
	} else {
		var ok bool
		t, ok = c.User.TabulaByName(tabula.TabulaName(args[0]))
		if !ok {
			h.Error(c, notFound(tabula.TabulaName(args[0])))
			return
		}
		args = args[1:]
	}

	for i := 0; i < len(args); i += 2 {
		switch strings.ToLower(args[i]) {
		case "gridcolor":
			if col, ok := colors.Colors[strings.ToLower(args[i+1])]; ok {
				t.GridColor = &col
			} else if !colorRegex.MatchString(args[i+1]) {
				colCode := strings.TrimLeft(args[i+1], "#")
				r, _ := strconv.ParseInt(colCode[0:2], 16, 9)
				g, _ := strconv.ParseInt(colCode[2:4], 16, 9)
				b, _ := strconv.ParseInt(colCode[4:6], 16, 9)
				t.GridColor = &color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
			} else {
				h.Error(c, "color should be a common color name; or an HTML-style RGB code, i.e., (red) #FF0000, (green) #00FF00, (blue) #0000FF")
				return
			}
		case "offsetx":
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				h.Error(c, fmt.Sprintf("value %q was not an integer: %s", args[i+1], err))
				return
			}
			t.OffsetX = n
		case "offsety":
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				h.Error(c, fmt.Sprintf("value %q was not an integer: %s", args[i+1], err))
				return
			}
			t.OffsetY = n
		case "dpi":
			n, err := strconv.ParseFloat(args[i+1], 32)
			if err != nil {
				h.Error(c, fmt.Sprintf("value %q was not an floating-point number: %s", args[i+1], err))
				return
			}
			if n == 0 {
				h.Error(c, "DPI cannot be zero")
				return
			}
			t.Dpi = float32(n)
		default:
			h.Error(c, fmt.Sprintf("hmmm, I don't know how to set %s. Please try: map set %s", args[i], processor.Commands["set"].Args))
			return
		}
		t.Version++
		if err := t.Save(db.Instance); err != nil {
			h.Error(c, fmt.Sprintf("failed saving updated map: %s", err))
			return
		}
		h.Publish(&hub.Command{
			Type:    hub.CommandType(c.From),
			Payload: fmt.Sprintf("map `%s` %s set to `%s`", args[0], args[i], args[i+1]),
			User:    c.User,
		})
	}
	h.Publish(&hub.Command{
		Type:    hub.CommandType(c.From),
		Payload: t,
		User:    c.User,
	})
	h.PublishUpdate(c.Context)
}

func cmdSelect(h *hub.Hub, c *hub.Command) {
	if args, ok := c.Payload.([]string); ok && len(args) == 1 {
		t, ok := c.User.TabulaByName(tabula.TabulaName(args[0]))
		if !ok {
			h.Error(c, notFound(tabula.TabulaName(args[0])))
			return
		}

		c.Context.SetActiveTabulaId(t.Id)
		c.Context.SetZoom(0, 0, 0, 0)

		if err := c.Context.Save(); err != nil {
			log.Errorf("Error saving context: %s", err)
			h.Error(c, "error saving context")
			return
		}

		h.Publish(&hub.Command{
			Type:    hub.CommandType(c.From),
			Payload: t,
			User:    c.User,
		})
		h.PublishUpdate(c.Context)
	} else {
		h.Error(c, "usage: map select <name>")
	}
}

func cmdShow(h *hub.Hub, c *hub.Command) {
	if args, ok := c.Payload.([]string); ok {
		var t *tabula.Tabula
		switch len(args) {
		case 0:
			tabId := c.Context.GetActiveTabulaId()
			if tabId == nil {
				h.Error(c, "no active map in this channel, use `map select <name>` to pick one, or provide a map name directly")
				return
			}

			var err error
			t, err = tabula.Load(db.Instance, *tabId)
			if err != nil {
				h.Error(c, "error loading active map")
				log.Errorf("error loading active map %d: %s", tabId, err)
				return
			}
		case 1:
			var ok bool
			t, ok = c.User.TabulaByName(tabula.TabulaName(args[0]))
			if !ok {
				h.Error(c, notFound(tabula.TabulaName(args[0])))
				return
			}
		default:
			h.Error(c, "usage: map show <name>")
			return
		}
		h.Publish(&hub.Command{
			Type:    hub.CommandType(c.From),
			Payload: t,
			User:    c.User,
		})
	}
}

func cmdRemove(h *hub.Hub, c *hub.Command) {
	if c.User == nil {
		log.Errorf("received command with nil user")
		return
	}
	args, ok := c.Payload.([]string)
	if !ok || len(args) != 1 {
		h.Error(c, "usage: map remove <map name>")
		return
	}

	t, ok := c.User.TabulaByName(tabula.TabulaName(args[0]))
	if !ok {
		h.Error(c, fmt.Sprintf("you have no map named %q", args[0]))
		return
	}

	if err := t.Delete(db.Instance); err != nil {
		h.Error(c, fmt.Sprintf("error updating your list of tables: %s", err))
		return
	}

	for i, t := range c.User.Tabulas {
		if t.Name == tabula.TabulaName(args[0]) {
			c.User.Tabulas = append(c.User.Tabulas[:i], c.User.Tabulas[i+1:]...)
			break
		}
	}

	h.Reply(c, "gone!")
}

func cmdAdd(h *hub.Hub, c *hub.Command) {
	if c.User == nil {
		log.Errorf("received command with nil user")
		return
	}
	if args, ok := c.Payload.([]string); ok && len(args) == 2 {
		t, ok := c.User.TabulaByName(tabula.TabulaName(args[0]))
		if ok {
			t.Url = args[1]

			if err := t.Save(db.Instance); err != nil {
				h.Error(c, fmt.Sprintf("error saving map to database: %s", err))
				return
			}
		} else {
			h.Publish(&hub.Command{
				Type:    hub.CommandType(c.From),
				Payload: "Getting background image.. this could take a moment.",
				User:    c.User,
			})
			t, err := tabula.New(args[0], args[1])
			if err != nil {
				h.Error(c, fmt.Sprintf("error creating map: %s", err))
				return
			}

			if err := t.Save(db.Instance); err != nil {
				h.Error(c, fmt.Sprintf("error saving map to database: %s", err))
				return
			}

			if err := c.User.Assign(db.Instance, t); err != nil {
				h.Error(c, fmt.Sprintf("error saving user record to database: %s", err))
				return
			}
		}

		h.Publish(&hub.Command{
			Type:    hub.CommandType(c.From),
			Payload: fmt.Sprintf("map %q saved", args[0]),
			User:    c.User,
		})
	} else {
		h.Error(c, "usage: map add <name> <url>")
	}
}

func notFound(n tabula.TabulaName) string {
	return fmt.Sprintf("you don't have a map named %q", string(n))
}
