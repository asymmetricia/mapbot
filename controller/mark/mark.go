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
	"    a circle -- use `circle(center,radius)` where `center` is a square or corner and `radius` is a number of feet, assuming 5 feet per square; example: `circle(m10,15)` or `circle(m10ne,15)`"

func clearMarks(h *hub.Hub, c *hub.Command) {
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

	c.Context.ClearMarks(*tabId)

	if err := c.Context.Save(); err != nil {
		log.Errorf("saving marks: %s", err)
		h.Error(c, ":warning: A problem occurred while saving your marks. This could indicate an bug.")
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
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

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	marks := []mark.Mark{}
	coloredMarks := []mark.Mark{}
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

		if strings.HasPrefix(a, "square(") && strings.HasSuffix(a, ")") {
			m, err := marksFromSquare(a)
			if err != nil {
				h.Error(c, fmt.Sprintf(":warning: %s", err))
				return
			}
			marks = append(marks, m...)
			continue
		}

		if strings.HasPrefix(a, "circle(") && strings.HasSuffix(a, ")") {
			m, err := mark.Circle(a)
			if err != nil {
				h.Error(c, fmt.Sprintf(":warning: %s", err))
				return
			}
			marks = append(marks, m...)
			continue
		}

		if strings.HasPrefix(a, "cone(") && strings.HasSuffix(a, ")") {
			log.Debugf("trying out cone %q", a)
			m, err := marksFromCone(a)
			if err != nil {
				h.Error(c, fmt.Sprintf(":warning: %s", err))
				return
			}
			marks = append(marks, m...)
			continue
		}

		if color, err := colors.ToColor(a); err == nil {
			// paint the squares the color
			for _, m := range marks {
				m = m.WithColor(color)
				coloredMarks = append(coloredMarks, m)
			}
			// reset the list of squares
			marks = []mark.Mark{}
			continue
		}

		h.Error(c, fmt.Sprintf(":warning: I couldn't figure out what you mean by `%s`.\n%s", a, usage))
		return
	}

	if len(marks) != 0 {
		h.Error(c, ":warning: A list of marks should always end with a color!")
		return
	}

	if cmdName == "mark" {
		for _, m := range coloredMarks {
			c.Context.Mark(*tabId, m)
		}
		if err := c.Context.Save(); err != nil {
			log.Errorf("saving marks: %s", err)
			h.Error(c, ":warning: A problem occurred while saving your marks. This could indicate an bug.")
		}

		h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
	} else {
		h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab.WithMarks(coloredMarks)))
	}
}

func distance(a image.Point, cornerA string, b image.Point, cornerB string) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = dx * -1
	}

	dy := a.Y - b.Y
	if dy < 0 {
		dy = dy * -1
	}

	diags := 0
	straights := 0
	if (cornerA[0] == cornerB[0]) != (cornerA[1] == cornerB[1]) {
		straights++
	} else if cornerA[0] != cornerB[0] {
		diags++
	}

	if dx < dy {
		diags = dx
		straights = dy - dx
	} else {
		diags = dy
		straights = dx - dy
	}
	return straights*5 + diags/2*15 + diags%2*5
}

func marksFromCircle(in string) (out []mark.Mark, err error) {
	out = []mark.Mark{}
	args := strings.Split(in[7:len(in)-1], ",")
	if len(args) != 2 {
		return nil, fmt.Errorf("in `%s`, `circle()` expects two comma-separated arguments", in)
	}
	center, dir, err := conv.RCToPoint(args[0], true)
	if err != nil {
		return nil, fmt.Errorf("`%s` looked like a circle, but could not parse coordinate `%s`: %s", in, args[0], err)
	}

	radius, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, fmt.Errorf("`%s` looked like a circle, but could not parse radius `%s`: %s", in, args[1], err)
	}

	if len(dir) == 1 {
		return nil, fmt.Errorf("`%s` specifies an edge, not a square or corner; but circles centered on edges are no valid.", args[1])
	}

	if len(dir) == 0 {
		for x := -radius / 5; x <= radius/5; x++ {
			for y := -radius / 5; y <= radius/5; y++ {
				pt := image.Point{center.X + x, center.Y + y}
				if distance(pt, "", center, "") <= radius {
					out = append(out, mark.Mark{Point: pt})
				}
			}
		}
	} else {
		switch dir {
		case "nw":
			center.X++
		case "sw":
			center.X++
			center.Y--
		case "se":
			center.Y--
		}

		for x := -radius/5 - 1; x <= radius/5+1; x++ {
		coord:
			for y := -radius/5 - 1; y <= radius/5+1; y++ {
				pt := image.Point{center.X + x, center.Y + y}
				for _, corner := range []string{"ne", "nw", "sw", "se"} {
					if distance(pt, corner, center, dir) > radius {
						continue coord
					}
				}
			}
		}
	}

	return out, nil
}

