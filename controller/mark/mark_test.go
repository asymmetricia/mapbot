package mark

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"
)

func TestMarksFromCone(t *testing.T) {
	type test struct {
		input   string
		squares map[image.Point]bool
	}

	tests := []test{
		{"cone(a1ne,e,15)", map[image.Point]bool{
			image.Pt(1, 0):  true,
			image.Pt(2, 0):  true,
			image.Pt(3, 0):  true,
			image.Pt(2, -1): true,
			image.Pt(3, -1): true,
			image.Pt(2, 1):  true,
			image.Pt(3, 1):  true,
		}},
	}

	for _, test := range tests {
		marks, err := marksFromCone(test.input)
		if err != nil {
			t.Fatalf("%q: expected non-nil error, got %s", test.input, err)
		}
		for _, mark := range marks {
			if _, ok := test.squares[mark.Point]; !ok {
				t.Fatalf("%q: square %q not expected", test.input, mark.Point)
			}
			delete(test.squares, mark.Point)
		}
		if len(test.squares) > 0 {
			sq := []string{}
			for s := range test.squares {
				sq = append(sq, fmt.Sprintf("%v", s))
			}
			t.Fatalf("%q: %d square(s) missing from cone: %s", test.input, len(sq), strings.Join(sq, ","))
		}
	}
}

func TestAngle(t *testing.T) {
	type test struct {
		From   image.Point
		FromC  string
		To     image.Point
		ToC    string
		result float64
	}
	tests := []test{
		{image.Pt(0, 0), "ne", image.Pt(1, 0), "ne", 0},
		{image.Pt(0, 0), "ne", image.Pt(0, 0), "se", 6 * math.Pi / 4},
		{image.Pt(0, 0), "ne", image.Pt(1, 0), "nw", math.NaN()},
	}

	for _, test := range tests {
		res := angle(test.From, test.FromC, test.To, test.ToC)
		if math.IsNaN(res) && math.IsNaN(test.result) {
			continue
		}
		if res == test.result {
			continue
		}

		t.Fatalf("expected angle(%v,%s,%v,%s) == %f, but was %f", test.From, test.FromC, test.To, test.ToC, test.result, res)
	}
}
