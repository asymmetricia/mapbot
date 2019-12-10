package draw

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func round(x float64) float64 {
	return math.Floor(x + 0.5)
}

func integerPart(x float64) float64 {
	return math.Floor(x)
}

func fractionalPart(x float64) float64 {
	return x - math.Floor(x)
}

func rfractionalPart(x float64) float64 {
	return 1 - fractionalPart(x)
}

// Xiaolin Wu's line algorithm from https://www.codeproject.com/Articles/13360/Antialiasing-Wu-Algorithm#dwuln
func Line(i draw.Image, a, b image.Point, col color.Color) {
	nrgba := color.NRGBAModel.Convert(col).(color.NRGBA)
	plot := func(x, y int, c uint8) {
		c2 := nrgba
		c2.A = c
		BlendAt(i, x, y, c2)
	}

	x0 := a.X
	y0 := a.Y
	x1 := b.X
	y1 := b.Y

	if y0 > y1 {
		y0, y1 = y1, y0
		x0, x1 = x1, x0
	}

	plot(x0, y0, 255)
	plot(x1, y1, 255)

	dx := x1 - x0
	xdir := 1
	if dx < 0 {
		xdir = -1
		dx = -dx
	}

	dy := y1 - y0

	// Horizontal
	if dy == 0 {
		for dx > 0 {
			dx--
			x0 = x0 + xdir
			plot(x0, y0, 255)
		}
		return
	}

	// Vertical
	if dx == 0 {
		for y := y0; y <= y1; y++ {
			plot(x0, y, 255)
		}
		return
	}

	// Perfect diagonal
	if dx == dy {
		x := x0
		for y := y0; y <= y1; y++ {
			plot(x-1, y, 51)
			plot(x, y, 255)
			plot(x+1, y, 51)
			x += xdir
		}
		return
	}

	errAccum := uint8(0)

	if dy > dx {
		// The fraction of (256) that x should increase for each y. Guaranteed less than 256, since dx < dy. 256 * dx / dy
		errAdj := uint8(uint32(dx) << 8 / uint32(dy))
		for dy > 1 {
			// Y-major; we move up one Y at a time, increasing X whenever we move up enough to warrant it.
			dy--
			errAccumTemp := errAccum
			errAccum += errAdj
			// detect uint8 rollover
			if errAccum <= errAccumTemp {
				x0 += xdir
			}
			y0++
			plot(x0, y0, 255-errAccum)
			plot(x0+xdir, y0, errAccum)
		}
	} else {
		errAdj := uint8(uint32(dy) << 8 / uint32(dx))
		for dx > 1 {
			dx--
			errAccumTemp := errAccum
			errAccum += errAdj
			if errAccum <= errAccumTemp {
				y0++
			}
			x0 += xdir
			plot(x0, y0, 255-errAccum)
			plot(x0, y0+1, errAccum)
		}
	}
}
