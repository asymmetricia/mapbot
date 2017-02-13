package user

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"sync"
)

var log = mbLog.Log

var Instance = &UserStore{
	Users: map[Id]*User{},
	Lock: sync.RWMutex{},
}

type UserStore struct {
	Users map[Id]*User
	Lock sync.RWMutex
}

type User struct {
	Id       Id
	Tabulas  []*tabula.Tabula
	AutoShow bool
}

func (u *User) String() string {
	if u.Tabulas == nil {
		return fmt.Sprintf("User{Id: %s, Tabulas: nil}", u.Id)
	}

	return fmt.Sprintf("User{Id: %s, Tabulas: %d}", u.Id, len(u.Tabulas))
}

func (u *User) Hydrate(db *sql.DB) error {
	res, err := db.Query("SELECT prefAutoShow FROM users WHERE id=$1", &u.Id)
	if err != nil {
		return fmt.Errorf("querying: %s", err)
	}
	defer res.Close()

	if res.Next() {
		if err := res.Scan(&(u.AutoShow)); err != nil {
			return fmt.Errorf("scanning: %s", err)
		}
	}
	return nil
}

func (u *User) Save(db *sql.DB) error {
	_, err := db.Exec(
		"INSERT INTO users (id, prefAutoShow) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET prefAutoShow=$2",
		&u.Id, u.AutoShow,
	)
	return err
}

func Get(db *sql.DB, id Id) (*User, error) {
	Instance.Lock.RLock()
	iUser, iUserOk := Instance.Users[id]
	Instance.Lock.RUnlock()
	if iUserOk {
		return iUser, nil
	}

	user := &User{Id: id, Tabulas: []*tabula.Tabula{}}

	res, err := db.Query("SELECT tabula_id FROM user_tabulas WHERE user_id=$1", &id)
	if err != nil {
		return nil, fmt.Errorf("querying tabulas: %s", err)
	}
	defer res.Close()

	if err := user.Hydrate(db); err != nil {
		return nil, fmt.Errorf("hydrating user: %s", err)
	}

	var tid types.TabulaId
	for res.Next() {
		if err := res.Scan(&tid); err != nil {
			return nil, fmt.Errorf("retrieving results: %s", err)
		}

		tab, err := tabula.Get(db, tid)
		if err != nil {
			log.Errorf("retrieving tabula %f: %s", tid, err)
			continue
		}

		user.Tabulas = append(user.Tabulas, tab)
	}

	Instance.Lock.Lock()
	defer Instance.Lock.Unlock()
	Instance.Users[id] = user

	return user, nil
}

func (u *User) TabulaByName(name tabula.TabulaName) (*tabula.Tabula, bool) {
	if u == nil || u.Tabulas == nil {
		return nil, false
	}

	for _, t := range u.Tabulas {
		if t.Name == name {
			return t, true
		}
	}
	return nil, false
}

func (u *User) Assign(db *sql.DB, t *tabula.Tabula) error {
	if u == nil {
		return errors.New("Assign called on nil User")
	}
	if u.Tabulas == nil {
		u.Tabulas = []*tabula.Tabula{}
	}

	found := false
	for _, user_tabula := range u.Tabulas {
		if user_tabula.Id == t.Id {
			found = true
		}
	}
	if !found {
		u.Tabulas = append(u.Tabulas, t)
	}

	_, err := db.Exec(
		"INSERT INTO users (id, prefAutoShow) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET prefAutoShow=$2",
		&u.Id, u.AutoShow,
	)
	if err != nil {
		return fmt.Errorf("upserting user: %s", err)
	}

	_, err = db.Exec(
		"INSERT INTO user_tabulas (user_id, tabula_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		&u.Id, t.Id,
	)

	return err
}

type Name string

func (n *Name) Value() (driver.Value, error) {
	return string(*n), nil
}

var _ driver.Valuer = (*Name)(nil)

type Id string

func (i *Id) Value() (driver.Value, error) {
	return string(*i), nil
}

var _ driver.Valuer = (*Id)(nil)
