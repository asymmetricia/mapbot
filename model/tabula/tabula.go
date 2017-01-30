// Tabula models an individual map. This includes the URL form which the background image is retrieved, the background
// image itself; and information about how to overlay a grid. In the future, tokens, masks, and overlays will also be
// included.
package tabula

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/nfnt/resize"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/mask"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var log = mbLog.Log

type TabulaName string

type Tabula struct {
	Id         *TabulaId
	Name       TabulaName
	Url        string
	Background *image.RGBA
	OffsetX    int
	OffsetY    int
	Dpi        float32
	GridColor  *color.NRGBA
	Masks      map[string]*mask.Mask
	Tokens     map[string]image.Point
}

type TabulaId int64

func (i *TabulaId) Value() (driver.Value, error) {
	return int64(*i), nil
}

var _ driver.Valuer = (*TabulaId)(nil)

func (t *Tabula) String() string {
	return fmt.Sprintf("Tabula{id=%d,Name=%s,Url=%s,Offset=(%d,%d),Dpi=%f}", t.Id, t.Name, t.Url, t.OffsetX, t.OffsetY, t.Dpi)
}

func Get(db *sql.DB, id TabulaId) (*Tabula, error) {
	res, err := db.Query("SELECT name, url, offset_x, offset_y, dpi, grid_r, grid_g, grid_b, grid_a FROM tabulas WHERE id=$1", int64(id))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, errors.New("not found")
	}

	ret := &Tabula{
		Id: new(TabulaId),
	}

	*ret.Id = id

	ret.GridColor = &color.NRGBA{}

	if err := res.Scan(
		&(ret.Name), &(ret.Url), &(ret.OffsetX), &(ret.OffsetY), &(ret.Dpi),
		&(ret.GridColor.R), &(ret.GridColor.G), &(ret.GridColor.B), &(ret.GridColor.A),
	); err != nil {
		return nil, fmt.Errorf("retrieving columns: %s", err)
	}

	if err := ret.loadMasks(db); err != nil {
		return nil, fmt.Errorf("loading masks: %s", err)
	}

	return ret, nil
}

func (t *Tabula) loadMasks(db *sql.DB) error {
	res, err := db.Query(`SELECT name, "order", red, green, blue, alpha, top, "left", width, height `+
		`FROM tabula_masks WHERE tabula_id=$1 ORDER BY "order"`, int64(*t.Id))
	if err != nil {
		return err
	}
	t.Masks = map[string]*mask.Mask{}
	defer res.Close()
	for res.Next() {
		m := &mask.Mask{}
		err := res.Scan(
			&m.Name,
			&m.Order,
			&m.Color.R, &m.Color.G, &m.Color.B, &m.Color.A,
			&m.Top, &m.Left,
			&m.Width, &m.Height,
		)
		if err != nil {
			return err
		}
		t.Masks[m.Name] = m
	}
	return nil
}

func (t *Tabula) Save(db *sql.DB) error {
	if t.GridColor == nil {
		t.GridColor = &color.NRGBA{A: 255}
	}
	r, g, b, a := t.GridColor.R, t.GridColor.G, t.GridColor.B, t.GridColor.A

	if t.Id == nil {
		result, err := db.Query(
			"INSERT INTO tabulas (name, url, offset_x, offset_y, dpi, grid_r, grid_g, grid_b, grid_a) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) "+
				"RETURNING id",
			string(t.Name), t.Url, t.OffsetX, t.OffsetY, t.Dpi, r, g, b, a,
		)
		if err != nil {
			return err
		}

		if !result.Next() {
			return errors.New("missing tabula ID from query result")
		}

		var i int
		if err := result.Scan(&i); err != nil {
			return err
		} else {
			t.Id = new(TabulaId)
			*t.Id = TabulaId(i)
		}
	} else {
		_, err := db.Exec(
			"INSERT INTO tabulas (id, name, url, offset_x, offset_y, dpi, grid_r, grid_g, grid_b, grid_a) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) "+
				"ON CONFLICT (id) DO UPDATE SET name=$2, url=$3, offset_x=$4, offset_y=$5, dpi=$6, "+
				"grid_r=$7, grid_g=$8, grid_b=$9, grid_a=$10",
			int64(*t.Id), string(t.Name), t.Url, t.OffsetX, t.OffsetY, t.Dpi, r, g, b, a,
		)
		if err != nil {
			return err
		}
	}

	if t.Masks != nil {
		for _, m := range t.Masks {
			if err := m.Save(db, int64(*t.Id)); err != nil {
				return err
			}
		}
	}

	return nil
}

// New attempts to create a new map from the image in the given URL.
func New(name, url string) (*Tabula, error) {
	return &Tabula{
		Name: TabulaName(name),
		Url:  url,
		//Background: ret,
		Dpi: 10,
	}, nil
}

