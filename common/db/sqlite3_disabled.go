// +build darwin windows

package db

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db/anydb"
)

type Sqlite struct{}

func (s *Sqlite) Name() string {
	return ""
}

func (s *Sqlite) Dialect() string {
	return ""
}

func OpenSqlite3(reset bool, resetFrom int) (anydb.AnyDb, error) {
	return nil, fmt.Errorf("sqlite3 not supported on darwin or windows")
}
