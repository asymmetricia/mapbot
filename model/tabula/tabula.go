// Tabula models an individual map. This includes the URL form which the background image is retrieved, the background
// image itself; and information about how to overlay a grid. In the future, tokens, masks, and overlays will also be
// included.
package tabula

import (
	"errors"
	"fmt"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/nfnt/resize"
	"github.com/pdbogen/mapbot/common/db/anydb"
	mbDraw "github.com/pdbogen/mapbot/common/draw"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/mark"
	"github.com/pdbogen/mapbot/model/mask"
	"github.com/pdbogen/mapbot/model/types"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"
)

var log = mbLog.Log

type TabulaName string

type Tabula struct {
	Id         *types.TabulaId
	Name       TabulaName
	Url        string
	Background *image.RGBA
	OffsetX    int
	OffsetY    int
	Dpi        float32
	GridColor  color.Color
	Masks      map[string]*mask.Mask
	Tokens     map[types.ContextId]map[string]Token
	Version    int

	// A list of marks appended to marks obtained from the context during rendering.
	Marks []mark.Mark
}

func (t *Tabula) String() string {
	return fmt.Sprintf("Tabula{id=%d,Name=%s,Url=%s,Offset=(%d,%d),Dpi=%f}", t.Id, t.Name, t.Url, t.OffsetX, t.OffsetY, t.Dpi)
}

var tabulaeInMemory = map[types.TabulaId]*Tabula{}

func Get(db anydb.AnyDb, id types.TabulaId) (*Tabula, error) {
	if t, ok := tabulaeInMemory[id]; ok {
		return t, nil
	}

	res, err := db.Query("SELECT name, url, offset_x, offset_y, dpi, grid_r, grid_g, grid_b, grid_a, version FROM tabulas WHERE id=$1", int64(id))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, errors.New("not found")
	}

	ret := &Tabula{
		Id: new(types.TabulaId),
	}

	*ret.Id = id

	var r, g, b, a uint16

	if err := res.Scan(
		&(ret.Name), &(ret.Url), &(ret.OffsetX), &(ret.OffsetY), &(ret.Dpi),
		&r, &g, &b, &a,
		&(ret.Version),
	); err != nil {
		return nil, fmt.Errorf("retrieving columns: %s", err)
	}

	ret.GridColor = &color.RGBA{
		uint8(r >> 8),
		uint8(g >> 8),
		uint8(b >> 8),
		uint8(a >> 8),
	}

	if err := ret.loadMasks(db); err != nil {
		return nil, fmt.Errorf("loading masks: %s", err)
	}

	if err := ret.loadTokens(db); err != nil {
		return nil, fmt.Errorf("loading tokens: %s", err)
	}

	return ret, nil
}

