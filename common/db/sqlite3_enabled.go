//go:build !darwin && !windows
// +build !darwin,!windows

package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pdbogen/mapbot/common/db/anydb"
)

type Sqlite struct {
	*sql.DB
	file string
}

func (s *Sqlite) Name() string {
	return "sqlite3:" + s.file
}

func (s *Sqlite) Dialect() string {
	return "sqlite3"
}

func OpenSqlite3(reset bool, resetFrom int, file string) (anydb.AnyDb, error) {
	if file == "" {
		file = ":memory:"
	}
	dbConn, err := sql.Open("sqlite3", "file:"+file)
	if err != nil {
		return nil, err
	}
	return scheme(&Sqlite{dbConn, file}, reset, resetFrom)
}
