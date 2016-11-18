package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pdbogen/mapbot/common/db/schema"
	"strings"
)

var Instance *sql.DB

func Open(host, user, pass, db string, port int, reset bool) (*sql.DB, error) {
	dbConn, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"dbname=%s user=%s password=%s host=%s port=%d sslmode=verify-full",
			sanitize(db),
			sanitize(user),
			sanitize(pass),
			sanitize(host),
			port,
		),
	)
	if err != nil {
		return nil, err
	}
	if err := dbConn.Ping(); err != nil {
		return nil, err
	}

	if reset {
		if err := schema.Reset(dbConn); err != nil {
			return nil, err
		}
	}

	if err := schema.Apply(dbConn); err != nil {
		return nil, err
	}

	Instance = dbConn

	return dbConn, nil
}

func sanitize(in string) (out string) {
	return fmt.Sprintf("'%s'", strings.Replace(in, "'", "\\'", -1))
}
