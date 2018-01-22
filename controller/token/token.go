package token

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/colors"
	"github.com/pdbogen/mapbot/common/conv"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/controller/cmdproc"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/mark"
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
			"add":     cmdproc.Subcommand{"[<name>] <point> [[<name2>] <pt2> ... [<nameN>] <ptN>]", "add a token(s) (or change its location) to the currently selected map (see `map select`). Token names should be emoji! (Or very short words). Space between coordinate pairs is optional.", cmdAdd},
			"move":    cmdproc.Subcommand{"[<name>] <point>", "synonym for add", cmdAdd},
			"color":   cmdproc.Subcommand{"[<name>] <color>", "sets the color for the given token, which can be a common name; the world 'clear'; a 6-digit hex code specifying red, green, and blue (optionally with two more digits specifying Alpha); https://en.wikipedia.org/wiki/List_of_Crayola_crayon_colors has a great list of colors.", cmdColor},
			"list":    cmdproc.Subcommand{"", "list tokens on the active map", cmdList},
			"clear":   cmdproc.Subcommand{"", "clear tokens from the field", cmdClear},
			"remove":  cmdproc.Subcommand{"[<name>]", "removes the named token from the active map.", cmdRemove},
			"swap":    cmdproc.Subcommand{"[<old>] <new>", "replace an old token with a new token, retaining other settings (location/color).", cmdSwap},
			"replace": cmdproc.Subcommand{"[<old>] <new>", "synonym for swap", cmdSwap},
			"size":    cmdproc.Subcommand{"[<name>] <size>", "sets the named token to be <size> squares big; medium creatures at 1, large are 2, etc.", cmdSize},
			"light":   cmdproc.Subcommand{"[<name>] <dim> [<normal> [<bright>]]", "sets 'light levels' to project as marks around the token; dim is orange, normal is yellow, and bright is bright yellow. values are in 'pathfinder feet'.", cmdLight},
		},
		Comment: "For command where the token effected is enclosed in `[]`, it is optional, and if not provided, the last token you have added or moved is effected.",
	}
}

func cmdLight(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 1 || len(args) > 4 {
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

	if _, err := strconv.Atoi(args[0]); err == nil {
		tok := c.Context.GetLastToken(c.User.Id)
		if tok == "" {
			h.Error(c, fmt.Sprintf("`%s` looks like a distance, but I don't remember the last token you moved.", args[0]))
			return
		}
		args = append([]string{tok}, args...)
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

	if len(args) < 1 || len(args) > 2 {
		h.Error(c, "usage: token size "+processor.Commands["size"].Args)
		return
	}

	if _, err := strconv.Atoi(args[0]); err == nil {
		tok := c.Context.GetLastToken(c.User.Id)
		if tok == "" {
			h.Error(c, fmt.Sprintf("`%s` looks like a size, but I don't remember the last token you moved.", args[0]))
			return
		}
		args = append([]string{tok}, args...)
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

	if len(args) < 1 || len(args) > 2 {
		h.Error(c, "usage: token swap "+processor.Commands["swap"].Args)
		return
	}

	if len(args) == 1 {
		tok := c.Context.GetLastToken(c.User.Id)
		if tok == "" {
			h.Error(c, fmt.Sprintf("You only gave me one token (`%s`), but I don't remember the last token you moved.", args[0]))
			return
		}
		args = append([]string{tok}, args...)
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

	if len(args) == 1 {
		tok := c.Context.GetLastToken(c.User.Id)
		if tok == "" {
			h.Error(c, fmt.Sprintf("You only gave me a color (`%s`), but I don't remember the last token you moved.", args[0]))
			return
		}
		args = append([]string{tok}, args...)
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
		tok := c.Context.GetLastToken(c.User.Id)
		if tok == "" {
			h.Error(c, "You didn't tell me what to remove, and I don't remember the last token you moved.")
			return
		}
		args = append([]string{tok}, args...)
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

func parseMovements(args []string, lastToken string) (map[string][]image.Point, error) {
	curToken := lastToken
	tokens := map[string][]image.Point{}
	for i := 0; i < len(args); i++ {
		a := args[i]

		if i+1 < len(args) {
			if pt, err := conv.CoordsToPoint(a, args[i+1]); err == nil {
				if curToken == "" {
					return nil, fmt.Errorf("I found a coordinate (`%s%s`), but you didn't give me a token, and I don't remember the last token you moved.", args[i], args[i+1])
				}
				tokens[curToken] = append(tokens[curToken], pt)
				i++
				continue
			}
		}

		// If it's not a two word coordinate and we don't have a token yet, this must be a token. But if it's the last or only token, it could instad be a
		// coordinate.
		if curToken == "" && i+1 < len(args) {
			curToken = a
			continue
		}

		if pt, _, err := conv.RCToPoint(a, false); err == nil {
			if curToken == "" {
				return nil, fmt.Errorf("I found a coordinate (`%s`), but you didn't give me a token, and I don't remember the last token you moved.", a)
			}
			tokens[curToken] = append(tokens[curToken], pt)
			continue
		}

		curToken = a
	}

	if _, ok := tokens[curToken]; !ok {
		return nil, fmt.Errorf("you ended with a token %q, but did not specify any movements for it", curToken)
	}

	return tokens, nil
}

func cmdAdd(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) == 0 {
		h.Error(c, "`token add` expects at least a coordinate pair; usage: token add "+processor.Commands["add"].Args)
		return
	}

	// We got two arguments, but they could be a space-separated coordinate pair
	if len(args) == 2 {
		if _, err := conv.CoordsToPoint(args[0], args[1]); err == nil {
			args = []string{args[0] + args[1]}
		}
	}

	curToken := c.Context.GetLastToken(c.User.Id)

	tokens, err := parseMovements(args, curToken)
	if err != nil {
		h.Error(c, err.Error())
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

	lines := []mark.Line{}

	lastToken := ""
	for name, coords := range tokens {
		if tab.Tokens == nil {
			tab.Tokens = map[types.ContextId]map[string]tabula.Token{}
		}

		if tab.Tokens[c.Context.Id()] == nil {
			tab.Tokens[c.Context.Id()] = map[string]tabula.Token{}
		}

		for _, coord := range coords {
			if tok, ok := tab.Tokens[c.Context.Id()][name]; !ok {
				tab.Tokens[c.Context.Id()][name] = tabula.Token{
					Coordinate: coord,
					TokenColor: color.RGBA{0, 0, 0, 0},
					Size:       1,
				}
			} else {
				orig := tok.Coordinate
				tab.Tokens[c.Context.Id()][name] = tok.WithCoords(coord)
				lines = append(lines,
					mark.Line{A: orig, B: coord, CA: "ne", CB: "ne", Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}},
					mark.Line{A: orig, B: coord, CA: "se", CB: "se", Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}},
					mark.Line{A: orig, B: coord, CA: "sw", CB: "sw", Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}},
					mark.Line{A: orig, B: coord, CA: "nw", CB: "nw", Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}},
				)
			}
		}
		lastToken = name
	}

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	if err := c.User.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving your user record")
		log.Errorf("error saving user record %v: %s", c.User, err)
	}

	c.Context.SetLastToken(c.User.Id, lastToken)
	if err := c.Context.Save(); err != nil {
		h.Error(c, "an error occured saving the context")
		log.Errorf("error saving context: %s", err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab.WithLines(lines)))
}
