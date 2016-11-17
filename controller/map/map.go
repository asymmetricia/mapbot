package mapController

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/tabula"
	"image/color"
	"regexp"
	"strconv"
	"strings"
)

var log = mbLog.Log

type subcommand struct {
	Args  string
	Usage string
	Cmd   hub.Subscriber
}

func Register(h *hub.Hub) {
	h.Subscribe("user:map", routeCommand)
}

var subCommands map[string]subcommand

func init() {
	subCommands = map[string]subcommand{
		"help": subcommand{"", "display this message", cmdHelp},
		"add":  subcommand{"<name> <url>", "add a map to your gazeteer", cmdAdd},
		"show": subcommand{"<name>", "show a gridded map", cmdShow},
		"set":  subcommand{"<name> {offsetX|offsetY|dpi|gridColor} <value>", "set a property of an existing map", cmdSet},
		"list": subcommand{"", "list your maps", cmdList},
	}
}

func notFound(n tabula.Name) string {
	return fmt.Sprintf("you don't have a map named %q", string(n))
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
		TeamId:  c.TeamId,
		User:    c.User,
	})
}

var colorRegex = regexp.MustCompile("^#[0-9a-fA-F]{8}$")

func cmdSet(h *hub.Hub, c *hub.Command) {
	if args, ok := c.Payload.([]string); ok && len(args) == 3 {
		t, ok := c.User.TabulaByName(tabula.Name(args[0]))
		if !ok {
			h.Error(c, notFound(tabula.Name(args[0])))
			return
		}
		switch strings.ToLower(args[1]) {
		case "gridcolor":
			if !colorRegex.MatchString(args[2]) {
				h.Error(c, "color should be an HTML-style RGB-with-alpha code, i.e., (red) #FF0000FF, (green) #00FF00FF, (blue) #0000FFFF")
				return
			}
			r, _ := strconv.ParseInt(args[2][1:3], 16, 9)
			g, _ := strconv.ParseInt(args[2][3:5], 16, 9)
			b, _ := strconv.ParseInt(args[2][5:7], 16, 9)
			a, _ := strconv.ParseInt(args[2][7:9], 16, 9)
			t.GridColor = &color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
		case "offsetx":
			n, err := strconv.Atoi(args[2])
			if err != nil {
				h.Error(c, fmt.Sprintf("value %q was not an integer: %s", args[2], err))
				return
			}
			t.OffsetX = n
		case "offsety":
			n, err := strconv.Atoi(args[2])
			if err != nil {
				h.Error(c, fmt.Sprintf("value %q was not an integer: %s", args[2], err))
				return
			}
			t.OffsetY = n
		case "dpi":
			n, err := strconv.ParseFloat(args[2], 32)
			if err != nil {
				h.Error(c, fmt.Sprintf("value %q was not an floating-point number: %s", args[2], err))
				return
			}
			if n == 0 {
				h.Error(c, "DPI cannot be zero")
				return
			}
			t.Dpi = float32(n)
		default:
			h.Error(c, "usage: map set "+subCommands["set"].Args)
			return
		}
		if err := t.Save(db.Instance); err != nil {
			h.Error(c, fmt.Sprintf("failed saving updated map: %s", err))
			return
		}
		h.Publish(&hub.Command{
			Type:    hub.CommandType(c.From),
			Payload: fmt.Sprintf("map %s %s set to %q", args[0], args[1], args[2]),
			TeamId:  c.TeamId,
			User:    c.User,
		})
	} else {
		h.Error(c, "usage: map set "+subCommands["set"].Args)
	}
}

func cmdShow(h *hub.Hub, c *hub.Command) {
	if args, ok := c.Payload.([]string); ok && len(args) == 1 {
		t, ok := c.User.TabulaByName(tabula.Name(args[0]))
		if ok {
			h.Publish(&hub.Command{
				Type:    hub.CommandType(c.From),
				Payload: t,
				TeamId:  c.TeamId,
				User:    c.User,
			})
		} else {
			h.Error(c, notFound(tabula.Name(args[0])))
		}
	} else {
		h.Error(c, "usage: map show <name>")
	}
}

func cmdAdd(h *hub.Hub, c *hub.Command) {
	if c.User == nil {
		log.Errorf("received command with nil user")
		return
	}
	if args, ok := c.Payload.([]string); ok && len(args) == 2 {
		h.Publish(&hub.Command{
			Type:    hub.CommandType(c.From),
			Payload: "Getting background image.. this could take a moment.",
			TeamId:  c.TeamId,
			User:    c.User,
		})
		t, err := tabula.New(args[0], args[1])
		if err != nil {
			h.Error(c, fmt.Sprintf("error creating map: %s", err))
			return
		}

		if err := t.Save(db.Instance); err != nil {
			h.Error(c, fmt.Sprintf("error saving to database: %s", err))
			return
		}

		if err := c.User.Assign(db.Instance, t); err != nil {
			h.Error(c, fmt.Sprintf("error saving to database: %s", err))
			return
		}

		h.Publish(&hub.Command{
			Type:    hub.CommandType(c.From),
			Payload: fmt.Sprintf("map %q saved", args[0]),
			TeamId:  c.TeamId,
			User:    c.User,
		})
	} else {
		h.Error(c, "usage: map add <name> <url>")
	}
}

func cmdHelp(h *hub.Hub, c *hub.Command) {
	help := "Select from the following map commands:"
	for cmd, sc := range subCommands {
		help += "\n" + cmd
		if sc.Args != "" {
			help += " " + sc.Args
		}
		help += " - " + sc.Usage
	}
	h.Publish(&hub.Command{
		Type:    hub.CommandType(c.From),
		Payload: help,
	})
}

func routeCommand(h *hub.Hub, c *hub.Command) {
	if args, ok := c.Payload.([]string); ok && len(args) > 0 {
		if s, ok := subCommands[args[0]]; ok {
			new_type := hub.CommandType(fmt.Sprintf("%s:%s", c.Type, args[0]))
			new_payload := args[1:]
			s.Cmd(h, c.WithType(new_type).WithPayload(new_payload))
		} else {
			h.Error(c, fmt.Sprintf("Sub-command %q not found; try 'help'", args[0]))
		}
	} else {
		h.Error(c, "No sub-command specified. Try 'help'")
	}
}
