package types

import (
	"database/sql/driver"
	"image/color"
)

type TabulaId int64

func (i *TabulaId) Value() (driver.Value, error) {
	return int64(*i), nil
}

var _ driver.Valuer = (*TabulaId)(nil)

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

type ContextId string

func (c ContextId) Value() (driver.Value, error) {
	return string(c), nil
}

var _ driver.Valuer = (*ContextId)(nil)
