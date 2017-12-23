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
		h      HorizontalAlignment
		v      VerticalAlignment
		points map[int]map[int]bool
	}

	const gsize = 2
	glyph := image.NewRGBA(image.Rect(0, 0, gsize, gsize))
	for i := 0; i < gsize; i++ {
		glyph.Set(i, i, color.White)
	}

	for _, size := range []int{3, 4, 19, 50} {
		half := size/2 - 1
		max := size - 1

		tests := []test{
			{Left, Top, map[int]map[int]bool{0: map[int]bool{0: true}, 1: map[int]bool{1: true}}},
			{Center, Top, map[int]map[int]bool{half: map[int]bool{0: true}, half + 1: map[int]bool{1: true}}},
			{Right, Top, map[int]map[int]bool{max - 1: map[int]bool{0: true}, max: map[int]bool{1: true}}},
			{Left, Middle, map[int]map[int]bool{0: map[int]bool{half: true}, 1: map[int]bool{half + 1: true}}},
			{Center, Middle, map[int]map[int]bool{half: map[int]bool{half: true}, half + 1: map[int]bool{half + 1: true}}},
			{Right, Middle, map[int]map[int]bool{max - 1: map[int]bool{half: true}, max: map[int]bool{half + 1: true}}},
			{Left, Bottom, map[int]map[int]bool{0: map[int]bool{max - 1: true}, 1: map[int]bool{max: true}}},
			{Center, Bottom, map[int]map[int]bool{half: map[int]bool{max - 1: true}, half + 1: map[int]bool{max: true}}},
			{Right, Bottom, map[int]map[int]bool{max - 1: map[int]bool{max - 1: true}, max: map[int]bool{max: true}}},
		}

		for n, test := range tests {
			aligned := align(glyph, size, size, test.h, test.v)
			for x := 0; x < size; x++ {
				for y := 0; y < size; y++ {
					should := color.RGBA{0, 0, 0, 0}
					if test.points[x][y] {
						should = color.RGBA{255, 255, 255, 255}
					}
					if aligned.At(x, y) != should {
						for y := 0; y < size; y++ {
							for x := 0; x < size; x++ {
								if aligned.At(x, y) == (color.RGBA{0, 0, 0, 0}) {
									fmt.Print(".")
								} else {
									fmt.Print("#")
								}
							}
							fmt.Print("    ")
							for x := 0; x < size; x++ {
								if test.points[x][y] {
									fmt.Print("#")
								} else {
									fmt.Print(".")
								}
							}
							fmt.Print("\n")
						}
						fmt.Print("\n")
						t.Fatalf("aligned-at #%d (v=%v, h=%v) (%d,%d) was %v, not %v", n, test.v, test.h, x, y, aligned.At(x, y), should)
					}
				}
			}
		}
	}
}
