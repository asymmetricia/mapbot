package tabula

import (
	"image"
	"errors"
	"image/color"
)

type Tabula struct {
	Background *image.RGBA
	OffsetX    int
	OffsetY    int
	DpiX       int
	DpiY       int
	Masks      []interface{}
	Overlays   []interface{}
}

func (t *Tabula) Render() (image.Image, error) {
	if t.DpiX == 0 || t.DpiY == 0 {
		return nil, errors.New("cannot render tabula with zero DPI")
	}

	if t.Background == nil {
		t.Background = image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	}

	bounds := t.Background.Bounds()

	gridded := copyImage(t.Background)

	// Vertical lines; X at DPI intervals, all Y
	for x := t.OffsetX; x < bounds.Max.X; x += t.DpiX {
		for y := 0; y < bounds.Max.Y; y++ {
			gridded.Set(x, y, color.Black)
		}
	}

	// Horizontal lines; X at DPI intervals, all Y
	for x := 0; x < bounds.Max.X; x++ {
		for y := t.OffsetY; y < bounds.Max.Y; y += t.DpiY {
			gridded.Set(x, y, color.Black)
		}
	}

	return gridded, nil
}

func copyImage(in *image.RGBA) *image.RGBA {
	out := &image.RGBA{
		Pix: make([]uint8, len(in.Pix)),
		Stride: in.Stride,
		Rect: image.Rect(
			in.Bounds().Min.X,
			in.Bounds().Min.Y,
			in.Bounds().Max.X,
			in.Bounds().Max.Y,
		),
	}

	copy(out.Pix, in.Pix)

	return out
}