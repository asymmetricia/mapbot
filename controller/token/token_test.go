package token

import (
	"image"
	"strings"
	"testing"
)

func TestParseMovements(t *testing.T) {
	//func parseMovements(args []string, lastToken string) (map[string][]image.Point, error) {
	type test struct {
		last string
		in   string
		err  bool
		out  map[string][]image.Point
	}

	tests := []test{
		{"", "basic a1", false, map[string][]image.Point{"basic": []image.Point{{0, 0}}}},
		{"", "a1 a1", false, map[string][]image.Point{"a1": []image.Point{{0, 0}}}},
		{"basic", "a1 a1", false, map[string][]image.Point{"basic": []image.Point{{0, 0}, {0, 0}}}},
		{"", "a1", true, nil},
		{"", "basic", true, nil},
		{"basic", "unbasic a1", false, map[string][]image.Point{"unbasic": []image.Point{{0, 0}}}},
		{"basic", "unbasic", true, nil},
	}

	for testN, test := range tests {
		out, err := parseMovements(strings.Fields(test.in), test.last)
		if (err != nil) != test.err {
			if test.err {
				t.Fatalf("test %d: expected non-nil err but was %v", testN, err)
			} else {
				t.Fatalf("test %d: expected nil err but was %v", testN, err)
			}
		}

		for token, pts := range out {
			exp, ok := test.out[token]
			if !ok {
				t.Fatalf("test %d: received token movements for %v but was not present at all in expected output", testN, token)
			}
			if len(exp) != len(pts) {
				t.Fatalf("test %d: received token movements %v for %v that did not match expected %v", testN, pts, token, exp)
			}
			for i, exp := range exp {
				if pts[i] != exp {
					t.Fatalf("test %d: received movement index %d of %v, but expected %v", testN, i, pts[i], exp)
				}
			}
		}
	}
}
