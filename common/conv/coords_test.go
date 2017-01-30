package conv

import "testing"

func TestCoordsToPoint(t *testing.T) {
	_, err := CoordsToPoint("-", "5")

	if err == nil {
		t.Fatal(`Expected CoordsToPoint("-", "5") to return err, returned nil`)
	}

	type Test struct {
		x string
		y string
		resX int
		resY int
	}
	tests := []Test{
		Test{"a", "5", 1, 5},
		Test{"Aa", "55", 27, 55},
		Test{"Z", "999", 26, 999},
		Test{"ZZ", "1", 702, 1},
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
