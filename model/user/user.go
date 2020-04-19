package user

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pdbogen/mapbot/common/db/anydb"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"sync"
)

var log = mbLog.Log

var Instance = &UserStore{
	Users: map[types.UserId]*User{},
	Lock:  sync.RWMutex{},
}

type UserStore struct {
	Users map[types.UserId]*User
	Lock  sync.RWMutex
}

type WorkflowState struct {
	State     string
	OpaqueRaw []byte
	Opaque    interface{}
}

func (state *WorkflowState) Hydrate(OpaqueFromJson func([]byte) (interface{}, error)) {
	if OpaqueFromJson == nil {
		return
	}
	if len(state.OpaqueRaw) == 0 {
		return
	}
	if state.Opaque != nil {
		return
	}
	var err error
	state.Opaque, err = OpaqueFromJson(state.OpaqueRaw)
	if err != nil {
		log.Errorf("could not hydrate opaque state from %q: %v", string(state.OpaqueRaw), err)
	}
}

type User struct {
	Id        types.UserId
	Tabulas   []*tabula.Tabula
	AutoShow  bool
	Workflows map[string]WorkflowState
}

func (u *User) String() string {
	if u.Tabulas == nil {
		return fmt.Sprintf("User{Id: %s, Tabulas: nil}", u.Id)
	}

	return fmt.Sprintf("User{Id: %s, Tabulas: %d}", u.Id, len(u.Tabulas))
}

func (u *User) Hydrate(db anydb.AnyDb) error {
	res, err := db.Query("SELECT prefAutoShow FROM users WHERE id=$1", &u.Id)
	if err != nil {
		return fmt.Errorf("querying %s: %s", db.Name(), err)
	}
	defer res.Close()

	if res.Next() {
		if err := res.Scan(&(u.AutoShow)); err != nil {
			return fmt.Errorf("scanning: %s", err)
		}
	}

	return u.hydrateWorkflows(db)
}

func (u *User) hydrateWorkflows(db anydb.AnyDb) error {
	u.Workflows = map[string]WorkflowState{}
	wfRes, err := db.Query("SELECT name, state, opaque FROM user_workflows WHERE user_id=$1", &u.Id)
	if err != nil {
		return fmt.Errorf("querying user_workflows: %s", err)
	}
	defer wfRes.Close()
	for wfRes.Next() {
		var name, state, opaqueRaw string
		if err := wfRes.Scan(&name, &state, &opaqueRaw); err != nil {
			return fmt.Errorf("scanning user_workflows: %s", err)
		}
		u.Workflows[name] = WorkflowState{State: state, OpaqueRaw: []byte(opaqueRaw)}
	}
	return nil
}

func (u *User) saveWorkflows(db anydb.AnyDb) (last_error error) {
	var query string
	switch dia := db.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO user_workflows (user_id, name, state, opaque) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id,name) DO UPDATE SET state=$3, opaque=$4"
	case "sqlite3":
		query = "REPLACE INTO user_workflows (user_id, name, state, opaque) VALUES ($1, $2, $3, $4)"
	}
	for name, wf_state := range u.Workflows {
		if wf_state.Opaque != nil {
			var err error
			wf_state.OpaqueRaw, err = json.Marshal(wf_state.Opaque)
			if err != nil {
				log.Warningf("user=%s workflow=%s error marshaling opaque data: %s", u.Id, name, err)
				wf_state.OpaqueRaw = []byte("{}")
				last_error = err
			}
		}
		if _, err := db.Exec(query, u.Id, name, wf_state.State, wf_state.OpaqueRaw); err != nil {
			log.Warningf("user=%s workflow=%s error saving to database: %s", u.Id, name, err)
			last_error = err
		}
	}
	return last_error
}

func (u *User) Save(db anydb.AnyDb) error {
	var query string
	switch dia := db.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO users (id, prefAutoShow) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET prefAutoShow=$2"
	case "sqlite3":
		query = "REPLACE INTO users (id, prefAutoShow) VALUES ($1, $2)"
	default:
		return fmt.Errorf("no User.Save query for SQL dialect %s", dia)
	}
	_, err := db.Exec(query, u.Id, u.AutoShow)
	if err != nil {
		return err
	}
	return u.saveWorkflows(db)
}

// Get uses the database given by `db` to retrieve the user identified by id
// `id`. On success, it returns a pointer to object representing the user; on
// failure, it returns nil and an error.
func Get(db anydb.AnyDb, id types.UserId) (*User, error) {
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

		tab, err := tabula.Load(db, tid)
		if err != nil {
			log.Errorf("retrieving tabula %d: %s", tid, err)
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

func (u *User) Assign(db anydb.AnyDb, t *tabula.Tabula) error {
	tx, err := db.Begin()
	if err == nil {
		err = u.AssignTx(db.Dialect(), tx, t)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil && tx != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			err = fmt.Errorf("%v and could not rollback: %v", err, rbErr)
		}
	}
	return err
}

func (u *User) AssignTx(dialect string, tx *sql.Tx, t *tabula.Tabula) error {
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

	var usersQuery, userTabulaeQuery string
	switch dialect {
	case "postgresql":
		usersQuery = "INSERT INTO users (id, prefAutoShow) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET prefAutoShow=$2"
		userTabulaeQuery = "INSERT INTO user_tabulas (user_id, tabula_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
	case "sqlite3":
		usersQuery = "REPLACE INTO users (id, prefAutoShow) VALUES ($1, $2)"
		userTabulaeQuery = "INSERT INTO user_tabulas (user_id, tabula_id) VALUES ($1, $2)"
	default:
		return fmt.Errorf("no User.Assign query for SQL dialect %s", dialect)
	}
	_, err := tx.Exec(usersQuery, &u.Id, u.AutoShow)
	if err != nil {
		return fmt.Errorf("upserting user: %s", err)
	}

	_, err = tx.Exec(userTabulaeQuery, &u.Id, t.Id)
	return err
}

type Name string

func (n *Name) Value() (driver.Value, error) {
	return string(*n), nil
}

var _ driver.Valuer = (*Name)(nil)
