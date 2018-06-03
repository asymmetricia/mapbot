package tabula

import (
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"github.com/pdbogen/mapbot/common/db/anydb"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/mark"
	"github.com/pdbogen/mapbot/model/types"
	"image"
	"image/color"
	"image/draw"
	"regexp"
	"sort"
)

type Token struct {
	Coordinate                         image.Point
	TokenColor                         color.Color
	Size                               int
	DimLight, NormalLight, BrightLight int
}

func (t Token) Color() color.Color {
	if t.TokenColor != nil {
		return t.TokenColor
	}
	return color.RGBA{0, 0, 0, 0}
}

func (t Token) WithLight(dim, normal, bright int) (ret Token) {
	ret = t
	ret.DimLight = dim
	ret.NormalLight = normal
	ret.BrightLight = bright
	return
}

func (t Token) WithColor(c color.Color) (ret Token) {
	ret = t
	ret.TokenColor = c
	return
}

func (t Token) WithCoords(p image.Point) (ret Token) {
	ret = t
	ret.Coordinate = p
	return
}

func (t Token) WithSize(s int) (ret Token) {
	ret = t
	ret.Size = s
	return
}

func (t *Tabula) loadTokens(db anydb.AnyDb) error {
	if t.Id == nil {
		return errors.New("cannot load tokens for tabula with nil ID")
	}
	// Read list of existing tokens
	res, err := db.Query("SELECT context_id, name, size, x, y, r, g, b, a, light_dim, light_normal, light_bright FROM tabula_tokens WHERE tabula_id=$1", t.Id)
	if err != nil {
		return fmt.Errorf("retrieving list to sync: %s", err)
	}
	defer res.Close()

	t.Tokens = map[types.ContextId]map[string]Token{}
	for res.Next() {
		var ctxId types.ContextId
		var name string
		var x, y, size int
		var r, g, b, a uint8
		var dim, normal, bright int
		if err := res.Scan(&ctxId, &name, &size, &x, &y, &r, &g, &b, &a, &dim, &normal, &bright); err != nil {
			log.Warningf("scanning row: %s", err)
			continue
		}

		if _, ok := t.Tokens[ctxId]; !ok {
			t.Tokens[ctxId] = map[string]Token{}
		}

		t.Tokens[ctxId][name] = Token{
			Coordinate:  image.Point{x, y},
			TokenColor:  color.RGBA{r, g, b, a},
			Size:        size,
			DimLight:    dim,
			NormalLight: normal,
			BrightLight: bright,
		}
	}

	return nil
}

