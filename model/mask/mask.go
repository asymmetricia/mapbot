package mask

import (
	"database/sql"
	"image/color"
	"fmt"
	"errors"
)

type Mask struct {
	Name   string
	Color  color.NRGBA
	Order  *int
	Top    int
	Left   int
	Width  int
	Height int
	Clear  bool
}

func (m *Mask) Up(db *sql.DB) error {
	if m.Order == nil {
		return fmt.Errorf("cannot reorder mask %q not saved to DB", m.Name)
	}
	if *m.Order == 0 {
		return fmt.Errorf("mask %q is already first", m.Name)
	}
	_, err := db.Exec(
		`UPDATE tabula_masks SET "order"=$1 WHERE "order"=$2; UPDATE tabula_masks SET "order"=$2 WHERE name=$3`,
		*m.Order, *m.Order-1, m.Name,
	)
	if err != nil {
		return fmt.Errorf("swapping masks: %s", err)
	}
	return nil
}

func (m *Mask) Save(db *sql.DB, id int64) error {
	if m.Order == nil {
		res, err := db.Query(`SELECT MAX("order")+1 FROM tabula_masks WHERE tabula_id=$1`, id)
		if err != nil {
			return fmt.Errorf("determining next order: %s", err)
		}
		defer res.Close()
		if !res.Next() {
			return errors.New("no result querying next order")
		}
		m.Order = new(int)
		if err := res.Scan(m.Order); err != nil {
			return fmt.Errorf("retrieving order: %s", err)
		}
	}

	_, err := db.Exec(
		`INSERT INTO tabula_masks (name, "order", tabula_id, red, green, blue, alpha, top, "left", width, height) ` +
			`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ` +
			`ON CONFLICT (name, tabula_id) DO UPDATE ` +
			`SET "order"=$2, red=$4, green=$5, blue=$6, alpha=$7, top=$8, "left"=$9, width=$10, height=$11`,
		m.Name, *m.Order, id, m.Color.R, m.Color.G, m.Color.B, m.Color.A, m.Top, m.Left, m.Width, m.Height,
	)
	if err != nil {
		return err
	}

	return nil
}