func (t *Tabula) loadMasks(db anydb.AnyDb) error {
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

func (t *Tabula) Save(db anydb.AnyDb) error {
	if t.GridColor == nil {
		t.GridColor = &color.NRGBA{A: 255}
	}
	r, g, b, a := t.GridColor.RGBA()

	if t.Id == nil {
		result, err := db.Query(
			"INSERT INTO tabulas (name, url, offset_x, offset_y, dpi, grid_r, grid_g, grid_b, grid_a, version) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) "+
				"RETURNING id",
			string(t.Name), t.Url, t.OffsetX, t.OffsetY, t.Dpi, r, g, b, a, t.Version,
		)
		if err != nil {
			return err
		}
		defer result.Close()

		if !result.Next() {
			return errors.New("missing tabula ID from query result")
		}

		var i int
		if err := result.Scan(&i); err != nil {
			return err
		} else {
			t.Id = new(types.TabulaId)
			*t.Id = types.TabulaId(i)
		}
	} else {
		var query string
		switch dia := db.Dialect(); dia {
		case "postgresql":
			query = "INSERT INTO tabulas (id, name, url, offset_x, offset_y, dpi, grid_r, grid_g, grid_b, grid_a) " +
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) " +
				"ON CONFLICT (id) DO UPDATE SET name=$2, url=$3, offset_x=$4, offset_y=$5, dpi=$6, " +
				"grid_r=$7, grid_g=$8, grid_b=$9, grid_a=$10"
		case "sqlite3":
			query = "REPLACE INTO tabula (id, name, url, offset_x, offset_y, dpi, grid_r, grid_g, grid_b, grid_a) " +
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
		default:
			return fmt.Errorf("no Tabula.Save query for SQL dialect %s", dia)
		}
		_, err := db.Exec(query,
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

	if t.Tokens != nil {
		if err := t.saveTokens(db); err != nil {
			return err
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

func (t *Tabula) addGrid(i draw.Image) draw.Image {
	bounds := i.Bounds()
	gridded := i //copyImage(i)

	xOff := float32(t.OffsetX)
	for xOff > 0 {
		xOff -= t.Dpi
	}

	yOff := float32(t.OffsetY)
	for yOff > 0 {
		yOff -= t.Dpi
	}

	var col color.Color = t.GridColor
	if col == nil {
		col = &color.Black
	}

	for x := xOff; x < float32(bounds.Max.X); x += t.Dpi {
		if x < 0 {
			continue
		}
		for y := yOff; y < float32(bounds.Max.Y); y++ {
			if y < 0 {
				continue
			}
			mbDraw.BlendAt(gridded, int(x), int(y), col)
		}
	}

	// Horizontal lines; X at DPI intervals, all Y
	for x := xOff; x < float32(bounds.Max.X); x++ {
		if x < 0 {
			continue
		}
		for y := yOff; y < float32(bounds.Max.Y); y += t.Dpi {
			if y < 0 {
				continue
			}
			mbDraw.BlendAt(gridded, int(x), int(y), col)
		}
	}

	return gridded
}

var font *truetype.Font

func init() {
	path := "fonts/DejaVuSerif.ttf"
	fontData, err := ioutil.ReadFile(path)
	if err != nil {
		fontData, err = ioutil.ReadFile("../../" + path)
		if err != nil {
			panic(fmt.Sprintf("reading %s: %s", path, err))
		}
	}

	font, err = freetype.ParseFont(fontData)
	if err != nil {
		panic(fmt.Sprintf("parsing %s: %s", path, err))
	}
}

type dimension struct {
	w, h   float32
	valign VerticalAlignment
	halign HorizontalAlignment
}

var coordCache map[dimension]map[string]*image.RGBA

type VerticalAlignment int

const (
	Top VerticalAlignment = iota
	Middle
	Bottom
)

type HorizontalAlignment int

const (
	Left HorizontalAlignment = iota
	Center
	Right
)

// glyph renders the string given by `s` so that it fits horizontally in the rectangle given by (width,height)
func glyph(s string, width float32, height float32, valign VerticalAlignment, halign HorizontalAlignment) *image.RGBA {
	dim := dimension{width, height, valign, halign}
	if coordCache == nil {
		coordCache = map[dimension]map[string]*image.RGBA{}
	}

	if _, ok := coordCache[dim]; !ok {
		coordCache[dim] = map[string]*image.RGBA{}
	}

	if i, ok := coordCache[dim][s]; ok {
		return i
	}

	w := 100 + len(s)*int(height)
	h := 100 + int(height)

	img := image.NewRGBA(image.Rect(0, 0, w, h))

	ctx := freetype.NewContext()
	ctx.SetClip(img.Bounds())
	ctx.SetDst(img)
	ctx.SetFont(font)
	ctx.SetDPI(72.0)
	ctx.SetFontSize(float64(height))
	ctx.SetSrc(image.Black)

	for _, x := range []int{-2, 0, 2} {
		for _, y := range []int{-2, 0, 3} {
			ctx.DrawString(s, fixed.Point26_6{X: fixed.I(50 + x), Y: fixed.I(50 + int(height) + y)})
		}
	}

	ctx.SetSrc(image.White)
	ctx.DrawString(s, fixed.Point26_6{X: fixed.I(50), Y: fixed.I(50 + int(height))})

	img = autocrop(img)

	var resized image.Image

	// If the resized image would be too tall, resize to height; otherwise to width
	aspect_ratio := float32(img.Bounds().Dx()) / float32(img.Bounds().Dy())
	if width/aspect_ratio > height {
		resized = resize.Resize(0, uint(height), img, resize.Bilinear)
	} else {
		resized = resize.Resize(uint(width), 0, img, resize.Bilinear)
	}

	img = align(resized, int(width), int(height), halign, valign)
	coordCache[dim][s] = img
	return img
}

// align aligns the given image within a larger image, with bounds described
// by (width,height), according to the requested alignment. width and height
// describe the numbers of pixels, i.e., a value one greater than the right- or
// bottom-most pixel's index.
func align(i image.Image, width int, height int, halign HorizontalAlignment, valign VerticalAlignment) *image.RGBA {
	var offsetX, offsetY int
	switch halign {
	case Center:
		offsetX = width/2 - (i.Bounds().Dx())/2
	case Right:
		offsetX = width - i.Bounds().Dx()
	}
	switch valign {
	case Middle:
		offsetY = height/2 - (i.Bounds().Dy())/2
	case Bottom:
		offsetY = height - i.Bounds().Dy()
	}

	result := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
		for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
			result.Set(x+offsetX, y+offsetY, i.At(x, y))
		}
	}
	return result
}

func autocrop(i image.Image) *image.RGBA {
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

	return crop(i, min_x, min_y, max_x, max_y)
}

func crop(i image.Image, min_x, min_y, max_x, max_y int) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, max_x-min_x, max_y-min_y))
	for x := min_x; x < max_x; x++ {
		for y := min_y; y < max_y; y++ {
			result.Set(x-min_x, y-min_y, i.At(x, y))
		}
	}
	return result
}

