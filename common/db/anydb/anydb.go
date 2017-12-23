package anydb

import (
	"database/sql"
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

type AnyDb interface {
	Dialect() string
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	Begin() (*sql.Tx, error)
	Prepare(string) (*sql.Stmt, error)
}

type WithRetries struct {
	AnyDb
}

var retryable = []*regexp.Regexp{
	regexp.MustCompile("too many connections"),
}

// delay delays for `ms` milliseconds, +/- a random 10% only if the duration is above 10ms
func delay(ms int) {
	if ms <= 10 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return
	}
	perc := ms / 10
	time.Sleep(time.Duration(ms-perc+int(rand.Int31n(int32(2*perc)))) * time.Millisecond)
}

func (wr *WithRetries) Exec(s string, v ...interface{}) (res sql.Result, err error) {
	for _, delayMs := range []int{0, 10, 100, 1000, 2000, 3000} {
		delay(delayMs)
		res, err = wr.AnyDb.Exec(s, v...)
		if err == nil {
			return
		}

		for _, re := range retryable {
			if re.MatchString(err.Error()) {
				continue
			}
		}
		break
	}
	return res, fmt.Errorf("after 5 tries: %s", err)
}

func (wr *WithRetries) Query(s string, v ...interface{}) (res *sql.Rows, err error) {
	for _, delayMs := range []int{0, 10, 100, 1000, 2000, 3000} {
		delay(delayMs)
		res, err = wr.AnyDb.Query(s, v...)
		if err == nil {
			return
		}
		for _, re := range retryable {
			if re.MatchString(err.Error()) {
				continue
			}
		}
		break
	}
	return res, fmt.Errorf("after 5 tries: %s", err)
}

func (wr *WithRetries) Begin() (res *sql.Tx, err error) {
	for _, delayMs := range []int{0, 10, 100, 1000, 2000, 3000} {
		delay(delayMs)
		res, err = wr.AnyDb.Begin()
		if err == nil {
			return
		}
		for _, re := range retryable {
			if re.MatchString(err.Error()) {
				continue
			}
		}
		break
	}
	return res, fmt.Errorf("after 5 tries: %s", err)
}

func (wr *WithRetries) Prepare(s string) (res *sql.Stmt, err error) {
	for _, delayMs := range []int{0, 10, 100, 1000, 2000, 3000} {
		delay(delayMs)
		res, err = wr.AnyDb.Prepare(s)
		if err == nil {
			return
		}
		for _, re := range retryable {
			if re.MatchString(err.Error()) {
				continue
			}
		}
		break
	}
	return res, fmt.Errorf("after 5 tries: %s", err)
}
