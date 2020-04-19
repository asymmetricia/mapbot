package mark

import (
	"errors"
	"fmt"
	"github.com/pdbogen/mapbot/common/colors"
	"github.com/pdbogen/mapbot/common/conv"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/mark"
	"github.com/pdbogen/mapbot/model/tabula"
	"image"
	"math"
	"strconv"
	"strings"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:mark", cmdMark)
	h.Subscribe("user:check", cmdMark)
}

const syntax = "<place> [<place2> ... <placeN>] <color>\n" +
	"specify one or more places followed by a color. There are a few ways to specify a place:\n" +
	"    a square -- given by a coordinate, with or without a space; i.e., `a1` or `a 1`\n" +
	"    a side   -- given by a coordinate (no space) and a cardinal direction (n, s, e, w); example: `a1n` or `a1s`\n" +
	"    a corner -- given by a coordinate (no space) and an intercardinal direction (ne, se, sw, nw); example: `a1ne`\n" +
	"    a square -- use `square(top-left,bottom-right)` where `top-left` and `bottom-right` are coordinates (without spaces); example: `square(a1,f6)`\n" +
	"    a circle -- use `circle(center,radius)` where `center` is a square or corner and `radius` is a number of feet, assuming 5 feet per square; example: `circle(m10,15)` or `circle(m10ne,15)`\n" +
	"    a cone   -- use `cone(origin-corner,direction,radius)` where `origin-corner` is a square with corner; `direction` is one of the allowable directions from that corner (e.g., ne corner can project a cone north, northeast, or east); and radius is the size of the cone. 15-foot cones are special-cased according to Pahfinder rules, but all other cones are computed as all squares such that 3/4 corners are within a 90-degree cone, and all corners are within the radius. Example: `cone(f6ne,ne,20)`\n" +
	"    a line   -- (or lines) use `line(A,B)` where A and B are squares or corners. Specifying a square will draw lines to/from all corners of that square. Example: `line(a1se,f5)` will draw four lines, from a1se to all corners of f5."

func clearMarks(h *hub.Hub, c *hub.Command) {
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

	c.Context.ClearMarks(*tabId)

	if err := c.Context.Save(); err != nil {
		log.Errorf("saving marks: %s", err)
		h.Error(c, ":warning: A problem occurred while saving your marks. This could indicate an bug.")
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func consumeUntilSuffix(args []string, i *int, suffix string) string {
	ret := ""
	for _, arg := range args {
		ret = ret + arg
		if strings.HasSuffix(arg, suffix) {
			break
		}
		*i++
	}
	log.Debugf("consumed up to %s", ret)
	return ret
}

var markFuncs = map[string]func([]string) ([]mark.Mark, error){
	"square": marksFromSquare,
	"circle": mark.Circle,
	"cone":   marksFromCone,
}

var lineFuncs = map[string]func([]string) ([]mark.Line, error){
	"line":  linesFromLine,
	"lines": linesFromLine,
}

func cmdMark(h *hub.Hub, c *hub.Command) {
	cmdName := strings.Split(string(c.Type), ":")[1]

	var usage string
	if cmdName == "mark" {
		usage = fmt.Sprintf("usage: %s %s\nThis command will save marks on the map. Use `check` to visualize marks once.", cmdName, syntax)
	} else {
		usage = fmt.Sprintf("usage: %s %s\nThis command will NOT save marks. Use `mark` to save marks on the map.", cmdName, syntax)
	}

	args, ok := c.Payload.([]string)
	if !ok || len(args) == 0 {
		h.Error(c, usage)
		return
	}

	if len(args) == 1 && strings.ToLower(args[0]) == "clear" {
		clearMarks(h, c)
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Load(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occurred loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	marks := []mark.Mark{}
	coloredMarks := []mark.Mark{}
	lines := []mark.Line{}
	coloredLines := []mark.Line{}
	for i := 0; i < len(args); i++ {
		a := strings.ToLower(args[i])
		// Option 1: RC-style coordinate (maybe with a direction)
		// Option 2: Row letter; i+1 contains column
		// Option 3: A shape (i.e., square(a,b))
		// Option 4: color
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

		if f, ok := markFuncs[strings.ToLower(strings.Split(a, "(")[0])]; ok {
			term := consumeUntilSuffix(args[i:], &i, ")")
			args := strings.Split(strings.TrimRight(strings.Split(term, "(")[1], ")"), ",")
			m, err := f(args)
			if err != nil {
				h.Error(c, fmt.Sprintf(":warning: while parsing `%s`, %s", term, err))
				return
			}
			marks = append(marks, m...)
			continue
		}

		if f, ok := lineFuncs[strings.ToLower(strings.Split(a, "(")[0])]; ok {
			term := consumeUntilSuffix(args[i:], &i, ")")
			args := strings.Split(strings.TrimRight(strings.Split(term, "(")[1], ")"), ",")
			l, err := f(args)
			if err != nil {
				h.Error(c, fmt.Sprintf(":warning: while parsing `%s`, %s", term, err))
				return
			}
			lines = append(lines, l...)
			continue
		}

		if color, err := colors.ToColor(a); err == nil {
			// paint the squares the color
			for _, m := range marks {
				m = m.WithColor(color)
				coloredMarks = append(coloredMarks, m)
			}
			for _, l := range lines {
				l = l.WithColor(color)
				coloredLines = append(coloredLines, l)
			}
			// reset the list of squares
			marks = []mark.Mark{}
			lines = []mark.Line{}
			continue
		}

		h.Error(c, fmt.Sprintf(":warning: I couldn't figure out what you mean by `%s`.\n%s", a, usage))
		return
	}

	for _, m := range marks {
		coloredMarks = append(coloredMarks, m.WithColor(colors.Colors["red"]))
	}
	for _, l := range lines {
		coloredLines = append(coloredLines, l.WithColor(colors.Colors["red"]))
	}

	if cmdName == "mark" {
		log.Debugf("marking up map...")
		for _, m := range coloredMarks {
			c.Context.Mark(*tabId, m)
		}
		log.Debugf("saving marks...")
		if err := c.Context.Save(); err != nil {
			log.Errorf("saving marks: %s", err)
			h.Error(c, ":warning: A problem occurred while saving your marks. This could indicate an bug.")
		}

		log.Debugf("rendering with %d marks", len(coloredMarks))
		if len(lines) != 0 {
			h.Error(c, ":warning: lines are not supported as permanent marks")
		}
		h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab.WithLines(coloredLines)))
		h.PublishUpdate(c.Context)
	} else {
		log.Debugf("rendering with %d temporary marks", len(coloredMarks))
		// TODO: Figure out a way to make temporary marks available to asynchronous UIs
		h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab.WithMarks(coloredMarks).WithLines(coloredLines)))
	}
}

