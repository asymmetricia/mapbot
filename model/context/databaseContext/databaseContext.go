package databaseContext

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/model/types"
)

type DatabaseContext struct {
	ContextId      types.ContextId
	ActiveTabulaId *types.TabulaId
}

func (dc *DatabaseContext) Id() types.ContextId {
	return dc.ContextId
}

func (dc *DatabaseContext) SetActiveTabulaId(tabId *types.TabulaId) {
	if tabId == nil {
		dc.ActiveTabulaId = nil
	} else {
		dc.ActiveTabulaId = new(types.TabulaId)
		*dc.ActiveTabulaId = *tabId
	}
}

func (dc *DatabaseContext) GetActiveTabulaId() *types.TabulaId {
	return dc.ActiveTabulaId
}

func (dc *DatabaseContext) Save() error {
	_, err := db.Instance.Exec(
		"INSERT INTO contexts (context_id, active_tabula) VALUES ($1,$2) "+
			"ON CONFLICT (context_id) DO UPDATE SET active_tabula=$2",
		dc.ContextId, int(*dc.ActiveTabulaId),
	)
	return err
}

func (dc *DatabaseContext) Load() error {
	res, err := db.Instance.Query("SELECT active_tabula FROM contexts WHERE context_id=$1", dc.ContextId)
	if err != nil {
		return err
	}
	defer res.Close()

	if !res.Next() {
		return nil
	}

	dc.ActiveTabulaId = new(types.TabulaId)

	if err := res.Scan(&dc.ActiveTabulaId); err != nil {
		return fmt.Errorf("retrieving columns: %s", err)
	}

	return nil
}