func (t *Tabula) Hydrate() error {
	if t.Background != nil {
		return nil
	}

	c := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := c.Get(t.Url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	img, _, err := image.Decode(res.Body)
	if err != nil {
		return err
	}

	var ret *image.RGBA
	switch i := img.(type) {
	case (*image.RGBA):
		ret = i
	default:
		ret = image.NewRGBA(img.Bounds())
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				ret.Set(x, y, img.At(x, y))
			}
		}
	}

	t.Background = ret
	return nil
}

// BlendAt alters the point (given by (x,y)) in the image i by blending the color there with the color c
func blendAt(i *image.RGBA, x, y int, c color.Color) {
	i.Set(x, y, blend(c, i.RGBAAt(x, y)))
}

// blend calculates the result of alpha blending of the two colors
func blend(a color.Color, b color.Color) color.Color {
	a_r, a_g, a_b, a_a := a.RGBA()
	b_r, b_g, b_b, b_a := b.RGBA()
	return &color.RGBA{
		R: uint8((a_r + b_r*(0xFFFF-a_a)/0xFFFF) >> 8),
		G: uint8((a_g + b_g*(0xFFFF-a_a)/0xFFFF) >> 8),
		B: uint8((a_b + b_b*(0xFFFF-a_a)/0xFFFF) >> 8),
		A: uint8((a_a + b_a*(0xFFFF-a_a)/0xFFFF) >> 8),
	}
}

func addGrid(i image.Image, xOff float32, yOff float32, Dpi float32, col color.Color) image.Image {
	bounds := i.Bounds()
	gridded := copyImage(i)

	for xOff > 0 {
		xOff -= Dpi
	}
	for yOff > 0 {
		yOff -= Dpi
	}

	if col == nil {
		col = &color.Black
	}

	for x := xOff; x < float32(bounds.Max.X); x += Dpi {
		if x < 0 {
			continue
		}
		for y := yOff; y < float32(bounds.Max.Y); y++ {
			if y < 0 {
				continue
			}
			blendAt(gridded, int(x), int(y), col)
		}
	}

	// Horizontal lines; X at DPI intervals, all Y
	for x := xOff; x < float32(bounds.Max.X); x++ {
		if x < 0 {
			continue
		}
		for y := yOff; y < float32(bounds.Max.Y); y += Dpi {
			if y < 0 {
				continue
			}
			blendAt(gridded, int(x), int(y), col)
		}
	}

	return gridded
}

var font *truetype.Font

func init() {
	path := "fonts/DejaVuSerif.ttf"
	fontData, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("reading %s: %s", path, err))
	}

	font, err = freetype.ParseFont(fontData)
	if err != nil {
		panic(fmt.Sprintf("parsing %s: %s", path, err))
	}
}

var coordCache map[float32]map[string]*image.RGBA

func glyph(s string, dpi float32) *image.RGBA {
	if coordCache == nil {
		coordCache = map[float32]map[string]*image.RGBA{}
	}

	if _, ok := coordCache[dpi]; !ok {
		coordCache[dpi] = map[string]*image.RGBA{}
	}

	if i, ok := coordCache[dpi][s]; ok {
		return i
	}

	w := 100 + (len(s)+1)*int(dpi)
	h := 100 + int(dpi)

	img := image.NewRGBA(image.Rect(0, 0, w, h))

	ctx := freetype.NewContext()
	ctx.SetClip(img.Bounds())
	ctx.SetDst(img)
	ctx.SetFont(font)
	ctx.SetDPI(float64(dpi))
	ctx.SetFontSize(70 / float64(len(s)))
	ctx.SetSrc(image.Black)

	for _, x := range []int{-2, 0, 2} {
		for _, y := range []int{-2, 0, 3} {
			ctx.DrawString(s, fixed.Point26_6{X: fixed.I(50 + x), Y: fixed.I(50 + y)})
		}
	}

	ctx.SetSrc(image.White)
	ctx.DrawString(s, fixed.Point26_6{X: fixed.I(50), Y: fixed.I(50)})

	img = center(crop(img), int(dpi))
	coordCache[dpi][s] = img
	return img
}

func center(i *image.RGBA, dim int) *image.RGBA {
	offsetX := (dim - i.Bounds().Dx()) / 2
	offsetY := (dim - i.Bounds().Dy()) / 2
	result := image.NewRGBA(image.Rect(0, 0, dim, dim))
	for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
		for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
			result.Set(x+offsetX, y+offsetY, i.At(x, y))
		}
	}
	return result
}

