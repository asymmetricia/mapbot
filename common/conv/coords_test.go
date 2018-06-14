package conv

import (
	"image"
	"strings"
	"testing"
)

type coordTest struct {
	x, y       string
	resX, resY int
}

var coordTests = []coordTest{
	{"a", "1", 0, 0},
	{"a", "-1", 0, -1},
	{"a", "-10", 0, -10},
	{"-b", "1", -1, 0},
	{"-b", "-1", -1, -1},
	{"-b", "-10", -1, -10},
	{"a", "5", 0, 4},
	{"Ba", "55", 26, 54},
	{"Z", "999", 25, 998},
	{"ZZ", "1", 675, 0},
	{"-ZZ", "1", -675, 0},
}

func TestCoordsToPoint(t *testing.T) {
	_, err := CoordsToPoint("-", "5")

	if err == nil {
		t.Fatal(`Expected CoordsToPoint("-", "5") to return err, returned nil`)
	}

	for _, test := range coordTests {
		pt, err := CoordsToPoint(test.x, test.y)
		if err != nil {
			t.Fatalf(`Expected CoordsToPoint("%s", "%s") to return nil, returned %q`, test.x, test.y, err)
		}

		if pt.X != test.resX {
			t.Fatalf(`Expected CoordsToPoint("%s", "%s") to return image.Point{%d,_}, returned image.Point{%d,_}`, test.x, test.y, test.resX, pt.X)
		}

		if pt.Y != test.resY {
			t.Fatalf(`Expected CoordsToPoint("%s", "%s") to return image.Point{_,%d}, returned image.Point{_,%d}`, test.x, test.y, test.resY, pt.Y)
		}
		t.Logf("CoordsToPoint(%s,%s) -> (%d,%d): OK", test.x, test.y, pt.X, pt.Y)
	}
}

func TestRCToPoint(t *testing.T) {
	type Test struct {
		inp      string
		allowDir bool
		resErr   bool
		resX     int
		resY     int
		resDir   string
	}
	tests := []Test{
		{"a1", true, false, 0, 0, ""},
		{"z25", true, false, 25, 25, ""},
		{"b2n", true, false, 1, 1, "n"},
		{"a2sw", true, false, 0, 1, "sw"},
		{"a2SW", true, false, 0, 1, "sw"},
		{"a1", false, false, 0, 0, ""},
		{"z25", false, false, 25, 25, ""},
		{"b2n", false, true, 0, 0, ""},
		{"a2sw", false, true, 0, 0, ""},
	}

	for _, test := range tests {
		pt, dir, err := RCToPoint(test.inp, test.allowDir)

		if test.resErr && err == nil {
			t.Fatalf(`Expected RCToPoint("%s") to return non-nil error, returned nil`, test.inp)
		}

		if !test.resErr && err != nil {
			t.Fatalf(`Expected RCToPoint("%s") to return nil error, returned %q`, test.inp, err)
		}

		if pt.X != test.resX {
			t.Fatalf(`Expected RCToPoint("%s") to return image.Point{%d,_}, returned %v`, test.inp, test.resX, pt)
		}

		if pt.X != test.resX {
			t.Fatalf(`Expected RCToPoint("%s") to return image.Point{_,%d}, returned %v`, test.inp, test.resY, pt)
		}

		if dir != test.resDir {
			t.Fatalf(`Expected RCToPoint("%s") to return direction %q, returned %q`, test.inp, test.resDir, dir)
		}

		t.Logf("RCToPoint(%s) -> (%d,%d,%q): OK", test.inp, test.resX, test.resY, test.resDir)
	}
}

func TestDistanceCorners(t *testing.T) {
	type test struct {
		FromPt image.Point
		FromC  string
		ToPt   image.Point
		ToC    string
		Result int
	}

	tests := []test{
		{image.Pt(0, 0), "nw", image.Pt(0, 0), "nw", 0},
		{image.Pt(0, 0), "nw", image.Pt(0, 0), "ne", 5},
		{image.Pt(0, 0), "nw", image.Pt(0, 0), "se", 5},
		{image.Pt(0, 0), "nw", image.Pt(0, 0), "sw", 5},
		{image.Pt(0, 0), "nw", image.Pt(1, 1), "se", 15},
		{image.Pt(0, 0), "se", image.Pt(1, 1), "nw", 0},
		{image.Pt(0, 0), "sw", image.Pt(1, 0), "se", 10},
		{image.Pt(0, 0), "sw", image.Pt(1, 0), "ne", 10},
		{image.Pt(0, 0), "sw", image.Pt(1, 0), "nw", 5},
		{image.Pt(0, 0), "sw", image.Pt(1, 0), "sw", 5},
	}

	for _, test := range tests {
		if res := DistanceCorners(test.FromPt, test.FromC, test.ToPt, test.ToC); res != test.Result {
			t.Fatalf("expected DistanceCorners(%v, %s, %v, %s) == %d, but was %d", test.FromPt, test.FromC, test.ToPt, test.ToC, test.Result, res)
		}
	}
}

func TestPointToCoords(t *testing.T) {
	for _, test := range coordTests {
		if res := PointToCoords(image.Pt(test.resX, test.resY)); strings.ToLower(res) != strings.ToLower(test.x+test.y) {
			t.Fatalf("expected PointToCoords(%d,%d) == %s%s, but was %q", test.resX, test.resY, test.x, test.y, res)
		}
	}
}