var coneAngles = map[string][]float64{
	"e":  []float64{0, 1, 7, 8},
	"ne": []float64{0, 2},
	"n":  []float64{1, 3},
	"nw": []float64{2, 4},
	"w":  []float64{3, 5},
	"sw": []float64{4, 6},
	"s":  []float64{5, 7},
	"se": []float64{6, 8},
}

var specialCones = map[string]map[int][]image.Point{
	"n": map[int][]image.Point{
		15: []image.Point{
			image.Pt(0, -1), image.Pt(-1, -2), image.Pt(0, -2), image.Pt(1, -2), image.Pt(-1, -3), image.Pt(0, -3), image.Pt(1, -3),
		},
	},
	"s": map[int][]image.Point{
		15: []image.Point{
			image.Pt(0, 1), image.Pt(-1, 2), image.Pt(0, 2), image.Pt(1, 2), image.Pt(-1, 3), image.Pt(0, 3), image.Pt(1, 3),
		},
	},
	"e": map[int][]image.Point{
		15: []image.Point{
			image.Pt(1, 0), image.Pt(2, -1), image.Pt(2, 0), image.Pt(2, 1), image.Pt(3, -1), image.Pt(3, 0), image.Pt(3, 1),
		},
	},
	"w": map[int][]image.Point{
		15: []image.Point{
			image.Pt(-1, 0), image.Pt(-2, -1), image.Pt(-2, 0), image.Pt(-2, 1), image.Pt(-3, -1), image.Pt(-3, 0), image.Pt(-3, 1),
		},
	},
}

func angle(a image.Point, cA string, b image.Point, cB string) float64 {
	switch cA {
	case "nw":
		a.X--
	case "sw":
		a.X--
		a.Y++
	case "se":
		a.Y++
	}
	switch cB {
	case "nw":
		b.X--
	case "sw":
		b.X--
		b.Y++
	case "se":
		b.Y++
	}
	if a == b {
		return 0
	}
	return math.Atan2(float64(b.Y-a.Y), float64(b.X-a.X))
}

func marksFromCone(in string) (out []mark.Mark, err error) {
	out = []mark.Mark{}
	args := strings.Split(in[5:len(in)-1], ",")
	if len(args) != 3 {
		return nil, fmt.Errorf("in `%s`, `cone()` expects three comma-separated arguments", in)
	}
	origin, corner, err := conv.RCToPoint(args[0], true)
	if err != nil {
		return nil, fmt.Errorf("`%s` looked like a cone, but could not parse coordinate `%s`: %s", in, args[0], err)
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
		return nil, fmt.Errorf("`%s` looked like a cone, but could not parse radius `%s`: %s", in, args[1], err)
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
			angles := []float64{
				angle(origin, corner, image.Pt(x, y), "ne"),
				angle(origin, corner, image.Pt(x, y), "nw"),
				angle(origin, corner, image.Pt(x, y), "sw"),
				angle(origin, corner, image.Pt(x, y), "se"),
			}
			for _, angle := range angles {
				if angle < 0 {
					angle += 2 * math.Pi
				}
				ok := false
				for angleIdx := 0; angleIdx < len(angleRange); angleIdx += 2 {
					if angle >= angleRange[angleIdx]*math.Pi/4 && angle <= angleRange[angleIdx+1]*math.Pi/4 {
						ok = true
					}
				}
				if !ok {
					continue coord
				}
			}
			// and all four corners must be withn the right range
			for _, ptC := range []string{"ne", "nw", "sw", "se"} {
				if distance(image.ZP, corner, image.Pt(x, y), ptC) > radius {
					continue coord
				}
			}
			out = append(out, mark.Mark{Point: origin.Add(image.Pt(x, y))})
		}
	}

	return out, nil
}

func marksFromSquare(in string) (out []mark.Mark, err error) {
	out = []mark.Mark{}
	args := strings.Split(in[7:len(in)-1], ",")
	if len(args) != 2 {
		return nil, fmt.Errorf("in `%s`, `square()` expects two comma-separated arguments", in)
	}

	min, _, err := conv.RCToPoint(args[0], false)
	if err != nil {
		return nil, fmt.Errorf("`%s` looked like a square, but could not parse coordinate `%s`: %s", in, args[0], err)
	}

	max, _, err := conv.RCToPoint(args[1], false)
	if err != nil {
		return nil, fmt.Errorf("`%s` looked like a square, but could not parse coordinate `%s`: %s", in, args[1], err)
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
