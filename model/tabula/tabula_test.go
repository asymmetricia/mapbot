package tabula

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

func TestBlend(t *testing.T) {
	a := &color.NRGBA{255, 0, 0, 128}
	b := &color.NRGBA{0, 255, 0, 128}
	c := blend(a, b)
	red, green, blue, alpha := c.RGBA()
	if alpha>>8 != 192 {
		t.Fatalf("blended alpha was %d, not 190..192", alpha>>8)
	}
	if red>>8 != 128 {
		t.Fatalf("blended red was %d, not 128", red>>8)
	}
	if green>>8 != 63 {
		t.Fatalf("blended green was %d, not 63", green>>8)
	}
	if blue>>8 != 0 {
		t.Fatalf("blended blue was %d, not 0", blue>>8)
	}
}

func TestBlendAt(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, &color.NRGBA{255, 0, 0, 255})
	blendAt(img, 0, 0, &color.NRGBA{0, 255, 0, 128})
	r, g, b, a := img.At(0, 0).RGBA()
	if r != 127<<8|127 {
		t.Fatalf("blended-at r was %d, not %d", r, 127<<8|127)
	}
	if g != 128<<8|128 {
		t.Fatalf("blended-at g was %d, not %d", g, 128<<8|128)
	}
	if b != 0 {
		t.Fatalf("blended-at b was %d, not %d", b, 0)
	}
	if a != 255<<8|255 {
		t.Fatalf("blended-at a was %d, not %d", a, 255<<8|255)
	}
}

func TestAlign(t *testing.T) {
	type test struct {
		h      float32
		v      float32
		points map[int]map[int]bool
	}

	glyph := image.NewRGBA(image.Rect(0, 0, 1, 1))
	glyph.Set(0, 0, color.White)

	for _, size := range []int{3, 19, 50} {
		max := size - 1
		half := max / 2

		tests := []test{
			{0, 0, map[int]map[int]bool{0: map[int]bool{0: true}}},
			{.5, 0, map[int]map[int]bool{half: map[int]bool{0: true}}},
			{1, 0, map[int]map[int]bool{max: map[int]bool{0: true}}},
			{0, .5, map[int]map[int]bool{0: map[int]bool{half: true}}},
			{.5, .5, map[int]map[int]bool{half: map[int]bool{half: true}}},
			{1, .5, map[int]map[int]bool{max: map[int]bool{half: true}}},
			{0, 1, map[int]map[int]bool{0: map[int]bool{max: true}}},
			{.5, 1, map[int]map[int]bool{half: map[int]bool{max: true}}},
			{1, 1, map[int]map[int]bool{max: map[int]bool{max: true}}},
		}

		for n, test := range tests {
			aligned := align(glyph, size, size, test.h, test.v)
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					if aligned.At(x, y) == (color.RGBA{0, 0, 0, 0}) {
						fmt.Print(".")
					} else {
						fmt.Print("#")
					}
				}
				fmt.Print("\n")
			}
			fmt.Print("\n")
			for x := 0; x < size; x++ {
				for y := 0; y < size; y++ {
					should := color.RGBA{0, 0, 0, 0}
					if test.points[x][y] {
						should = color.RGBA{255, 255, 255, 255}
					}
					if aligned.At(x, y) != should {
						t.Fatalf("aligned-at #%d (v=%f, h=%f) (%d,%d) was %v, not %v", n, test.v, test.h, x, y, aligned.At(x, y), should)
					}
				}
			}
		}
	}
}
