package webSession

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db/anydb"
	"github.com/pdbogen/mapbot/common/rand"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/types"
)

type WebSession struct {
	SessionId   string
	ContextId   types.ContextId
	ContextType types.ContextType
}

func NewWebSession(db anydb.AnyDb, ctx types.ContextId, ctxTyp types.ContextType) (*WebSession, error) {
	ret := &WebSession{
		SessionId:   rand.RandHex(32),
		ContextId:   ctx,
		ContextType: ctxTyp,
	}

	if err := ret.Save(db); err != nil {
		return nil, fmt.Errorf("saving new session: %s", err)
	}

	return ret, nil
}

func (w WebSession) Save(db anydb.AnyDb) error {
	_, err := db.Exec(
		"INSERT INTO web_sessions (session_id, context_id, context_type) VALUES ($1,$2,$3)",
		w.SessionId, w.ContextId, w.ContextType)
	if err != nil {
		return fmt.Errorf("saving session %q: %v", w.SessionId, err)
	}
	return nil
}

type NotFound error

func Load(db anydb.AnyDb, sessionId string) (*WebSession, error) {
	res, err := db.Query("SELECT session_id, context_id, context_type FROM web_Sessions WHERE session_id=$1", sessionId)
	if err != nil {
		return nil, fmt.Errorf("querying web_sessions for session %q: %v", sessionId, err)
	}
	defer res.Close()
	if !res.Next() {
		return nil, NotFound(fmt.Errorf("session %q not found", sessionId))
	}
	ret := &WebSession{}
	if err := res.Scan(&ret.SessionId, &ret.ContextId, &ret.ContextType); err != nil {
		return nil, fmt.Errorf("scanning session row: %s", err)
	}
	return ret, nil
}

func (w WebSession) GetContext(prov *context.ContextProvider) (context.Context, error) {
	provFunc, ok := prov.ContextTypes[w.ContextType]
	if !ok {
		return nil, fmt.Errorf("no context provider function for type %s", w.ContextType)
	}
	ctx, err := provFunc(w.ContextId)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}
