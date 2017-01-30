package context

import (
	"database/sql"
	"fmt"
	"github.com/pdbogen/mapbot/model/tabula"
)

type Context struct {
	Id           string
	ActiveTabula *tabula.Tabula
}

func Load(db *sql.DB, id string) (*Context, error) {
	res, err := db.Query("SELECT active_tabula FROM contexts WHERE context_id=$1", id)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	ret := &Context{
		Id:     id,
	}

	if !res.Next() {
		return ret, nil
	}

	//ret.ActiveTabula = new(tabula.TabulaId)
	tabula_id := new(tabula.TabulaId)

	if err := res.Scan(&tabula_id); err != nil {
		return nil, fmt.Errorf("retrieving columns: %s", err)
	}

	if tabula_id != nil {
		ret.ActiveTabula, err = tabula.Get(db, *tabula_id)
		if err != nil {
			return nil, fmt.Errorf("loading active tabula %d: %s", int(*tabula_id), err)
		}
	}

	return ret, nil
}

func (c *Context) Save(db *sql.DB) error {
	_, err := db.Exec(
		"INSERT INTO contexts (context_id, active_tabula) VALUES ($1,$2) "+
			"ON CONFLICT (context_id) DO UPDATE SET active_tabula=$2",
		c.Id, int(*c.ActiveTabula.Id),
	)
	return err
}