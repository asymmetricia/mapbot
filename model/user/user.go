package user

import (
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

type User struct {
	Id        types.UserId
	Workflows map[string]WorkflowState
}

func (u *User) String() string {
	return fmt.Sprintf("User{Id: %s}", u.Id)
}

func (u *User) Hydrate(db anydb.AnyDb) error {
	u.Workflows = map[string]WorkflowState{}
	wfRes, err := db.Query("SELECT name, state, opaque FROM user_workflows WHERE user_id=$1", &u.Id)
	if err != nil {
		return fmt.Errorf("querying user_workflows: %s", err)
	}
	defer wfRes.Close()
	for wfRes.Next() {
		var name, state, opaque string
		if err := wfRes.Scan(&name, &state, &opaque); err != nil {
			return fmt.Errorf("scanning user_workflows: %s", err)
		}
		u.Workflows[name] = WorkflowState{State: state, OpaqueRaw: []byte(opaque)}
	}
	return nil
}

func (u *User) Save(db anydb.AnyDb) error {
	var query string
	switch dia := db.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING"
	case "sqlite3":
		query = "REPLACE INTO users (id) VALUES ($1)"
	default:
		return fmt.Errorf("no User.Save query for SQL dialect %s", dia)
	}
	_, err := db.Exec(query, u.Id)
	if err != nil {
		return err
	}
	return u.saveWorkflows(db)
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
			opaqueJson, err := json.Marshal(wf_state.Opaque)
			if err != nil {
				log.Warningf("user=%s workflow=%s error marshaling opaque data: %s", u.Id, name, err)
				opaqueJson = []byte("{}")
				last_error = err
			}
			wf_state.OpaqueRaw = make([]byte, len(opaqueJson))
			copy(wf_state.OpaqueRaw, opaqueJson)
		}
		if _, err := db.Exec(query, u.Id, name, wf_state.State, wf_state.OpaqueRaw); err != nil {
			log.Warningf("user=%s workflow=%s error saving to database: %s", u.Id, name, err)
			last_error = err
		}
	}
	return last_error
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

	Instance.Lock.Lock()
	defer Instance.Lock.Unlock()

	// It's just possible that another goroutine created the user between our RUnlock and our Lock
	iUser, iUserOk := Instance.Users[id]
	if iUserOk {
		return iUser, nil
	}

	user := &User{Id: id}

	if err := user.Hydrate(db); err != nil {
		return nil, fmt.Errorf("hydrating user: %s", err)
	}

	Instance.Users[id] = user

	return user, nil
}

type Name string

func (n *Name) Value() (driver.Value, error) {
	return string(*n), nil
}

var _ driver.Valuer = (*Name)(nil)
