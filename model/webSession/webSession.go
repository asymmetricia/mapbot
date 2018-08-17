package webSession

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db/anydb"
	"github.com/pdbogen/mapbot/common/rand"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/context/databaseContext"
	"github.com/pdbogen/mapbot/model/types"
)

type WebSession struct {
	SessionId string
	ContextId types.ContextId
}

func NewWebSession(db anydb.AnyDb, ctx types.ContextId) (*WebSession, error) {
	ret := &WebSession{
		SessionId: rand.RandHex(32),
		ContextId: ctx,
	}

	if err := ret.Save(db); err != nil {
		return nil, fmt.Errorf("saving new session: %s", err)
	}

	return ret, nil
}

func (w WebSession) Save(db anydb.AnyDb) error {
	_, err := db.Exec("INSERT INTO web_sessions (session_id, context_id) VALUES ($1,$2)", w.SessionId, w.ContextId)
	if err != nil {
		return fmt.Errorf("saving session %q: %v", w.SessionId, err)
	}
	return nil
}

type NotFound error

func Load(db anydb.AnyDb, sessionId string) (*WebSession, error) {
	res, err := db.Query("SELECT session_id, context_id FROM web_Sessions WHERE session_id=$1", sessionId)
	if err != nil {
		return nil, fmt.Errorf("querying web_sessions for session %q: %v", sessionId, err)
	}
	if !res.Next() {
		return nil, NotFound(fmt.Errorf("session %q not found", sessionId))
	}
	ret := &WebSession{}
	res.Scan(&ret.SessionId, &ret.ContextId)
	return ret, nil
}

func (w WebSession) GetContext(db anydb.AnyDb) (context.Context, error) {
	ctx, err := databaseContext.Load(db, w.ContextId)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}