func (t *Tabula) squareAtFloat(i draw.Image, minX, minY, maxX, maxY float32, inset int, col color.Color) {
	for x := int(minX*t.Dpi) + t.OffsetX + inset; x < int(maxX*t.Dpi)+t.OffsetX-inset; x++ {
		if x < 0 {
			continue
		}
		for y := int(minY*t.Dpi) + t.OffsetY + inset; y < int(maxY*t.Dpi)+t.OffsetY-inset; y++ {
			if y < 0 {
				continue
			}
			mbDraw.BlendAt(i, x, y, col)
		}
	}
}

func (t *Tabula) squareAt(i draw.Image, bounds image.Rectangle, inset int, col color.Color) {
	t.squareAtFloat(i, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Max.X), float32(bounds.Max.Y), inset, col)
}

// drawAt *modifies* the image given by `i` so that the string given by `what` is printed in the square at tabula
// coordinates x,y (not image coordinates), scaled so that the string occupies a rectangle described by (width,height) tabula squares
func (t *Tabula) printAt(i draw.Image, what string, x float32, y float32, width float32, height float32, valign VerticalAlignment, halign HorizontalAlignment) {
	g := glyph(what, t.Dpi*width, t.Dpi*height, valign, halign)
	draw.Draw(
		i,
		image.Rect(
			int(x*t.Dpi)+t.OffsetX, int(y*t.Dpi)+t.OffsetY,
			int((x+width)*t.Dpi)+t.OffsetX, int((y+height)*t.Dpi)+t.OffsetY,
		),
		g,
		image.Pt(0, 0),
		draw.Over,
	)
}

