package schema

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db/anydb"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"sort"
)

var log = mbLog.Log

var schema = []Migration{
	Migration{
		0,
		map[string]string{"any": "CREATE TABLE slack_teams (token VARCHAR(128) UNIQUE, bot_id VARCHAR(64), bot_token VARCHAR(64))"},
		map[string]string{"any": "DROP TABLE IF EXISTS slack_teams"},
	},
	Migration{
		1,
		map[string]string{"any": "CREATE TABLE slack_nonces (nonce VARCHAR(64), expiry TIMESTAMP)"},
		map[string]string{"any": "DROP TABLE IF EXISTS slack_nonces"},
	},
	Migration{
		2,
		map[string]string{
			"sqlite3": "CREATE TABLE tabulas (" +
				"id INTEGER PRIMARY KEY, " +
				"name VARCHAR(128), " +
				"url VARCHAR(256)," +
				"offset_x integer default 0," +
				"offset_y integer default 0," +
				"dpi real default 0" +
				")",
			"any": "CREATE TABLE tabulas (" +
				"id BIGSERIAL PRIMARY KEY, " +
				"name VARCHAR(128), " +
				"url VARCHAR(256)," +
				"offset_x integer default 0," +
				"offset_y integer default 0," +
				"dpi real default 0" +
				")"},
		map[string]string{"any": "DROP TABLE IF EXISTS tabulas"},
	},
	Migration{
		3,
		map[string]string{"any": "CREATE TABLE users (id VARCHAR(9) PRIMARY KEY, prefAutoShow BOOLEAN)"},
		map[string]string{"any": "DROP TABLE IF EXISTS users"},
	},
	Migration{
		4,
		map[string]string{"any": "CREATE TABLE user_tabulas (" +
			"user_id VARCHAR(9) REFERENCES users (id) ON DELETE CASCADE, " +
			"tabula_id BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE, " +
			"PRIMARY KEY (user_id, tabula_id)" +
			")"},
		map[string]string{"any": "DROP TABLE IF EXISTS user_tabulas"},
	},
	Migration{
		5,
		map[string]string{"any": "ALTER TABLE tabulas ADD COLUMN grid_r INT NOT NULL DEFAULT 0; " +
			"ALTER TABLE tabulas ADD COLUMN grid_g INT NOT NULL DEFAULT 0; " +
			"ALTER TABLE tabulas ADD COLUMN grid_b INT NOT NULL DEFAULT 0; " +
			"ALTER TABLE tabulas ADD COLUMN grid_a INT NOT NULL DEFAULT 0;"},
		map[string]string{"any": "ALTER TABLE tabulas DROP COLUMN grid_r, DROP COLUMN grid_g, DROP COLUMN grid_b, DROP COLUMN grid_a"},
	},
	Migration{
		6,
		map[string]string{"any": `CREATE TABLE tabula_masks (` +
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
			`)`},
		map[string]string{"any": "DROP TABLE tabula_masks"},
	},
	Migration{
		7,
		map[string]string{"any": `CREATE TABLE contexts (context_id VARCHAR(128) PRIMARY KEY, active_tabula BIGSERIAL REFERENCES tabulas (id) ON DELETE SET NULL)`},
		map[string]string{"any": `DROP TABLE contexts`},
	},
	Migration{
		8,
		map[string]string{"any": `CREATE TABLE tabula_tokens(` +
			`name VARCHAR(128),` +
			`context_id VARCHAR(128),` +
			`tabula_id BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE,` +
			`x INT NOT NULL DEFAULT 0,` +
			`y INT NOT NULL DEFAULT 0,` +
			`PRIMARY KEY (name, context_id, tabula_id)` +
			`)`},
		map[string]string{"any": `DROP TABLE tabula_tokens`},
	},
	Migration{
		9,
		map[string]string{"any": `ALTER TABLE tabulas ADD COLUMN version INT NOT NULL DEFAULT 0`},
		map[string]string{"any": `ALTER TABLE tabulas DROP COLUMN version`},
	},
	Migration{
		10,
		map[string]string{"any": `ALTER TABLE tabula_tokens ADD COLUMN r SMALLINT NOT NULL DEFAULT 0;` +
			`ALTER TABLE tabula_tokens ADD COLUMN g SMALLINT NOT NULL DEFAULT 0;` +
			`ALTER TABLE tabula_tokens ADD COLUMN b SMALLINT NOT NULL DEFAULT 0;` +
			`ALTER TABLE tabula_tokens ADD COLUMN a SMALLINT NOT NULL DEFAULT 0;`},
		map[string]string{"any": `ALTER TABLE tabula_tokens DROP COLUMN r, DROP COLUMN g, DROP COLUMN b, DROP COLUMN a`},
	},
	Migration{
		11,
		map[string]string{"any": `CREATE TABLE user_workflows(` +
			`user_id VARCHAR(9) REFERENCES users (id) ON DELETE CASCADE, ` +
			`name    VARCHAR(32), ` +
			`state   VARCHAR(32), ` +
			`opaque  TEXT,` +
			`PRIMARY KEY (user_id, name)` +
			`)`},
		map[string]string{"any": `DROP TABLE user_workflows`},
	},
	Migration{
		12,
		map[string]string{"any": `ALTER TABLE contexts ADD COLUMN MinX SMALLINT NOT NULL DEFAULT 0;` +
			`ALTER TABLE contexts ADD COLUMN MinY SMALLINT NOT NULL DEFAULT 0;` +
			`ALTER TABLE contexts ADD COLUMN MaxX SMALLINT NOT NULL DEFAULT 0;` +
			`ALTER TABLE contexts ADD COLUMN MaxY SMALLINT NOT NULL DEFAULT 0`,
		},
		map[string]string{"any": `ALTER TABLE contexts DROP COLUMN MinX, DROP COLUMN MinY, DROP COLUMN MaxX, DROP COLUMN MaxY`},
	},
	Migration{
		13,
		map[string]string{"any": `ALTER TABLE tabula_tokens ADD COLUMN size SMALLINT NOT NULL DEFAULT 1`},
		map[string]string{"any": `ALTER TABLE tabula_tokens DROP COLUMN size`},
	},
	Migration{
		14,
		map[string]string{"any": `CREATE TABLE context_marks (` +
			`context_id VARCHAR(128) REFERENCES contexts(context_id) ON DELETE CASCADE, ` +
			`tabula_id  BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE,` +
			`square_x   SMALLINT,` +
			`square_y   SMALLINT,` +
			`red        SMALLINT,` +
			`green      SMALLINT,` +
			`blue       SMALLINT,` +
			`alpha      SMALLINT,` +
			`PRIMARY KEY (context_id, tabula_id, square_x, square_y)` +
			`)`},
		map[string]string{"any": `DROP TABLE context_marks`},
	},
	Migration{
		15,
		map[string]string{
			"sqlite3": `CREATE TABLE context_marks2 (` +
				`context_id VARCHAR(128) REFERENCES contexts(context_id) ON DELETE CASCADE, ` +
				`tabula_id  BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE,` +
				`square_x   SMALLINT,` +
				`square_y   SMALLINT,` +
				`red        SMALLINT,` +
				`green      SMALLINT,` +
				`blue       SMALLINT,` +
				`alpha      SMALLINT,` +
				`direction VARCHAR(2) NOT NULL DEFAULT '',` +
				`PRIMARY KEY (context_id, tabula_id, square_x, square_y, direction));` +
				`INSERT INTO context_marks2 (context_id,tabula_id,square_x,square_y,red,green,blue,alpha) SELECT * FROM context_marks;` +
				`DROP TABLE context_marks;` +
				`ALTER TABLE context_marks2 RENAME TO context_marks`,
			"any": `ALTER TABLE context_marks ADD COLUMN direction VARCHAR(2) NOT NULL DEFAULT '';` +
				`ALTER TABLE context_marks DROP CONSTRAINT context_marks_pkey;` +
				`ALTER TABLE context_marks ADD PRIMARY KEY (context_id, tabula_id, square_x, square_y, direction);`},
		map[string]string{
			"sqlite3": `CREATE TABLE context_marks2 (` +
				`context_id VARCHAR(128) REFERENCES contexts(context_id) ON DELETE CASCADE, ` +
				`tabula_id  BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE,` +
				`square_x   SMALLINT,` +
				`square_y   SMALLINT,` +
				`red        SMALLINT,` +
				`green      SMALLINT,` +
				`blue       SMALLINT,` +
				`alpha      SMALLINT,` +
				`PRIMARY KEY (context_id, tabula_id, square_x, square_y));` +
				`INSERT INTO context_marks2 (context_id,tabula_id,square_x,square_y,red,green,blue,alpha) SELECT * FROM context_marks;` +
				`DROP TABLE context_marks;` +
				`ALTER TABLE context_marks2 RENAME TO context_marks`,
			"any": `ALTER TABLE context_marks DROP CONSTRAINT context_marks_pkey;` +
				`ALTER TABLE context_marks DROP COLUMN direction;` +
				`ALTER TABLE context_marks ADD PRIMARY KEY (context_id, tabula_id, square_x, square_y);`},
	},
	Migration{
		16,
		map[string]string{"any": `ALTER TABLE context_marks ADD COLUMN stale BOOLEAN DEFAULT TRUE`},
		map[string]string{"any": `ALTER TABLE context_marks DROP COLUMN stale`},
	},
	Migration{
		17,
		map[string]string{"any": `ALTER TABLE tabula_tokens ADD COLUMN light_low INT NOT NULL DEFAULT 0; ` +
			`ALTER TABLE tabula_tokens ADD COLUMN light_normal INT NOT NULL DEFAULT 0; ` +
			`ALTER TABLE tabula_tokens ADD COLUMN light_bright INT NOT NULL DEFAULT 0; `,
		},
		map[string]string{"any": `ALTER TABLE tabula_tokens DROP COLUMN light_low;` +
			`ALTER TABLE tabula_tokens DROP COLUMN light_normal;` +
			`ALTER TABLE tabula_tokens DROP COLUMN light_bright;`,
		},
	},
	Migration{
		18,
		map[string]string{"any": `ALTER TABLE tabula_tokens RENAME light_low TO light_dim`},
		map[string]string{"any": `ALTER TABLE tabula_tokens RENAME light_dim TO light_low`},
	},
	Migration{
		19,
		map[string]string{"any": `CREATE TABLE entities (` +
			`context_id VARCHAR(128) REFERENCES contexts(context_id) ON DELETE CASCADE, ` +
			`tabula_id  BIGSERIAL REFERENCES tabulas (id) ON DELETE CASCADE,` +
			`entity_id  INTEGER,` +
			`string     TEXT,` +
			`PRIMARY KEY (context_id, tabula_id, entity_id)` +
			`)`},
		map[string]string{"any": `DROP TABLE entities`},
	},
	Migration{
		20,
		map[string]string{"any": `CREATE TABLE last_token (` +
			`context_id VARCHAR(128) REFERENCES contexts(context_id) ON DELETE CASCADE, ` +
			`user_id VARCHAR(9) REFERENCES users (id) ON DELETE CASCADE, ` +
			`token_name VARCHAR(128),` +
			`PRIMARY KEY (context_id, user_id)` +
			`)`},
		map[string]string{"any": `DROP TABLE last_token`},
	},
	Migration{
		21,
		map[string]string{"any": `CREATE TABLE IF NOT EXISTS web_sessions (` +
			`session_id VARCHAR(64),` +
			`context_id VARCHAR(128) REFERENCES contexts(context_id) ON DELETE CASCADE,` +
			`PRIMARY KEY (session_id)` +
			`)`,
		},
		map[string]string{"any": `DROP TABLE IF EXISTS web_sessions`},
	},
	Migration{
		22,
		map[string]string{"any": `ALTER TABLE web_sessions ADD COLUMN context_type VARCHAR(32)`},
		map[string]string{"any": `ALTER TABLE web_sessions DROP COLUMN context_type`},
	},
}

