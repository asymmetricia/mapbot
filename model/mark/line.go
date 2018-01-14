package mark

import (
	"image"
	"image/color"
)

type Line struct {
	A, B   image.Point
	CA, CB string
	Color  color.Color
}

func (l Line) WithColor(c color.Color) Line {
	ret := l
	ret.Color = c
	return ret
}
