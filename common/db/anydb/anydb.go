package anydb

import "database/sql"

type AnyDb interface {
	Dialect() string
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	Begin() (*sql.Tx, error)
}
