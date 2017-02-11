package schema

import (
	"database/sql"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"sort"
)

var log = mbLog.Log

var schema = []Migration{
	Migration{
		0,
		"CREATE TABLE slack_teams (token VARCHAR(128) UNIQUE, bot_id VARCHAR(64), bot_token VARCHAR(64))",
		"DROP TABLE IF EXISTS slack_teams",
	},
	Migration{
		1,
		"CREATE TABLE slack_nonces (nonce VARCHAR(64), expiry TIMESTAMP)",
		"DROP TABLE IF EXISTS slack_nonces",
	},
	Migration{
		2,
		"CREATE TABLE tabulas (" +
			"id BIGSERIAL PRIMARY KEY, " +
			"name VARCHAR(128), " +
			"url VARCHAR(256)," +
			"offset_x integer default 0," +
			"offset_y integer default 0," +
			"dpi real default 0" +
			")",
		"DROP TABLE IF EXISTS tabulas",
	},
	Migration{
		3,
		"CREATE TABLE user_tabulas (" +
			"user_id VARCHAR(9), " +
			"tabula_id BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE, " +
			"PRIMARY KEY (user_id, tabula_id)" +
			")",
		"DROP TABLE IF EXISTS user_tabulas",
	},
	Migration{
		4,
		"CREATE TABLE users (id VARCHAR(9) PRIMARY KEY, prefAutoShow BOOLEAN)",
		"DROP TABLE IF EXISTS users",
	},
	Migration{
		5,
		"ALTER TABLE user_tabulas ADD CONSTRAINT user_id_fk FOREIGN KEY (user_id) REFERENCES users (id)",
		"ALTER TABLE user_tabulas DROP CONSTRAINT user_id_fk",
	},
	Migration{
		6,
		"ALTER TABLE tabulas " +
			"ADD COLUMN grid_r INT NOT NULL DEFAULT 0, " +
			"ADD COLUMN grid_g INT NOT NULL DEFAULT 0, " +
			"ADD COLUMN grid_b INT NOT NULL DEFAULT 0, " +
			"ADD COLUMN grid_a INT NOT NULL DEFAULT 0",
		"ALTER TABLE tabulas DROP COLUMN grid_r, DROP COLUMN grid_g, DROP COLUMN grid_b, DROP COLUMN grid_a",
	},
	Migration{
		7,
		`CREATE TABLE tabula_masks (` +
			`name VARCHAR(128),` +
			`"order" INT NOT NULL DEFAULT 0,` +
			`tabula_id BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE,` +
			`red INT NOT NULL DEFAULT 0,` +
			`green INT NOT NULL DEFAULT 0,` +
			`blue INT NOT NULL DEFAULT 0,` +
			`alpha INT NOT NULL DEFAULT 0,` +
			`top INT NOT NULL DEFAULT 0,` +
			`"left" INT NOT NULL DEFAULT 0,` +
			`width INT NOT NULL DEFAULT 0,` +
			`height INT NOT NULL DEFAULT 0,` +
			`PRIMARY KEY (tabula_id, name)` +
			`)`,
		"DROP TABLE tabula_masks",
	},
	Migration{
		8,
		`CREATE TABLE contexts (context_id VARCHAR(128) PRIMARY KEY, active_tabula BIGSERIAL REFERENCES tabulas (id) ON DELETE SET NULL)`,
		`DROP TABLE contexts`,
	},
	Migration{
		9,
		`CREATE TABLE tabula_tokens(` +
			`name VARCHAR(128),` +
			`tabula_id BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE,` +
			`x INT NOT NULL DEFAULT 0,` +
			`y INT NOT NULL DEFAULT 0` +
			`)`,
		`DROP TABLE tabula_tokens`,
	},
}

func Reset(db *sql.DB) error {
	sort.Sort(sort.Reverse(SortMigrationById(schema)))
	for _, m := range schema {
		if err := m.ApplyDown(db); err != nil {
			return err
		}
	}
	return nil
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
	Id   int
	Up   string
	Down string
}

func Apply(db *sql.DB) error {
	sort.Sort(SortMigrationById(schema))
	for _, m := range schema {
		if err := m.ApplyUp(db); err != nil {
			return fmt.Errorf("applying migation #%d: %s", m.Id, err)
		}
	}
	return nil
}

func (m *Migration) ApplyUp(db *sql.DB) error {
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

	log.Infof("executing up-migration %d: %s", m.Id, m.Up)
	_, err = db.Exec(m.Up)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO migrations VALUES ($1)", m.Id)
	return err
}

func (m *Migration) ApplyDown(db *sql.DB) error {
	if err := initSchema(db); err != nil {
		return err
	}

	applied, err := m.Applied(db)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}

	log.Infof("executing down-migration %d: %s", m.Id, m.Down)
	_, err = db.Exec(m.Down)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM migrations WHERE migration_id=$1", m.Id)
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
