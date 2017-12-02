package colors

import (
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"
)

var Colors = map[string]color.NRGBA{
	"red":    {0xE6, 0x33, 0x5F, 0x7F},
	"green":  {0x01, 0xA3, 0x68, 0x7F},
	"blue":   {0x00, 0x6A, 0x93, 0x7F},
	"orange": {0xFF, 0x86, 0x1F, 0x7F},
	"yellow": {0xFB, 0xE8, 0x70, 0x7F},
	"purple": {0x83, 0x59, 0xA3, 0x7F},
	"violet": {0x83, 0x59, 0xA3, 0x7F},
	"brown":  {0xA3, 0x6F, 0x40, 0x7F},
	"black":  {0x00, 0x00, 0x00, 0x7F},
	"white":  {0xFF, 0xFF, 0xFF, 0x7F},
	"gray":   {0x8B, 0x86, 0x80, 0x7F},
	"grey":   {0x8B, 0x86, 0x80, 0x7F},
	"pink":   {0xCD, 0x91, 0x9E, 0x7F},
}

var hexColorRe = regexp.MustCompile(`^#?[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$`)

// ToColor returns the color.NRGBA corresponding to the named color; or if the
// color is a hex code, returns a color.NRGBA corresponding to those values.
// Hex codes may be three bytes (six characters) or four bytes (eight
// characters); in the latter case, the final byte represents the alpha channel.
func ToColor(name string) (color.NRGBA, error) {
	if namedColor, ok := Colors[strings.ToLower(name)]; ok {
		return namedColor, nil
	}

	if !hexColorRe.MatchString(name) {
		return color.NRGBA{}, fmt.Errorf("I don't know of a color named %q, and that doesn't look like a hex color code", name)
	}

	name = strings.TrimLeft(name, "#")

	var r, g, b, a uint64
	var err error

	r, err = strconv.ParseUint(name[0:2], 16, 8)

	if err == nil {
		g, err = strconv.ParseUint(name[2:4], 16, 8)
	}

	if err == nil {
		b, err = strconv.ParseUint(name[4:6], 16, 8)
	}

	a = 0xFF
	if len(name) == 8 && err == nil {
		a, err = strconv.ParseUint(name[6:8], 16, 8)
	}

	if err != nil {
		return color.NRGBA{}, fmt.Errorf("`%s` looks like a hex color, but I can't parse it: %s", name, err)
	}

	return color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
}
