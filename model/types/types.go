package types

import (
	"database/sql/driver"
	"image/color"
	"strconv"
)

type TabulaId int64

func (i *TabulaId) Value() (driver.Value, error) {
	return int64(*i), nil
}

func (i *TabulaId) String() string {
	if i == nil {
		return "nil"
	}
	return strconv.FormatInt(int64(*i), 10)
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

type ContextType string
type ContextId string

func (c ContextId) Value() (driver.Value, error) {
	return string(c), nil
}

var _ driver.Valuer = (*ContextId)(nil)

type UserId string

func (i *UserId) Value() (driver.Value, error) {
	return string(*i), nil
}

var _ driver.Valuer = (*UserId)(nil)
