package conv

import (
	"errors"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"image"
	"regexp"
	"strconv"
	"strings"
)

var log = mbLog.Log

var xCoordRe = regexp.MustCompile(`^[a-z]+$`)
var yCoordRe = regexp.MustCompile(`^[0-9]+$`)
var coordRe = regexp.MustCompile(`^([a-z]+)([0-9]+)(n|ne|e|se|s|sw|w|nw)?$`)

func RCToPoint(rc string, directionAllowed bool) (point image.Point, direction string, err error) {
	rc = strings.ToLower(rc)
	if matches := coordRe.FindStringSubmatch(rc); matches != nil && len(matches) >= 3 {
		if !directionAllowed && len(matches) > 3 && matches[3] != "" {
			return image.Point{}, "", errors.New("direction not allowed in this context")
		}

		point, err = CoordsToPoint(matches[1], matches[2])
		if len(matches) > 3 {
			direction = matches[3]
		}
		return
	} else {
		return image.Point{}, "", errors.New("not an RC coordinate")
	}
}

func CoordsToPoint(x, y string) (image.Point, error) {
	x = strings.ToLower(x)
	if !xCoordRe.MatchString(x) {
		return image.Point{}, errors.New("X coordinate must be a column letter")
	}

	y = strings.ToLower(y)
	if !yCoordRe.MatchString(y) {
		return image.Point{}, errors.New("Y coordinate must be a number")
	}

	accumX := 0
	for i := 0; i < len(x); i++ {
		accumX = accumX*26 + int(x[i]) - int('a') + 1
	}

	accumY, err := strconv.Atoi(y)
	if err != nil {
		return image.Point{}, fmt.Errorf("invalid Y coordinate: %s", err)
	}
	return image.Point{accumX - 1, accumY - 1}, nil
}

// Distance calculates the "pathfinder-style" distance between two points;
// i.e., nourth/south count as 5 feet, odd-numbered diagonals are 5, and
// even-numbered diagnols are 10.
func Distance(a, b image.Point) int {
	return DistanceCorners(a, "", b, "")
}

func DistanceCorners(a image.Point, cornerA string, b image.Point, cornerB string) int {
	if len(cornerA) != len(cornerB) {
		log.Warningf("incalculable distance from %q to %q", cornerA, cornerB)
		return -1
	}

	if len(cornerA) != 0 && len(cornerA) != 2 {
		log.Warningf("invalid distances %q and %q", cornerA, cornerB)
		return -1
	}

	cdx := 0
	cdy := 0

	// (0,0)se -> (1,1)nw == 10 (squares) - 5 (dx>0 and s->n) - 5 (dx>0 and e->w) == 0
	if len(cornerA) == 2 && cornerA != cornerB {
		if cornerA[1] != cornerB[1] {
			if cornerA[1] == 'e' {
				cdx--
			} else {
				cdx++
			}
		}
		if cornerA[0] != cornerB[0] {
			if cornerA[0] == 'n' {
				cdy++
			} else {
				cdy--
			}
		}
	}

	dx := b.X - a.X + cdx
	if dx < 0 {
		dx = dx * -1
	}

	dy := b.Y - a.Y + cdy
	if dy < 0 {
		dy = dy * -1
	}

	diags := 0
	straights := 0
	if dx < dy {
		diags = dx
		straights = dy - dx
	} else {
		diags = dy
		straights = dx - dy
	}

	return straights*5 + diags/2*15 + diags%2*5
}
