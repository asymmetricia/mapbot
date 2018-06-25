package types

import (
	"database/sql/driver"
)

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
