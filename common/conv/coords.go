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

func CoordsToPoint(x, y string) (image.Point, error) {
	if !xCoordRe.MatchString(x) {
		return image.Point{}, errors.New("X coordinage must be a column letter")
	}

	if !yCoordRe.MatchString(y) {
		return image.Point{}, errors.New("Y coordinage must be a number")
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
	return image.Point{accumX, accumY}, nil
}