// List of pairs, each pair is a min and max, in units of Pi/4 (i.e., eighth of a circle)
var coneAngles = map[string][]float64{
	"e":  []float64{0, 1, 7, 8},
	"ne": []float64{0, 2},
	"n":  []float64{1, 3},
	"nw": []float64{2, 4},
	"w":  []float64{3, 5},
	"sw": []float64{4, 6},
	"s":  []float64{5, 7},
	"se": []float64{6, 8, 0, 0},
}

var specialCones = map[string]map[int][]image.Point{
	"n": map[int][]image.Point{
		15: []image.Point{
			{-1, -3}, {0, -3}, {1, -3},
			{-1, -2}, {0, -2}, {1, -2},
			/*     */ {0, -1},
		},
	},
	"s": map[int][]image.Point{
		15: []image.Point{
			/*    */ {0, 1},
			{-1, 2}, {0, 2}, {1, 2},
			{-1, 3}, {0, 3}, {1, 3},
		},
	},
	"e": map[int][]image.Point{
		15: []image.Point{
			/*   */ {2, -1}, {3, -1},
			{1, 0}, {2, 0}, {3, 0},
			/*   */ {2, 1}, {3, 1},
		},
	},
	"w": map[int][]image.Point{
		15: []image.Point{
			{-3, -1}, {-2, -1},
			{-3, 0}, {-2, 0}, {-1, 0},
			{-3, 1}, {-2, 1},
		},
	},
}

func angle(a image.Point, cA string, b image.Point, cB string) float64 {
	cdx := 0
	cdy := 0
	if a == b && cA == cB {
		return math.NaN()
	}
	if len(cA) != 0 && len(cA) != 2 || len(cA) != len(cB) {
		return math.NaN()
	}

	if cA != cB {
		if cA[1] != cB[1] {
			if cA[1] == 'e' {
				cdx--
			} else {
				cdx++
			}
		}
		if cA[0] != cB[0] {
			if cA[0] == 'n' {
				cdy++
			} else {
				cdy--
			}
		}
	}

	dx := b.X - a.X + cdx
	dy := b.Y - a.Y + cdy

	if dx == 0 && dy == 0 {
		return math.NaN()
	}
	angle := math.Atan2(float64(-dy), float64(dx))
	if angle < 0 {
		return 2*math.Pi + angle
	} else {
		return angle
	}
}

