package databaseContext

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/model/mark"
	"github.com/pdbogen/mapbot/model/types"
	"image"
	"image/color"
)

type DatabaseContext struct {
	ContextId              types.ContextId
	ActiveTabulaId         *types.TabulaId
	MinX, MinY, MaxX, MaxY int
	Marks                  map[types.TabulaId]map[image.Point]map[string]mark.Mark
	LastTokens             map[types.UserId]string
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
	if _, err := db.Instance.Exec(query, dc.ContextId, int(*dc.ActiveTabulaId), dc.MinX, dc.MinY, dc.MaxX, dc.MaxY); err != nil {
		return err
	}
	if err := dc.saveMarks(); err != nil {
		return err
	}
	return dc.saveLastTokens()
}

func (dc *DatabaseContext) saveLastTokens() error {
	var query string
	switch dia := db.Instance.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO last_token (context_id, user_id, token_name) VALUES ($1, $2, $3) ON CONFLICT (context_id, user_id) DO UPDATE SET token_name=$3"
	case "sqlite3":
		query = "REPLACE INTO last_token (context_id, user_id, token_name) VALUES ($1, $2, $3)"
	default:
		return fmt.Errorf("no DatabaseContext.saveLastTokens query for dialect %s", dia)
	}

	stmt, err := db.Instance.Prepare(query)
	if err != nil {
		return fmt.Errorf("preparing DatabaseContext.saveLastTokens query: %s", err)
	}

	for user, token := range dc.LastTokens {
		if _, err := stmt.Exec(dc.ContextId, user, token); err != nil {
			return fmt.Errorf("executing DatabaseContext.saveLastTokens for (%v,%v,%v): %s", dc.ContextId, user, token, err)
		}
	}
	return nil
}

func (dc *DatabaseContext) saveMarks() error {
	var query string

	if _, err := db.Instance.Exec("UPDATE context_marks SET stale=TRUE WHERE context_id=$1", dc.ContextId); err != nil {
		return err
	}

	switch dia := db.Instance.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO context_marks (context_id, tabula_id, square_x, square_y, direction, red, green, blue, alpha, stale) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,FALSE) " +
			"ON CONFLICT (context_id, tabula_id, square_x, square_y, direction) DO UPDATE SET red=$6, green=$7, blue=$8, alpha=$9, stale=FALSE"
	case "sqlite3":
		query = "REPLACE INTO context_marks (context_id, tabula_id, square_x, square_y, direction, red, green, blue, alpha, stale) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,FALSE)"
	default:
		return fmt.Errorf("no DatabaseContext.saveMarks query for dialect %s", dia)
	}
	stmt, err := db.Instance.Prepare(query)
	if err != nil {
		return fmt.Errorf("preparing DatabaseContext.saveMarks query: %s", err)
	}
	for tabId, tabMarks := range dc.Marks {
		for _, dirMarks := range tabMarks {
			for _, mark := range dirMarks {
				r, g, b, a := mark.Color.RGBA()
				if _, err := stmt.Exec(dc.ContextId, tabId, mark.Point.X, mark.Point.Y, mark.Direction, r>>8, g>>8, b>>8, a>>8); err != nil {
					return fmt.Errorf("executing DatabaseContext.saveMarks for (%v,%v,%v,%v): %s", dc.ContextId, tabId, mark.Point, mark.Color, err)
				}
			}
		}
	}

	if _, err := db.Instance.Exec("DELETE FROM context_marks WHERE context_id=$1 AND stale=TRUE", dc.ContextId); err != nil {
		return err
	}

	return nil
}

func (dc *DatabaseContext) loadMarks() error {
	res, err := db.Instance.Query("SELECT tabula_id, square_x, square_y, direction, red, green, blue, alpha FROM context_marks WHERE context_id=$1", dc.ContextId)
	if err != nil {
		return fmt.Errorf("querying context_marks: %s", err)
	}
	defer res.Close()

	var t types.TabulaId
	var x, y int
	var r, g, b, a uint8
	var d string

	for res.Next() {
		if err := res.Scan(&t, &x, &y, &d, &r, &g, &b, &a); err != nil {
			return fmt.Errorf("retrieving mark: %s", err)
		}
		dc.Mark(t, mark.Mark{Point: image.Point{x, y}, Color: color.RGBA{r, g, b, a}, Direction: d})
	}
	return nil
}

func (dc *DatabaseContext) Load() error {
	if err := dc.loadMarks(); err != nil {
		return err
	}

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

func (dc *DatabaseContext) Mark(tid types.TabulaId, mk mark.Mark) {
	if dc.Marks == nil {
		dc.Marks = map[types.TabulaId]map[image.Point]map[string]mark.Mark{}
	}
	if _, ok := dc.Marks[tid]; !ok {
		dc.Marks[tid] = map[image.Point]map[string]mark.Mark{}
	}
	if _, ok := dc.Marks[tid][mk.Point]; !ok {
		dc.Marks[tid][mk.Point] = map[string]mark.Mark{}
	}
	dc.Marks[tid][mk.Point][mk.Direction] = mk
}

func (dc *DatabaseContext) GetMarks(tid types.TabulaId) map[image.Point]map[string]mark.Mark {
	return dc.Marks[tid]
}

func (dc *DatabaseContext) ClearMarks(tid types.TabulaId) {
	delete(dc.Marks, tid)
}

func (dc *DatabaseContext) GetLastToken(UserId types.UserId) (TokenName string) {
	return dc.LastTokens[UserId]
}

func (dc *DatabaseContext) SetLastToken(UserId types.UserId, TokenName string) {
	dc.LastTokens[UserId] = TokenName
}