func Reset(db anydb.AnyDb) error {
	return ResetFrom(db, 0)
}

func ResetFrom(db anydb.AnyDb, migration int) error {
	sort.Sort(sort.Reverse(SortMigrationById(schema)))
	for _, m := range schema {
		if m.Id >= migration {
			if err := m.ApplyDown(db); err != nil {
				return err
			}
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
	Up   map[string]string
	Down map[string]string
}

func Apply(db anydb.AnyDb) error {
	sort.Sort(SortMigrationById(schema))
	for _, m := range schema {
		if err := m.ApplyUp(db); err != nil {
			return fmt.Errorf("applying migation #%d: %s", m.Id, err)
		}
	}
	return nil
}

func (m *Migration) ApplyUp(db anydb.AnyDb) error {
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

	stmt, ok := m.Up[db.Dialect()]
	if !ok {
		stmt, ok = m.Up["any"]
	}
	if !ok {
		return fmt.Errorf("migration %d has no Up statement for dialect %s", m.Id, db.Dialect())
	}

	_, err = db.Exec(stmt)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO migrations VALUES ($1)", m.Id)
	return err
}

func (m *Migration) ApplyDown(db anydb.AnyDb) error {
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

	stmt, ok := m.Down[db.Dialect()]
	if !ok {
		stmt, ok = m.Down["any"]
	}

	if !ok {
		return fmt.Errorf("migration %d has no Down statement for dialect %s", m.Id, db.Dialect())
	}

	log.Infof("executing down-migration %d: %s", m.Id, stmt)
	_, err = db.Exec(stmt)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM migrations WHERE migration_id=$1", m.Id)
	return err
}

func (m *Migration) Applied(db anydb.AnyDb) (bool, error) {
	results, err := db.Query("SELECT * FROM migrations WHERE migration_id=$1", m.Id)
	if err != nil {
		return false, err
	}
	defer results.Close()

	rowsExist := results.Next()
	return rowsExist, nil
}

func initSchema(db anydb.AnyDb) error {
	if initialized {
		return nil
	}

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS migrations (migration_id INTEGER NOT NULL)"); err != nil {
		return err
	}
	initialized = true
	return nil
}
