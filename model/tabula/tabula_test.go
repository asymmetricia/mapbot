package tabula

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

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
