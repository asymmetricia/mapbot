package tabula

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/types"
	"image"
	"image/color"
	"image/draw"
)

func (t *Tabula) loadTokens(db *sql.DB) error {
	if t.Id == nil {
		return errors.New("cannot load tokens for tabula with nil ID")
	}
	// Read list of existing tokens
	res, err := db.Query("SELECT context_id, name, x, y, r, g, b, a FROM tabula_tokens WHERE tabula_id=$1", t.Id)
	if err != nil {
		return fmt.Errorf("retrieving list to sync: %s", err)
	}
	defer res.Close()

	t.Tokens = map[types.ContextId]map[string]Token{}
	for res.Next() {
		var ctxId types.ContextId
		var name string
		var x, y int
		var r, g, b, a uint8
		if err := res.Scan(&ctxId, &name, &x, &y, &r, &g, &b, &a); err != nil {
			log.Warningf("scanning row: %s", err)
			continue
		}
		if ctxTokens, ok := t.Tokens[ctxId]; !ok {
			t.Tokens[ctxId] = map[string]Token{
				name: {image.Point{x, y}, color.RGBA{r, g, b, a}},
			}
		} else {
			ctxTokens[name] = Token{
				image.Point{x, y},
				color.RGBA{r, g, b, a},
			}
		}
	}

	return nil
}

func (t *Tabula) saveTokens(db *sql.DB) error {
	if t.Id == nil {
		return errors.New("cannot save tokens for tabula with nil ID")
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("preparing transaction: %s", err)
	}
	// Read list of existing tokens
	res, err := tx.Query("SELECT name, context_id FROM tabula_tokens WHERE tabula_id=$1", t.Id)
	if err != nil {
		return fmt.Errorf("retrieving list to sync: %s", err)
	}
	tokens := map[types.ContextId]map[string]bool{}
	for res.Next() {
		var ctxId types.ContextId
		var name string
		res.Scan(&name, &ctxId)
		if tokens[ctxId] == nil {
			tokens[ctxId] = map[string]bool{
				name: true,
			}
		} else {
			tokens[ctxId][name] = true
		}
	}

	// Delete tokens not on tabula
	del, err := tx.Prepare("DELETE FROM tabula_tokens WHERE name=$1 AND tabula_id=$2 AND context_id=$3")
	if err != nil {
		return fmt.Errorf("error preparing DELETE: %s", err)
	}
	for ctxId, ctxTokens := range tokens {
		if _, ok := t.Tokens[ctxId]; !ok {
			// This context doesn't even exist on the table; delete all associated tokens
			for name := range ctxTokens {
				_, err := del.Exec(name, t.Id, ctxId)
				if err != nil {
					log.Warningf("error attempting to delete token %q on tabula %d with context %q: %s", name, t.Id, ctxId, err)
				}
			}
		} else {
			// The context exists, so check each token
			for name := range ctxTokens {
				if _, ok := t.Tokens[ctxId][name]; !ok {
					_, err := del.Exec(name, t.Id, ctxId)
					if err != nil {
						log.Warningf("error attempting to delete token %q on tabula %d with context %q: %s", name, t.Id, ctxId, err)
					}
				}
			}
		}
	}

	// Add Or Replace existing tokens
	add, err := tx.Prepare(
		"INSERT INTO tabula_tokens (name, context_id, tabula_id, x, y, r, g, b, a) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) " +
			"ON CONFLICT (name, context_id, tabula_id) DO UPDATE SET x=$4, y=$5, r=$6, g=$7, b=$8, a=$9",
	)
	if err != nil {
		return fmt.Errorf("error preparing ADD: %s", err)
	}
	for ctxId, ctxTokens := range t.Tokens {
		for name, token := range ctxTokens {
			pos := token.Coordinate
			r, g, b, a := token.TokenColor.RGBA()
			if _, err := add.Exec(name, ctxId, t.Id, pos.X, pos.Y, r>>8, g>>8, b>>8, a>>8); err != nil {
				log.Warningf("error saving token %q at pos (%d,%d) on tabula %d, context ID %q: %s", name, pos.X, pos.Y, t.Id, ctxId, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing changes: %s", err)
	}
	log.Debugf("Token save for tabula %d complete", t.Id)
	return nil
}

func (t *Tabula) drawAt(i draw.Image, obj image.Image, x float32, y float32, size float32, inset int) {
	oX := obj.Bounds().Dx()
	oY := obj.Bounds().Dy()
	tgt := uint(size*t.Dpi - 2*float32(inset))
	var scaled image.Image
	if oX > oY {
		scaled = resize.Resize(tgt, 0, obj, resize.Bilinear)
	} else {
		scaled = resize.Resize(0, tgt, obj, resize.Bilinear)
	}

	draw.Draw(
		i,
		image.Rect(
			int(x*t.Dpi)+t.OffsetX+int(inset), int(y*t.Dpi)+t.OffsetY+int(inset),
			int((x+1)*t.Dpi)+t.OffsetX-int(inset), int((y+1)*t.Dpi)+t.OffsetY-int(inset),
		),
		scaled,
		image.Pt(0, 0),
		draw.Over,
	)
}

func (t *Tabula) addTokens(in image.Image, ctx context.Context) error {
	drawable, ok := in.(draw.Image)
	if !ok {
		return errors.New("image provided could not be used as a draw.Image")
	}

	for name, token := range t.Tokens[ctx.Id] {
		coord := token.Coordinate
		r, g, b, a := token.TokenColor.RGBA()
		log.Debugf("Adding token %s (color:%d,%d,%d,%d) at (%d,%d)", name, r, g, b, a, coord.X, coord.Y)
		if ctx.IsEmoji(name) {
			e, err := ctx.GetEmoji(name)
			if err != nil {
				log.Warningf("error obtaining emoji %q: %s", name, err)
				// no return here, we'll fall through to rendering token name
			} else {
				t.squareAt(drawable, image.Rect(coord.X, coord.Y, coord.X+1, coord.Y+1), 1, token.TokenColor)
				t.drawAt(drawable, e, float32(coord.X), float32(coord.Y), 1, 2)
				continue
			}
		}
		//e, err := ctx.
		t.squareAt(drawable, image.Rect(coord.X, coord.Y, coord.X+1, coord.Y+1), 1, token.TokenColor)
		t.printAt(drawable, name, float32(coord.X), float32(coord.Y), 1)
	}
	return nil
}