func (t *Tabula) addCoordinates(i draw.Image, first_x, first_y int) draw.Image {
	letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
		"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	result := i //copyImage(i)

	rows := int(float32(i.Bounds().Max.Y) / t.Dpi)
	cols := int(float32(i.Bounds().Max.X) / t.Dpi)
	// 0 1 2 3 4 ... 25 26 27 28
	// A B C D E ... Y  Z  AA AB
	for x := first_x; x < cols; x++ {
		c := letters[x%26] // 0..25
		tmp := x
		for tmp > 25 {
			tmp = int(tmp/26) - 1
			c = letters[tmp%26] + c
		}
		t.printAt(result, c, float32(x), float32(first_y), 1, 0.5, Middle, Left)
	}

	for y := first_y; y < rows; y++ {
		t.printAt(result, strconv.Itoa(y+1), float32(first_x), float32(y)+0.5, 1, 0.5, Middle, Right)
	}

	return result
}

type cacheEntry struct {
	version int
	image   image.Image
}

var cache = map[string]cacheEntry{}

func (t *Tabula) Render(ctx context.Context, sendStatusMessage func(string)) (image.Image, error) {
	if ctx == nil {
		return nil, fmt.Errorf("render of tabula %d received nil context", t.Id)
	}
	if t.Dpi == 0 {
		return nil, errors.New("cannot render tabula with zero DPI")
	}

	cacheKey := fmt.Sprintf("%s|%fdpi+%dx%d", t.Url, t.Dpi, t.OffsetX, t.OffsetY)

	var gridded image.Image
	if cached, ok := cache[cacheKey]; ok && cached.version == t.Version {
		gridded = copyImage(cached.image)
	} else {
		log.Infof("Cache miss: %s", cacheKey)
		if t.Background == nil {
			bg, ok := cache[t.Url]
			if ok && bg.version == t.Version {
				t.Background = copyImage(bg.image)
			} else {
				sendStatusMessage("I have to retrieve the background image; this could take a moment.")
				if err := t.Hydrate(); err != nil {
					return nil, fmt.Errorf("retrieving background: %s", err)
				}
				cache[t.Url] = cacheEntry{t.Version, copyImage(t.Background)}
			}
		}

		dx := t.Background.Rect.Dx()
		dy := t.Background.Rect.Dy()
		var resized image.Image = t.Background
		if dx > 2000 || dy > 2000 {
			if dx > dy {
				resized = resize.Resize(2000, 0, t.Background, resize.Bilinear)
			} else {
				resized = resize.Resize(0, 2000, t.Background, resize.Bilinear)
			}
		}
		if dx < 2000 && dy < 2000 {
			if dx > dy {
				resized = resize.Resize(2000, 0, t.Background, resize.Bilinear)
			} else {
				resized = resize.Resize(0, 2000, t.Background, resize.Bilinear)
			}
		}
		if drawable, ok := resized.(draw.Image); ok {
			gridded = t.addGrid(drawable)
			cache[cacheKey] = cacheEntry{t.Version, copyImage(gridded)}
		} else {
			panic("resize didn't return a drawable image?!")
		}
	}

	log.Debugf("adding marks...")
	if err := t.addMarks(gridded, ctx); err != nil {
		return nil, err
	}

	log.Debugf("adding lighting...")
	if err := t.addTokenLights(gridded, ctx); err != nil {
		return nil, err
	}

	log.Debugf("adding tokens...")
	if err := t.addTokens(gridded, ctx); err != nil {
		return nil, err
	}

	minx, miny, maxx, maxy := ctx.GetZoom()

	var coord image.Image
	if drawable, ok := gridded.(draw.Image); ok {
		coord = t.addCoordinates(drawable, minx, miny)
	} else {
		panic("resize didn't return a drawable image?!")
	}

	if maxx > minx && maxy > miny {
		coord = crop(
			coord,
			int(float64(minx)*float64(t.Dpi))+t.OffsetX,
			int(float64(miny)*float64(t.Dpi))+t.OffsetY,
			int(math.Ceil(float64(maxx+1)*float64(t.Dpi)))+t.OffsetX,
			int(math.Ceil(float64(maxy+1)*float64(t.Dpi)))+t.OffsetY,
		)
	}

	return coord, nil
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
