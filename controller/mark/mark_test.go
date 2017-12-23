package mark

import (
	"fmt"
	"image"
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
