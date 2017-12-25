package draw

import (
	"image"
	"image/color"
	"testing"
)

func TestBlend(t *testing.T) {
	a := &color.NRGBA{255, 0, 0, 128}
	b := &color.NRGBA{0, 255, 0, 128}
	c := Blend(a, b)
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
	BlendAt(img, 0, 0, &color.NRGBA{0, 255, 0, 128})
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
