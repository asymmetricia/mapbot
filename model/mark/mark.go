package mark

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/conv"
	"image"
	"image/color"
	"regexp"
	"strconv"
	"strings"
)

type Mark struct {
	Point     image.Point
	Direction string
	Color     color.Color
}

func (m Mark) WithColor(c color.Color) (ret Mark) {
	ret = m
	ret.Color = c
	return ret
}

func Circle(in string) (out []Mark, err error) {
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
	out, err = CirclePoint(center, dir, radius)
	if err != nil {
		return nil, fmt.Errorf("`%s` looked like a circle, but: %s", in, err)
	}
	return out, nil
}

var dirRe = regexp.MustCompile(`^(n|ne|e|se|s|sw|w|nw)?$`)

func CirclePoint(center image.Point, dir string, radius int) (out []Mark, err error) {
	out = []Mark{}

	dir = strings.ToLower(dir)

	if !dirRe.MatchString(dir) {
		return nil, fmt.Errorf("could not parse direction `%s`", dir)
	}

	if len(dir) == 1 {
		return nil, fmt.Errorf("`%s` specifies an edge, not a square or corner, and circles centered on edges are not valid.", dir)
	}

	if len(dir) == 0 {
		for x := -radius / 5; x <= radius/5; x++ {
			for y := -radius / 5; y <= radius/5; y++ {
				pt := image.Point{center.X + x, center.Y + y}
				if conv.Distance(pt, center) <= radius {
					out = append(out, Mark{Point: pt})
				}
			}
		}
	} else {
		for x := -radius/5 - 1; x <= radius/5+1; x++ {
			for y := -radius/5 - 1; y <= radius/5+1; y++ {
				pt := image.Point{center.X + x, center.Y + y}
				if conv.DistanceCorners(center, dir, pt, "ne") <= radius &&
					conv.DistanceCorners(center, dir, pt, "se") <= radius &&
					conv.DistanceCorners(center, dir, pt, "sw") <= radius &&
					conv.DistanceCorners(center, dir, pt, "nw") <= radius {
					out = append(out, Mark{Point: pt})
				}
			}
		}
	}

	return out, nil
}
