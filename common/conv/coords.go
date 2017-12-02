package conv

import (
	"errors"
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
)

var xCoordRe = regexp.MustCompile(`^[a-zA-Z]+$`)
var yCoordRe = regexp.MustCompile(`^[0-9]+$`)
var coordRe = regexp.MustCompile(`^([a-zA-Z]+)([0-9]+)$`)

func RCToPoint(rc string) (image.Point, error) {
	if matches := coordRe.FindStringSubmatch(rc); matches != nil && len(matches) >= 3 {
		return CoordsToPoint(matches[1], matches[2])
	} else {
		return image.Point{}, errors.New("not an RC coordinate")
	}
}

func CoordsToPoint(x, y string) (image.Point, error) {
	if !xCoordRe.MatchString(x) {
		return image.Point{}, errors.New("X coordinate must be a column letter")
	}

	if !yCoordRe.MatchString(y) {
		return image.Point{}, errors.New("Y coordinate must be a number")
	}

	accumX := 0
	x = strings.ToUpper(x)
	for i := 0; i < len(x); i++ {
		accumX = accumX*26 + int(x[i]) - int('A') + 1
	}

	accumY, err := strconv.Atoi(y)
	if err != nil {
		return image.Point{}, fmt.Errorf("invalid Y coordinate: %s", err)
	}
	return image.Point{accumX - 1, accumY - 1}, nil
}
