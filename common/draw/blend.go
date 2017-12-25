package draw

import (
	"image/color"
	"image/draw"
)

// BlendAt alters the point (given by (x,y)) in the image i by blending the color there with the color c
func BlendAt(i draw.Image, x, y int, c color.Color) {
	i.Set(x, y, Blend(c, i.At(x, y)))
}

// blend calculates the result of alpha blending of the two colors
func Blend(a color.Color, b color.Color) color.Color {
	a_r, a_g, a_b, a_a := a.RGBA()
	b_r, b_g, b_b, b_a := b.RGBA()
	return &color.RGBA{
		R: uint8((a_r + b_r*(0xFFFF-a_a)/0xFFFF) >> 8),
		G: uint8((a_g + b_g*(0xFFFF-a_a)/0xFFFF) >> 8),
		B: uint8((a_b + b_b*(0xFFFF-a_a)/0xFFFF) >> 8),
		A: uint8((a_a + b_a*(0xFFFF-a_a)/0xFFFF) >> 8),
	}
}
