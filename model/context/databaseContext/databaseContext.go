package databaseContext

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/model/types"
)

type DatabaseContext struct {
	ContextId              types.ContextId
	ActiveTabulaId         *types.TabulaId
	MinX, MinY, MaxX, MaxY int
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
	var query string
	switch dia := db.Instance.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO contexts (context_id, active_tabula, MinX, MinY, MaxX, MaxY) VALUES ($1,$2,$3,$4,$5,$6) " +
			"ON CONFLICT (context_id) DO UPDATE SET active_tabula=$2, MinX=$3, MinY=$4, MaxX=$5, MaxY=$6"

	case "sqlite3":
		query = "REPLACE INTO contexts (context_id, active_tabula, MinX, MinY, MaxX, MaxY) VALUES ($1,$2,$3,$4,$5,$6)"
	default:
		return fmt.Errorf("no DatabaseContext.Save query for dialect %s", dia)
	}
	_, err := db.Instance.Exec(query, dc.ContextId, int(*dc.ActiveTabulaId), dc.MinX, dc.MinY, dc.MaxX, dc.MaxY)
	return err
}

func (dc *DatabaseContext) Load() error {
	res, err := db.Instance.Query("SELECT active_tabula, MinX, MinY, MaxX, MaxY FROM contexts WHERE context_id=$1", dc.ContextId)
	if err != nil {
		return err
	}
	defer res.Close()

	if !res.Next() {
		return nil
	}

	dc.ActiveTabulaId = new(types.TabulaId)

	if err := res.Scan(&dc.ActiveTabulaId, &dc.MinX, &dc.MinY, &dc.MaxX, &dc.MaxY); err != nil {
		return fmt.Errorf("retrieving columns: %s", err)
	}

	return nil
}

func (dc *DatabaseContext) GetZoom() (MinX, MinY, MaxX, MaxY int) {
	return dc.MinX, dc.MinY, dc.MaxX, dc.MaxY
}

func (dc *DatabaseContext) SetZoom(MinX, MinY, MaxX, MaxY int) {
	dc.MinX = MinX
	dc.MinY = MinY
	dc.MaxX = MaxX
	dc.MaxY = MaxY
}
