package draw

import (
	"image"
	"image/draw"
)

// Pad returns a new image the size of `bounds` with the image `in` draw on it.
// If the lower bounds of `bounds` are negative, `in` will be shifted down and
// right.
func Pad(in draw.Image, bounds image.Rectangle) draw.Image {
	dstR := image.Rectangle{}
	srcPt := image.Point{}

	// If Xmin is negative, we'll be padding on the left by Xmin pixels; and
	// the image will thus be shifted right by Xmin pixels.
	if bounds.Min.X <= 0 {
		dstR.Min.X = -bounds.Min.X
		dstR.Max.X = dstR.Min.X + bounds.Dx()
	} else {
		// If Xmin is positive, the image will be shifted off the view.
		srcPt.X = bounds.Min.X
		dstR.Max.X = bounds.Dx()
	}

	if bounds.Min.Y <= 0 {
		dstR.Min.Y = -bounds.Min.Y
		dstR.Max.Y = dstR.Min.Y + bounds.Dy()
	} else {
		srcPt.Y = bounds.Min.Y
		dstR.Max.Y = bounds.Dy()
	}

	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, dstR, in, srcPt, draw.Over)
	return out
}
