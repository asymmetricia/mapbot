package types

import (
	"image/color"
	"image"
)

type TabulaName string

type Tabula struct {
	Id         *TabulaId
	Name       TabulaName
	Url        string
	Background *image.RGBA
	OffsetX    int
	OffsetY    int
	Dpi        float32
	GridColor  *color.NRGBA
	Masks      map[string]*Mask
}

type TabulaId int64

type Context struct {
	Id           string
	ActiveTabula *TabulaId
	Tokens       map[string]image.Point
}

type Mask struct {
	Name   string
	Color  color.NRGBA
	Order  *int
	Top    int
	Left   int
	Width  int
	Height int
	Clear  bool
}