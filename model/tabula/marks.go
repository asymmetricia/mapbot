package tabula

import (
	"errors"
	"github.com/pdbogen/mapbot/model/context"
	"image"
	"image/draw"
)

func (t *Tabula) addMarks(in image.Image, ctx context.Context) error {
	drawable, ok := in.(draw.Image)
	if !ok {
		return errors.New("image provided could not be used as a draw.Image")
	}

	for pt, color := range ctx.GetMarks(*t.Id) {
		log.Debugf("marking (%d,%d) %v", pt.X, pt.Y, color)
		t.squareAt(drawable, image.Rect(pt.X, pt.Y, pt.X+1, pt.Y+1), 1, color)
	}

	return nil
}
