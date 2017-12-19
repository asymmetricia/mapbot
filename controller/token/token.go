package token

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/colors"
	"github.com/pdbogen/mapbot/common/conv"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/controller/cmdproc"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"image"
	"image/color"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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
			"add":    cmdproc.Subcommand{"<name> <X> <Y> [<name2> <x2> <y2> ... <nameN> <xN> <yN>]", "add a token(s) (or change its location) to the currently selected map (see `map select`). Token names should be emoji! (Or very short words). Space between coordinate pairs is optional.", cmdAdd},
			"move":   cmdproc.Subcommand{"<name> <X> <Y>", "synonym for add", cmdAdd},
			"color":  cmdproc.Subcommand{"<name> <color>", "sets the color for the given token, which can be a common name; the world 'clear'; a 6-digit hex code specifying red, green, and blue (optionally with two more digits specifying Alpha); https://en.wikipedia.org/wiki/List_of_Crayola_crayon_colors has a great list of colors.", cmdColor},
			"list":   cmdproc.Subcommand{"", "list tokens on the active map", cmdList},
			"clear":  cmdproc.Subcommand{"", "clear tokens from the field", cmdClear},
			"remove": cmdproc.Subcommand{"<name>", "removes the named token from the active map", cmdRemove},
			"swap":   cmdproc.Subcommand{"<old> <new>", "replace an old token with a new token, retaining other settings (location/color)", cmdSwap},
			"size":   cmdproc.Subcommand{"<name> <size>", "sets the named token to be <size> squares big; medium creatures at 1, large are 2, etc.", cmdSize},
			"light":  cmdproc.Subcommand{"<name> <dim> [<normal> [<bright>]]", "sets 'light levels' to project as marks around the token; dim is orange, normal is yellow, and bright is bright yellow. values are in 'pathfinder feet'.", cmdLight},
		},
	}
}

