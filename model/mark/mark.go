package mark

import (
	"image"
	"image/color"
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
