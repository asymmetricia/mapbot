package draw

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

func TestLine(t *testing.T) {
	type test struct {
		Size     image.Rectangle
		From, To image.Point
		Marks    map[image.Point]uint8
	}

	tests := []test{
		{Size: image.Rect(0, 0, 3, 3), From: image.ZP, To: image.Pt(2, 0), Marks: map[image.Point]uint8{
			image.Pt(0, 0): 255, image.Pt(1, 0): 255, image.Pt(2, 0): 255,
		}},
		{Size: image.Rect(0, 0, 3, 3), From: image.ZP, To: image.Pt(2, 2), Marks: map[image.Point]uint8{
			image.Pt(0, 0): 255, image.Pt(1, 0): 51, image.Pt(2, 0): 0,
			image.Pt(0, 1): 51, image.Pt(1, 1): 255, image.Pt(2, 1): 51,
			image.Pt(0, 2): 0, image.Pt(1, 2): 51, image.Pt(2, 2): 255,
		}},
		{Size: image.Rect(0, 0, 6, 6), From: image.ZP, To: image.Pt(4, 5), Marks: map[image.Point]uint8{
			image.Pt(0, 0): 255, image.Pt(1, 0): 0,
			image.Pt(0, 1): 51, image.Pt(1, 1): 204, image.Pt(2, 1): 0,
			image.Pt(0, 2): 0, image.Pt(1, 2): 103, image.Pt(2, 2): 152, image.Pt(3, 2): 0,
		}},
		{Size: image.Rect(0, 0, 6, 6), From: image.ZP, To: image.Pt(5, 4), Marks: map[image.Point]uint8{
			image.Pt(0, 0): 255, image.Pt(1, 0): 51, image.Pt(2, 0): 0,
			image.Pt(0, 1): 0, image.Pt(1, 1): 204, image.Pt(2, 1): 103,
			image.Pt(0, 2): 0, image.Pt(1, 2): 0, image.Pt(2, 2): 152, image.Pt(3, 2): 155,
		}},
	}

	for idx, test := range tests {
		img := image.NewNRGBA(test.Size)
		Line(img, test.From, test.To, color.Black)
		for y := 0; y < test.Size.Dx(); y++ {
			for x := 0; x < test.Size.Dx(); x++ {
				fmt.Printf("% -4d", color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA).A)
			}
			fmt.Print("\n")
		}
		fmt.Print("\n")

		for pt, value := range test.Marks {
			c := color.NRGBAModel.Convert(img.At(pt.X, pt.Y)).(color.NRGBA)
			if c.A != value {
				t.Fatalf("line %d: expected pixel at %v to be alpha=%d, but was %v", idx, pt, value, img.At(pt.X, pt.Y))
			}
		}
	}
}
