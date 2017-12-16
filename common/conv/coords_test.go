package conv

import "testing"

func TestCoordsToPoint(t *testing.T) {
	_, err := CoordsToPoint("-", "5")

	if err == nil {
		t.Fatal(`Expected CoordsToPoint("-", "5") to return err, returned nil`)
	}

	type Test struct {
		x    string
		y    string
		resX int
		resY int
	}
	tests := []Test{
		Test{"a", "5", 0, 4},
		Test{"Aa", "55", 26, 54},
		Test{"Z", "999", 25, 998},
		Test{"ZZ", "1", 701, 0},
	}

	for _, test := range tests {
		pt, err := CoordsToPoint(test.x, test.y)
		if err != nil {
			t.Fatalf(`Expected CoordsToPoint("%s", "%s") to return nil, returned %q`, test.x, test.y, err)
		}

		if pt.X != test.resX {
			t.Fatalf(`Expected CoordsToPoint("%s", "%s") to return image.Point{1,_}, returned image.Point{%d,_}`, test.x, test.y, pt.X)
		}

		if pt.Y != test.resY {
			t.Fatalf(`Expected CoordsToPoint("%s", "%s") to return image.Point{_,5}, returned image.Point{_,%d}`, test.x, test.y, pt.Y)
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
