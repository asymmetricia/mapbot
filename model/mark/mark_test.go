package mark

import (
	"fmt"
	"image"
	"testing"
)

func TestCircle(t *testing.T) {
	type test struct {
		input  string
		output []image.Point
	}

	tests := []test{
		{"circle(a1,0)", []image.Point{image.Pt(0, 0)}},
		{"circle(b2,5)", []image.Point{
			{0, 0}, {1, 0}, {2, 0},
			{0, 1}, {1, 1}, {2, 1},
			{0, 2}, {1, 2}, {2, 2},
		}},
		{"circle(c3,10)", []image.Point{
			/*   */ {1, 0}, {2, 0}, {3, 0},

			{0, 1}, {1, 1}, {2, 1}, {3, 1}, {4, 1},

			{0, 2}, {1, 2}, {2, 2}, {3, 2}, {4, 2},

			{0, 3}, {1, 3}, {2, 3}, {3, 3}, {4, 3},

			/*   */ {1, 4}, {2, 4}, {3, 4},
		}},
		{"circle(a1se,0)", []image.Point{}},
		{"circle(a1se,5)", []image.Point{
			{0, 0}, {1, 0},

			{0, 1}, {1, 1},
		}},
		{"circle(b2se,10)", []image.Point{
			/*   */ {1, 0}, {2, 0},

			{0, 1}, {1, 1}, {2, 1}, {3, 1},

			{0, 2}, {1, 2}, {2, 2}, {3, 2},

			/*   */ {1, 3}, {2, 3},
		}},
		{"circle(c3se,15)", []image.Point{
			/*           */ {2, 0}, {3, 0},

			/*   */ {1, 1}, {2, 1}, {3, 1}, {4, 1},

			{0, 2}, {1, 2}, {2, 2}, {3, 2}, {4, 2}, {5, 2},

			{0, 3}, {1, 3}, {2, 3}, {3, 3}, {4, 3}, {5, 3},

			/*   */ {1, 4}, {2, 4}, {3, 4}, {4, 4},

			/*           */ {2, 5}, {3, 5},
		}},
		{"circle(d4se,20)", []image.Point{
			/*                   */ {3, 0}, {4, 0},

			/*   */ {1, 1}, {2, 1}, {3, 1}, {4, 1}, {5, 1}, {6, 1},

			/*   */ {1, 2}, {2, 2}, {3, 2}, {4, 2}, {5, 2}, {6, 2},

			{0, 3}, {1, 3}, {2, 3}, {3, 3}, {4, 3}, {5, 3}, {6, 3}, {7, 3},

			{0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}, {5, 4}, {6, 4}, {7, 4},

			/*   */ {1, 5}, {2, 5}, {3, 5}, {4, 5}, {5, 5}, {6, 5},

			/*   */ {1, 6}, {2, 6}, {3, 6}, {4, 6}, {5, 6}, {6, 6},

			/*                   */ {3, 7}, {4, 7},
		}},
	}

	for _, test := range tests {
		res, err := Circle(test.input)
		if err != nil {
			t.Fatalf("%q: expected non-nil err, produced %q", test.input, err)
		}

	outputs:
		for _, pt := range test.output {
			for _, mark := range res {
				if mark.Point == pt {
					continue outputs
				}
			}
			t.Fatalf("%q: point %v should have been marked but was not", test.input, pt)
		}

	marks:
		for _, mark := range res {
			for _, pt := range test.output {
				if mark.Point == pt {
					continue marks
				}
			}
			t.Fatalf("%q: point %v in output should not have been marked", test.input, mark.Point)
		}
		fmt.Printf("%q: correctly produced the expected %d marks\n", test.input, len(res))
	}
}
