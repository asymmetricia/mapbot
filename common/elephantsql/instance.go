package elephantsql

import (
	"fmt"
	"net/url"
	"strings"
	"database/sql"
	"errors"
	"strconv"
)

type Instance struct {
	Id     int    `json:"id"`
	Plan   string `json:"plan,omitempty"`
	Region string `json:"region,omitempty"`
	Name   string `json:"name,omitempty"`
	Url    string `json:"url,omitempty"`
	DbUser string `json:"-"`
	DbPass string `json:"-"`
	DbHost string `json:"-"`
	DbPort int    `json:"-"`
	DbName string `json:"-"`
}

// ParseUrl parses the instance's URL field and populates the various Db* fields. An error is returned if anything goes
// wrong during parsing, or if required parameters are absent.
func (i *Instance) ParseUrl() error {
	dbUrl, err := url.Parse(i.Url)
	if err != nil {
		return fmt.Errorf("error parsing instance URL %q: %s", i.Url, err)
	}

	i.DbUser = dbUrl.User.Username()
	if pass, ok := dbUrl.User.Password(); ok {
		i.DbPass = pass
	} else {
		return fmt.Errorf("instance URL %q did not contain password", i.Url)
	}
	hostParts := strings.Split(dbUrl.Host, ":")
	i.DbHost = hostParts[0]
	if len(hostParts) > 1 {
		i.DbPort, err = strconv.Atoi(hostParts[1])
		if err != nil {
			return fmt.Errorf("instance URL %q had malformed port: %s", i.Url, err)
		}
	}
	i.DbName = dbUrl.Path
	return nil
}

// Connect establishes a sql.DB connection to the given instance, and pings it to ensure the connection is working. The
// Instance must be hydrated/detailed before this method can be used.
func (i *Instance) Connect() (*sql.DB, error) {
	if i.DbHost == "" {
		return nil, errors.New("cannot call Connect() on an un-detailed instance")
	}
	dbConn, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"dbname=%s user=%s password=%s host=%s port=%d sslmode=verify-full",
			sanitize(i.DbName),
			sanitize(i.DbUser),
			sanitize(i.DbPass),
			sanitize(i.DbHost),
			i.DbPort,
		),
	)
	if err != nil {
		return nil, err
	}

	if err := dbConn.Ping(); err != nil {
		return nil, err
	}

	return dbConn, nil
}

func sanitize(in string) (out string) {
	return fmt.Sprintf("'%s'", strings.Replace(in, "'", "\\'", -1))
}
