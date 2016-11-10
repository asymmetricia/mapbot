package db

import (
	"database/sql"
	_ "github.com/lib/pq"
	"fmt"
	"strings"
	"github.com/pdbogen/mapbot/common/db/schema"
)

func Open(host, user, pass, db string, port int) (*sql.DB, error) {
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

	if err := schema.Apply(dbConn); err != nil {
		return nil, err
	}

	return dbConn, nil
}

func sanitize(in string) (out string) {
	return fmt.Sprintf("'%s'", strings.Replace(in, "'", "\\'", -1))
}