func linesFromLine(args []string) (out []mark.Line, err error) {
	out = []mark.Line{}
	if len(args) != 2 {
		return nil, fmt.Errorf("`line()` expects two comma-separated arguments: `from`, `to`")
	}

	a, ac, err := conv.RCToPoint(args[0], true)
	if err != nil {
		return nil, fmt.Errorf("looked like a line, but could not parse coordinate `%s`: %s", args[0], err)
	}

	b, bc, err := conv.RCToPoint(args[1], true)
	if err != nil {
		return nil, fmt.Errorf("looked like a line, but could not parse coordinate `%s`: %s", args[1], err)
	}

	if len(ac) != 0 && len(ac) != 2 {
		return nil, fmt.Errorf("only corners or entire squares can be used to draw lines; you gave `%s`", args[0])
	}

	if len(bc) != 0 && len(bc) != 2 {
		return nil, fmt.Errorf("only corners or entire squares can be used to draw lines; you gave `%s`", args[1])
	}

	cornersA := []string{ac}
	if len(ac) == 0 {
		cornersA = []string{"ne", "se", "sw", "nw"}
	}

	cornersB := []string{bc}
	if len(bc) == 0 {
		cornersB = []string{"ne", "se", "sw", "nw"}
	}

	for _, cA := range cornersA {
		for _, cB := range cornersB {
			out = append(out, mark.Line{A: a, CA: cA, B: b, CB: cB})
		}
	}

	return out, nil
}

func marksFromCone(args []string) (out []mark.Mark, err error) {
	out = []mark.Mark{}
	if len(args) != 3 {
		return nil, fmt.Errorf("`cone()` expects three comma-separated arguments: `corner`, `direction`, `distance`")
	}
	origin, corner, err := conv.RCToPoint(args[0], true)
	if err != nil {
		return nil, fmt.Errorf("looked like a cone, but could not parse coordinate `%s`: %s", args[0], err)
	}

	if len(corner) != 2 {
		return nil, errors.New("cones must originate from corners")
	}

	if corner == "ne" && args[1] != "n" && args[1] != "ne" && args[1] != "e" ||
		corner == "se" && args[1] != "s" && args[1] != "se" && args[1] != "e" ||
		corner == "sw" && args[1] != "s" && args[1] != "sw" && args[1] != "w" ||
		corner == "nw" && args[1] != "n" && args[1] != "nw" && args[1] != "w" {
		return nil, fmt.Errorf("`%s` is not a legal direction from a %s corner", args[1], corner)
	}

	radius, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, fmt.Errorf("looked like a cone, but could not parse radius `%s`: %s", args[1], err)
	}

	if coneRanges, ok := specialCones[args[1]]; ok {
		if cone, ok := coneRanges[radius]; ok {
			for _, pt := range cone {
				out = append(out, mark.Mark{Point: pt.Add(origin)})
			}
			return out, nil
		}
	}

	angleRange := coneAngles[args[1]]

	for y := -radius / 5; y <= radius/5; y++ {
	coord:
		for x := -radius / 5; x <= radius/5; x++ {
			// each square has four corners, and all four must be within the right angle
			cornerCount := 0
			angles := []float64{
				angle(image.ZP, corner, image.Pt(x, y), "ne"),
				angle(image.ZP, corner, image.Pt(x, y), "nw"),
				angle(image.ZP, corner, image.Pt(x, y), "sw"),
				angle(image.ZP, corner, image.Pt(x, y), "se"),
			}
		corner:
			for _, angle := range angles {
				// if angle is NaN, the corners are co-incident
				if math.IsNaN(angle) {
					cornerCount++
					continue corner
				}
				angle = angle / math.Pi * 4
				for angleIdx := 0; angleIdx < len(angleRange); angleIdx += 2 {
					if angle >= angleRange[angleIdx] && angle <= angleRange[angleIdx+1] {
						cornerCount++
						continue corner
					}
				}
			}
			if cornerCount < 3 {
				continue coord
			}

			// and all four corners must be withn the right range
			for _, targetCorner := range []string{"ne", "nw", "sw", "se"} {
				if conv.DistanceCorners(image.ZP, corner, image.Pt(x, y), targetCorner) > radius {
					continue coord
				}
			}
			out = append(out, mark.Mark{Point: origin.Add(image.Pt(x, y))})
		}
	}

	return out, nil
}

func marksFromSquare(args []string) (out []mark.Mark, err error) {
	out = []mark.Mark{}
	if len(args) != 2 {
		return nil, fmt.Errorf("`square()` expects two comma-separated arguments")
	}

	min, _, err := conv.RCToPoint(args[0], false)
	if err != nil {
		return nil, fmt.Errorf("looked like a square, but could not parse coordinate `%s`: %s", args[0], err)
	}

	max, _, err := conv.RCToPoint(args[1], false)
	if err != nil {
		return nil, fmt.Errorf("looked like a square, but could not parse coordinate `%s`: %s", args[1], err)
	}

	if min.X > max.X {
		min.X, max.X = max.X, min.X
	}

	if min.Y > max.Y {
		min.Y, max.Y = max.Y, min.Y
	}

	pt := min
	for pt.Y <= max.Y {
		out = append(out, mark.Mark{Point: pt})
		pt.X++
		if pt.X > max.X {
			pt.X = min.X
			pt.Y++
		}
	}
	return out, nil
}
