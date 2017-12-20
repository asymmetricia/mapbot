// +build !darwin,!windows

package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pdbogen/mapbot/common/db/anydb"
)

type Sqlite struct {
	*sql.DB
}

func (s *Sqlite) Dialect() string {
	return "sqlite3"
}

func OpenInMemory(reset bool, resetFrom int) (anydb.AnyDb, error) {
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	return scheme(&Sqlite{dbConn}, reset, resetFrom)
}
