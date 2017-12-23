package db

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pdbogen/mapbot/common/db/anydb"
	"github.com/pdbogen/mapbot/common/db/schema"
	"github.com/pdbogen/mapbot/common/elephantsql"
	"strings"
)

type PostgreSql struct {
	*sql.DB
}

func (p *PostgreSql) Dialect() string {
	return "postgresql"
}

var Instance anydb.AnyDb

func OpenElephant(key, instance_type string, reset bool, resetFrom int) (anydb.AnyDb, error) {
	if key == "" {
		return nil, errors.New("key cannot be blank")
	}
	if instance_type == "" {
		return nil, errors.New("instance type cannot be blank")
	}

	es := &elephantsql.ElephantSql{
		ApiKey: key,
	}
	instance, err := es.FindInstance("mapbot")
	if err != nil {
		return nil, fmt.Errorf("communicating with ElephantSQL: %s", err)
	}

	if instance == nil {
		instance, err = es.NewInstance("mapbot", instance_type, nil)
		if err != nil {
			return nil, fmt.Errorf("creating new instance: %s", err)
		}
	} else {
		var err error
		instance, err = es.Enrich(instance)
		if err != nil {
			return nil, fmt.Errorf("enriching found instance: %s", err)
		}
	}

	conn, err := instance.Connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to instance %s:%d: %s", instance.DbHost, instance.DbPort, err)
	}
	return scheme(&anydb.WithRetries{&PostgreSql{conn}}, reset, resetFrom)
}

func OpenPsql(host, user, pass, db string, port int, reset bool, resetFrom int) (anydb.AnyDb, error) {
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
	return scheme(&anydb.WithRetries{&PostgreSql{dbConn}}, reset, resetFrom)
}

func scheme(dbConn anydb.AnyDb, reset bool, resetFrom int) (anydb.AnyDb, error) {
	if reset {
		if err := schema.Reset(dbConn); err != nil {
			return nil, err
		}
	} else if resetFrom >= 0 {
		if err := schema.ResetFrom(dbConn, resetFrom); err != nil {
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
