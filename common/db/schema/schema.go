package schema

import (
	"database/sql"
	"sort"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
)

var log = mbLog.Log

var schema = []Migration{
	Migration{0, "CREATE TABLE slack_teams (token VARCHAR(1024))"},
}

type SortMigrationById []Migration

func (s SortMigrationById) Len() int {
	return len(s)
}

func (s SortMigrationById) Less(a, b int) bool {
	return s[a].Id < s[b].Id
}

func (s SortMigrationById) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

var _ sort.Interface = (SortMigrationById)(nil)

var initialized bool = false

type Migration struct {
	Id int
	Up string
}

func Apply(db *sql.DB) error {
	sort.Sort(SortMigrationById(schema))
	for _, m := range schema {
		if err := m.Apply(db); err != nil {
			return fmt.Errorf("applying migation #%d: %s", m.Id, err)
		}
	}
	return nil
}

func (m *Migration) Apply(db *sql.DB) error {
	if err := initSchema(db); err != nil {
		return err
	}

	applied, err := m.Applied(db)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	log.Infof("executing migration %s", m.Up)
	_, err = db.Exec(m.Up)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO migrations VALUES ($1)", m.Id)
	return err
}

func (m *Migration) Applied(db *sql.DB) (bool, error) {
	results, err := db.Query("SELECT * FROM migrations WHERE migration_id=$1", m.Id)
	if err != nil {
		return false, err
	}
	defer results.Close()

	rowsExist := results.Next()
	return rowsExist, nil
}

func initSchema(db *sql.DB) error {
	if initialized {
		return nil
	}

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS migrations (migration_id INTEGER NOT NULL)"); err != nil {
		return err
	}
	initialized = true
	return nil
}