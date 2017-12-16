package conv

import (
	"errors"
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
)

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
