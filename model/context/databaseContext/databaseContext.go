package databaseContext

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/model/types"
	"image"
	"image/color"
)

type DatabaseContext struct {
	ContextId              types.ContextId
	ActiveTabulaId         *types.TabulaId
	MinX, MinY, MaxX, MaxY int
	Marks                  map[types.TabulaId]map[image.Point]color.Color
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
	if err != nil {
		return err
	}
	return dc.saveMarks()
}

func (dc *DatabaseContext) saveMarks() error {
	var query string
	switch dia := db.Instance.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO context_marks (context_id, tabula_id, square_x, square_y, red, green, blue, alpha) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) " +
			"ON CONFLICT (context_id) DO UPDATE SET red=$5, green=$6, blue=$7, alpha=$8"
	case "sqlite3":
		query = "REPLACE INTO context_marks (context_id, tabula_id, square_x, square_y, red, green, blue, alpha) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)"
	default:
		return fmt.Errorf("no DatabaseContext.saveMarks query for dialect %s", dia)
	}
	stmt, err := db.Instance.Prepare(query)
	if err != nil {
		return fmt.Errorf("preparing DatabaseContext.saveMarks query: %s", err)
	}
	for tabId, tabMarks := range dc.Marks {
		for pt, col := range tabMarks {
			r, g, b, a := col.RGBA()
			if _, err := stmt.Exec(dc.ContextId, tabId, pt.X, pt.Y, r, g, b, a); err != nil {
				return fmt.Errorf("executing DatabaseContext.saveMarks for (%v,%v,%v,%v): %s", dc.ContextId, tabId, pt, col, err)
			}
		}
	}
	return nil
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

func (dc *DatabaseContext) Mark(tid types.TabulaId, point image.Point, col color.Color) {
	if dc.Marks == nil {
		dc.Marks = map[types.TabulaId]map[image.Point]color.Color{}
	}
	if _, ok := dc.Marks[tid]; !ok {
		dc.Marks[tid] = map[image.Point]color.Color{}
	}
	dc.Marks[tid][point] = col
}

func (dc *DatabaseContext) GetMarks(tid types.TabulaId) map[image.Point]color.Color {
	return dc.Marks[tid]
}

func (dc *DatabaseContext) ClearMarks(tid types.TabulaId) {
	delete(dc.Marks, tid)
}