func cmdLight(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 2 || len(args) > 4 {
		h.Error(c, "usage: token light "+processor.Commands["light"].Args)
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

	dim, err := strconv.Atoi(args[1])
	if err != nil {
		h.Error(c, fmt.Sprintf("`%s` is not a number of feet: %s", args[1], err))
		return
	}

	var normal, bright int

	if len(args) >= 3 {
		normal, err = strconv.Atoi(args[2])
		if err != nil {
			h.Error(c, fmt.Sprintf("`%s` is not a number of feet: %s", args[2], err))
			return
		}
	}

	if len(args) == 4 {
		bright, err = strconv.Atoi(args[3])
		if err != nil {
			h.Error(c, fmt.Sprintf("`%s` is not a number of feet: %s", args[3], err))
			return
		}
	}

	tab.Tokens[c.Context.Id()][args[0]] = tab.Tokens[c.Context.Id()][args[0]].WithLight(dim, normal, bright)

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdSize(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) != 2 {
		h.Error(c, "usage: token size "+processor.Commands["size"].Args)
		return
	}

	size, err := strconv.Atoi(args[1])
	if err != nil {
		h.Error(c, fmt.Sprintf("%q is not an integer: %s\nusage: token size %s", args[1], err, processor.Commands["size"].Args))
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

	name := args[0]

	if tab.Tokens == nil || tab.Tokens[c.Context.Id()] == nil {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	token, tokenOk := tab.Tokens[c.Context.Id()][name]
	if !tokenOk {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	tab.Tokens[c.Context.Id()][name] = token.WithSize(size)

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdSwap(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) != 2 {
		h.Error(c, "usage: token swap "+processor.Commands["swap"].Args)
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

	name := args[0]

	if tab.Tokens == nil || tab.Tokens[c.Context.Id()] == nil {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	token, tokenOk := tab.Tokens[c.Context.Id()][name]
	if !tokenOk {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	tab.Tokens[c.Context.Id()][args[1]] = token
	delete(tab.Tokens[c.Context.Id()], name)

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdColor(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) != 2 {
		h.Error(c, "usage: token color "+processor.Commands["color"].Args)
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

	name := args[0]

	if tab.Tokens == nil || tab.Tokens[c.Context.Id()] == nil {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	token, tokenOk := tab.Tokens[c.Context.Id()][name]
	if !tokenOk {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	newColor, err := colors.ToColor(args[1])
	if err != nil {
		h.Error(c, err.Error())
		return
	}

	tab.Tokens[c.Context.Id()][name] = token.WithColor(newColor)

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

var emojiToken = regexp.MustCompile(`^(:[^:]+:)`)

func cmdList(h *hub.Hub, c *hub.Command) {
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

	ctxId := c.Context.Id()
	if tab.Tokens == nil || tab.Tokens[ctxId] == nil || len(tab.Tokens[ctxId]) == 0 {
		h.Reply(c, "There are no tokens on the active map.")
		return
	}

	rep := fmt.Sprintf("There are %d tokens on the active map:", len(tab.Tokens[ctxId]))
	for name, token := range tab.Tokens[ctxId] {
		bits := emojiToken.FindStringSubmatch(name)
		if bits != nil {
			name = name + " (`" + bits[1] + "`)"
		}
		rep += fmt.Sprintf("\n- %s at (%d,%d)", name, token.Coordinate.X, token.Coordinate.Y)

		r, g, b, a := token.Color().RGBA()
		if a > 0 {
			rep += fmt.Sprintf(", color (%d,%d,%d,%d)", r, g, b, a)
		}

		lights := []string{}
		if token.DimLight > 0 {
			lights = append(lights, fmt.Sprintf("%dft dim", token.DimLight))
		}
		if token.NormalLight > 0 {
			lights = append(lights, fmt.Sprintf("%dft normal", token.NormalLight))
		}
		if token.BrightLight > 0 {
			lights = append(lights, fmt.Sprintf("%dft bright", token.BrightLight))
		}
		if len(lights) > 0 {
			rep += fmt.Sprintf(", light (%s)", strings.Join(lights, ", "))
		}
	}
	h.Reply(c, rep)
	return
}

func cmdRemove(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) == 0 {
		h.Error(c, "`token remove` expects a list of tokens to clear. usage: `token clear "+processor.Commands["remove"].Args+"`")
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

	for _, token := range args {
		log.Debugf("removing token %s", token)
		delete(tab.Tokens[c.Context.Id()], token)
	}

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdClear(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) > 0 {
		h.Error(c, "`token clear` expects no arguments; usage: token clear")
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

	tab.Tokens[c.Context.Id()] = map[string]tabula.Token{}

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdAdd(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	tokens := map[string]image.Point{}

	// Walk through the arguments; we have to get a token name first, and then a coordinate pair, space-separated or not.
	curToken := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if curToken == "" {
			curToken = a
			continue
		}
		if pt, _, err := conv.RCToPoint(a, false); err == nil {
			tokens[curToken] = pt
			curToken = ""
			continue
		}

		if i+1 < len(args) {
			if pt, err := conv.CoordsToPoint(a, args[i+1]); err == nil {
				tokens[curToken] = pt
				curToken = ""
				i++
				continue
			}
		}
		tokens = map[string]image.Point{}
		break
	}

	if len(tokens) == 0 {
		h.Error(c, "`token add` expects at least a token name and coordinate pair; usage: token add "+processor.Commands["add"].Args)
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

	for name, coord := range tokens {
		if tab.Tokens == nil {
			tab.Tokens = map[types.ContextId]map[string]tabula.Token{}
		}

		if tab.Tokens[c.Context.Id()] == nil {
			tab.Tokens[c.Context.Id()] = map[string]tabula.Token{}
		}

		if tok, ok := tab.Tokens[c.Context.Id()][name]; !ok {
			tab.Tokens[c.Context.Id()][name] = tabula.Token{
				Coordinate: coord,
				TokenColor: color.RGBA{0, 0, 0, 0},
				Size:       1,
			}
		} else {
			tab.Tokens[c.Context.Id()][name] = tok.WithCoords(coord)
		}
	}

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}
