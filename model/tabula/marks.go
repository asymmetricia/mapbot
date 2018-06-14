package tabula

import (
	"errors"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/mark"
	"image"
	"image/draw"
)

func (t *Tabula) addLines(in image.Image, ctx context.Context, offset image.Point) error {
	drawable, ok := in.(draw.Image)
	if !ok {
		return errors.New("not drawable!")
	}

	for _, l := range t.Lines {
		log.Debugf("adding line %v", l)
		from := l.A
		switch l.CA {
		case "nw":
		case "ne":
			from.X++
		case "se":
			from.X++
			from.Y++
		case "sw":
			from.Y++
		}

		to := l.B
		switch l.CB {
		case "nw":
		case "ne":
			to.X++
		case "se":
			to.X++
			to.Y++
		case "sw":
			to.Y++
		}

		log.Debugf("post-cornering it's %v -> %v", from, to)
		t.line(drawable, float32(from.X), float32(from.Y), float32(to.X), float32(to.Y), l.Color, offset)
	}
	return nil
}

func (t *Tabula) addMarks(in image.Image, ctx context.Context, offset image.Point) error {
	dirMarkSlice := []mark.Mark{}
	for _, dirMarks := range ctx.GetMarks(*t.Id) {
		for _, mark := range dirMarks {
			dirMarkSlice = append(dirMarkSlice, mark)
		}
	}

	if err := t.addMarkSlice(in, dirMarkSlice, offset); err != nil {
		return err
	}

	return t.addMarkSlice(in, t.Marks, offset)
}

func (t *Tabula) addMarkSlice(in image.Image, marks []mark.Mark, offset image.Point) error {
	drawable, ok := in.(draw.Image)
	if !ok {
		return errors.New("image provided could not be used as a draw.Image")
	}

	for _, mark := range marks {
		switch mark.Direction {
		case "n":
			t.squareAtFloat(drawable, float32(mark.Point.X), float32(mark.Point.Y)-.1, float32(mark.Point.X)+1, float32(mark.Point.Y)+.1, 0, mark.Color, offset)
		case "s":
			t.squareAtFloat(drawable, float32(mark.Point.X), float32(mark.Point.Y)+.9, float32(mark.Point.X)+1, float32(mark.Point.Y)+1.1, 0, mark.Color, offset)
		case "e":
			t.squareAtFloat(drawable, float32(mark.Point.X)+.9, float32(mark.Point.Y), float32(mark.Point.X)+1.1, float32(mark.Point.Y)+1, 0, mark.Color, offset)
		case "w":
			t.squareAtFloat(drawable, float32(mark.Point.X)-.1, float32(mark.Point.Y), float32(mark.Point.X)+.1, float32(mark.Point.Y)+1, 0, mark.Color, offset)
		case "ne":
			t.squareAtFloat(drawable, float32(mark.Point.X)+.9, float32(mark.Point.Y)-.1, float32(mark.Point.X)+1.1, float32(mark.Point.Y)+.1, 0, mark.Color, offset)
		case "se":
			t.squareAtFloat(drawable, float32(mark.Point.X)+.9, float32(mark.Point.Y)+.9, float32(mark.Point.X)+1.1, float32(mark.Point.Y)+1.1, 0, mark.Color, offset)
		case "nw":
			t.squareAtFloat(drawable, float32(mark.Point.X)-.1, float32(mark.Point.Y)-.1, float32(mark.Point.X)+.1, float32(mark.Point.Y)+.1, 0, mark.Color, offset)
		case "sw":
			t.squareAtFloat(drawable, float32(mark.Point.X)-.1, float32(mark.Point.Y)+.9, float32(mark.Point.X)+.1, float32(mark.Point.Y)+1.1, 0, mark.Color, offset)
		default:
			t.squareAt(drawable, image.Rect(mark.Point.X, mark.Point.Y, mark.Point.X+1, mark.Point.Y+1), 1, mark.Color, offset)
		}
	}

	return nil
}

func (t *Tabula) WithMarks(marks []mark.Mark) *Tabula {
	t.Marks = make([]mark.Mark, len(marks))
	for i, m := range marks {
		t.Marks[i] = m
	}
	return t
}

func (t *Tabula) WithLines(lines []mark.Line) *Tabula {
	t.Lines = make([]mark.Line, len(lines))
	for i, l := range lines {
		t.Lines[i] = l
	}
	return t
}

func (t *Tabula) WithNote(note string) *Tabula {
	t.Note = note
	return t
}
