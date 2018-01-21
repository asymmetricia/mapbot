package colors

import (
	"testing"
)

func TestToColor(t *testing.T) {
	type test struct {
		in         string
		r, g, b, a uint8
		err        bool
	}

	tests := []test{
		{"red", 0xE6, 0x33, 0x5F, 0x7F, false},
		{"solidred", 0xE6, 0x33, 0x5F, 0xFF, false},
		{"lightred", 0xE6, 0x33, 0x5F, 0x3F, false},
		{"#ffffff", 0xff, 0xff, 0xff, 0xff, false},
		{"#ffffff7f", 0xff, 0xff, 0xff, 0x7f, false},
		{"#fffff", 0, 0, 0, 0, true},
		{"#fffffff", 0, 0, 0, 0, true},
		{"reed", 0, 0, 0, 0, true},
		{"#reeeed", 0, 0, 0, 0, true},
	}

	for testN, test := range tests {
		res, err := ToColor(test.in)
		if err != nil != test.err {
			if test.err {
				t.Fatalf("test %d: expected non-nil err, but received nil", testN)
			} else {
				t.Fatalf("test %d: expected nil err, but received %q", err)
			}
		}

		if res.R != test.r {
			t.Fatalf("test %d: expected red=%x but was %x", testN, test.r, res.R)
		}

		if res.G != test.g {
			t.Fatalf("test %d: expected green=%x but was %x", testN, test.g, res.G)
		}

		if res.B != test.b {
			t.Fatalf("test %d: expected blue=%x but was %x", testN, test.b, res.B)
		}

		if res.A != test.a {
			t.Fatalf("test %d: expected alpha=%x but was %x", testN, test.a, res.A)
		}
	}
}