func crop(i *image.RGBA) *image.RGBA {
	min_x := i.Bounds().Min.X
	min_y := i.Bounds().Min.Y
	max_x := i.Bounds().Max.X
	max_y := i.Bounds().Max.Y

min_y:
	for y := min_y; y < max_y; y++ {
		for x := min_x; x < max_x; x++ {
			_, _, _, a := i.At(x, y).RGBA()
			if a > 0 {
				min_y = y
				break min_y
			}
		}
	}

max_y:
	for y := max_y; y > min_y; y-- {
		for x := min_x; x < max_x; x++ {
			_, _, _, a := i.At(x, y).RGBA()
			if a > 0 {
				max_y = y
				break max_y
			}
		}
	}

min_x:
	for x := min_x; x < max_x; x++ {
		for y := min_y; y < max_y; y++ {
			_, _, _, a := i.At(x, y).RGBA()
			if a > 0 {
				min_x = x
				break min_x
			}
		}
	}

max_x:
	for x := max_x; x > min_x; x-- {
		for y := min_y; y < max_y; y++ {
			_, _, _, a := i.At(x, y).RGBA()
			if a > 0 {
				max_x = x
				break max_x
			}
		}
	}

	result := image.NewRGBA(image.Rect(0, 0, max_x-min_x, max_y-min_y))
	for x := min_x; x < max_x; x++ {
		for y := min_y; y < max_y; y++ {
			result.Set(x-min_x, y-min_y, i.At(x, y))
		}
	}
	return result
}

func addCoordinates(i image.Image, xOff float32, yOff float32, dpi float32) image.Image {
	letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
		"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	result := copyImage(i)

	rows := int(float32(i.Bounds().Max.Y) / dpi)
	cols := int(float32(i.Bounds().Max.X) / dpi)
	// 0 1 2 3 4 ... 25 26 27 28
	// A B C D E ... Y  Z  AA AB
	for x := 0; x < cols; x++ {
		c := letters[x%26] // 0..25
		tmp := x
		for tmp > 25 {
			tmp = int(tmp/26) - 1
			c = letters[tmp%26] + c
		}

		g := glyph(c, dpi/2)
		draw.Draw(
			result,
			image.Rect(
				int(float32(x)*dpi), int(0),
				int(float32(x+1)*dpi), int(dpi/2),
			),
			g,
			image.Pt(0, 0),
			draw.Over,
		)
	}

	for y := 0; y < rows; y++ {
		g := glyph(strconv.Itoa(y+1), dpi/2)
		draw.Draw(
			result,
			image.Rect(
				int(dpi/2), int(dpi*float32(y)+dpi/2),
				int(dpi), int(dpi*float32(y+1)),
			),
			g,
			image.Pt(0, 0),
			draw.Over,
		)
	}

	return result
}

func addTokens(in image.Image, xOff float32, yOff float32, dpi float32, tokens map[string]image.Point) image.Image {
	//ctx, err := context.Load(db.Instance, ctxId)
	//if err != nil {
	//	log.Warningf("retrieving context %q failed; skipping token rendering: %s", ctxId, err)
	//	return in
	//}
	//
	//for name, coord := range ctx.Tokens {
	//}
	return in
}

var cache = map[string]image.Image{}

func (t *Tabula) Render(ctxId string) (image.Image, error) {
	if t.Dpi == 0 {
		return nil, errors.New("cannot render tabula with zero DPI")
	}

	var resized image.Image
	if cached, ok := cache[t.Url]; ok {
		resized = cached
	} else {
		log.Info("Cache miss: %s", t.Url)
		if t.Background == nil {
			if err := t.Hydrate(); err != nil {
				return nil, fmt.Errorf("retrieving background: %s", err)
			}
		}

		dx := t.Background.Rect.Dx()
		dy := t.Background.Rect.Dy()
		resized = t.Background
		if dx > 2000 || dy > 2000 {
			if t.Background.Rect.Dx() > t.Background.Rect.Dy() {
				resized = resize.Resize(2000, 0, t.Background, resize.Bilinear)
			} else {
				resized = resize.Resize(0, 2000, t.Background, resize.Bilinear)
			}
		}
		cache[t.Url] = resized
	}
	gridded := addGrid(resized, float32(t.OffsetX), float32(t.OffsetY), t.Dpi, t.GridColor)
	coord := addCoordinates(gridded, float32(t.OffsetX), float32(t.OffsetY), t.Dpi)
	tokened := addTokens(coord, float32(t.OffsetX), float32(t.OffsetY), t.Dpi, t.Tokens)

	return tokened, nil
}

func copyImage(in image.Image) *image.RGBA {
	out := image.NewRGBA(in.Bounds())
	for x := 0; x < in.Bounds().Max.X; x++ {
		for y := 0; y < in.Bounds().Max.Y; y++ {
			out.Set(x, y, in.At(x, y))
		}
	}
	return out
}