func (t *Tabula) saveTokens(db anydb.AnyDb) error {
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
	var query string
	switch dia := db.Dialect(); dia {
	case "postgresql":
		query = "INSERT INTO tabula_tokens (name, context_id, tabula_id, size, x, y, r, g, b, a, light_dim, light_normal, light_bright) " +
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, $11, $12, $13) " +
			"ON CONFLICT (name, context_id, tabula_id) DO UPDATE SET size=$4, x=$5, y=$6, r=$7, g=$8, b=$9, a=$10, light_dim = $11, light_normal=$12, light_bright=$13"
	case "sqlite3":
		query = "REPLACE INTO tabula_tokens (name, context_id, tabula_id, size, x, y, r, g, b, a, light_dim, light_normal, light_bright) " +
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, $11, $12, $13)"
	default:
		return fmt.Errorf("no Tabula.saveTokens query for SQL dialect %s", dia)
	}

	add, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing ADD: %s", err)
	}
	for ctxId, ctxTokens := range t.Tokens {
		for name, token := range ctxTokens {
			pos := token.Coordinate
			r, g, b, a := token.Color().RGBA()
			if _, err := add.Exec(name, ctxId, t.Id, token.Size, pos.X, pos.Y, r>>8, g>>8, b>>8, a>>8, token.DimLight, token.NormalLight, token.BrightLight); err != nil {
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

func (t *Tabula) drawAt(i draw.Image, obj image.Image, x float32, y float32, size float32, inset int, offset image.Point) {
	t.drawAtAlign(i, obj, x, y, size, inset, Middle, Center, offset)
}

func (t *Tabula) drawAtAlign(i draw.Image, obj image.Image, x float32, y float32, size float32, inset int, vert VerticalAlignment, horiz HorizontalAlignment, offset image.Point) {
	oX := obj.Bounds().Dx()
	oY := obj.Bounds().Dy()
	targetSize := uint(size*t.Dpi - 2*float32(inset))
	targetWidth := 0
	targetHeight := 0
	var scaled image.Image
	if oX > oY {
		scaled = resize.Resize(targetSize, 0, obj, resize.Bilinear)
		targetWidth = int(targetSize)
		targetHeight = int(targetSize) * oY / oX
	} else {
		scaled = resize.Resize(0, targetSize, obj, resize.Bilinear)
		targetWidth = int(targetSize) * oX / oY
		targetHeight = int(targetSize)
	}

	top := 0
	left := 0

	switch vert {
	case Top:
		top = 0
	case Middle:
		top = (int(size*t.Dpi) - 2*inset - targetHeight) / 2
	case Bottom:
		top = int(size*t.Dpi) - 2*inset - targetHeight
	}

	switch horiz {
	case Left:
		left = 0
	case Center:
		left = (int(size*t.Dpi) - 2*inset - targetWidth) / 2
	case Right:
		left = int(size*t.Dpi) - 2*inset - targetWidth
	}

	log.Debugf("should draw emoji %dx%d at (%d,%d)", targetWidth, targetHeight, left, top)

	draw.Draw(
		i,
		image.Rect(
			int(x*t.Dpi)+offset.X+int(inset)+left, int(y*t.Dpi)+offset.Y+int(inset)+top,
			int((x+1)*t.Dpi*size)+offset.X-int(inset)+left, int((y+1)*t.Dpi*size)+offset.Y-int(inset)+top,
		),
		scaled,
		image.Pt(0, 0),
		draw.Over,
	)
}

var emojiRe = regexp.MustCompile(`^(:[^:]+:)(.*)$`)

func light(t *Tabula, in image.Image, radius int, coord image.Point, col color.Color) ([]mark.Mark, error) {
	if radius <= 0 {
		return []mark.Mark{}, nil
	}
	marks, err := mark.CirclePoint(coord, "", radius)
	if err != nil {
		return nil, fmt.Errorf("rendering a circle radius %d at %v failed: %s", radius, coord, err)
	}
	for i := range marks {
		marks[i].Color = col
	}
	return marks, nil
}

func (t *Tabula) addTokenLights(in image.Image, ctx context.Context, offset image.Point) error {
	// Map out light levels; brightest lights win.
	lighting := map[image.Point]mark.Mark{}
	for tokenName, token := range t.Tokens[ctx.Id()] {
		if token.DimLight == 0 {
			continue
		}
		log.Debugf("adding dim lighting %dft for token %q at %v", token.DimLight, tokenName, token.Coordinate)
		// Add "dim lighting" marks
		marks, err := light(t, in, token.DimLight, token.Coordinate, color.NRGBA{231, 114, 0, 63})
		if err != nil {
			return fmt.Errorf("drawing lights: %s", err)
		}

		for _, m := range marks {
			lighting[m.Point] = m
		}
	}

	for tokenName, token := range t.Tokens[ctx.Id()] {
		if token.NormalLight == 0 {
			continue
		}
		log.Debugf("adding normal lighting %dft for token %q at %v", token.NormalLight, tokenName, token.Coordinate)
		// Add "normal lighting" marks
		marks, err := light(t, in, token.NormalLight, token.Coordinate, color.NRGBA{250, 250, 55, 63})
		if err != nil {
			return fmt.Errorf("drawing lights: %s", err)
		}

		for _, m := range marks {
			lighting[m.Point] = m
		}
	}

	for tokenName, token := range t.Tokens[ctx.Id()] {
		if token.BrightLight == 0 {
			continue
		}
		log.Debugf("adding bright lighting %dft for token %q at %v", token.BrightLight, tokenName, token.Coordinate)
		// Add "bright lighting" marks
		marks, err := light(t, in, token.BrightLight, token.Coordinate, color.NRGBA{149, 224, 232, 63})
		if err != nil {
			return fmt.Errorf("drawing lights: %s", err)
		}

		for _, m := range marks {
			lighting[m.Point] = m
		}
	}

	// Render all marks
	marks := []mark.Mark{}
	for _, m := range lighting {
		marks = append(marks, m)
	}
	return t.addMarkSlice(in, marks, offset)
}

type TokensByNameThenSize struct {
	names  []string
	tokens map[string]Token
}

func (t *TokensByNameThenSize) Len() int {
	return len(t.names)
}

func (t *TokensByNameThenSize) Less(a, b int) bool {
	return t.names[a] < t.names[b] ||
		(t.names[a] == t.names[b] && t.tokens[t.names[a]].Size < t.tokens[t.names[b]].Size)
}

func (t *TokensByNameThenSize) Swap(a, b int) {
	t.names[a], t.names[b] = t.names[b], t.names[a]
}

var _ sort.Interface = (*TokensByNameThenSize)(nil)

func (t *Tabula) addTokens(in image.Image, ctx context.Context, offset image.Point) error {
	drawable, ok := in.(draw.Image)
	if !ok {
		return errors.New("image provided could not be used as a draw.Image")
	}

	tokens := t.Tokens[ctx.Id()]
	names := make([]string, len(tokens))
	n := 0
	for name := range tokens {
		names[n] = name
		n++
	}
	sort.Sort(&TokensByNameThenSize{names, tokens})

	for _, tokenName := range names {
		token := tokens[tokenName]
		coord := token.Coordinate
		r, g, b, a := token.Color().RGBA()

		var name, label string

		comps := emojiRe.FindStringSubmatch(tokenName)
		if comps == nil {
			name = tokenName
		} else {
			name = comps[1]
			label = comps[2]
		}

		log.Debugf("Adding token (name=%q) (label=%q) (color:%d,%d,%d,%d) at (%d,%d)", name, label, r, g, b, a, coord.X, coord.Y)

		if a > 0 {
			t.squareAt(drawable, image.Rect(coord.X, coord.Y, coord.X+token.Size, coord.Y+token.Size), 1, token.Color(), offset)
		}

		if ctx.IsEmoji(name) {
			emoji, err := ctx.GetEmoji(name)
			if err != nil {
				log.Warningf("error obtaining emoji %q: %s", name, err)
				// no return here, we'll fall through to rendering token name
			} else {
				t.drawAtAlign(drawable, emoji, float32(coord.X), float32(coord.Y), float32(token.Size), 2, Middle, Center, offset)
				if label != "" {
					t.printAt(drawable, label, float32(coord.X), float32(coord.Y)+float32(token.Size)/2, float32(token.Size), float32(token.Size)/2, Bottom, Center, offset)
				}
				continue
			}
		}
		t.printAt(drawable, name, float32(coord.X), float32(coord.Y), float32(token.Size), float32(token.Size), Middle, Center, offset)
	}
	return nil
}
