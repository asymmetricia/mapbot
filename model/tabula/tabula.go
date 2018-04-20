// Tabula models an individual map. This includes the URL form which the background image is retrieved, the background
// image itself; and information about how to overlay a grid. In the future, tokens, masks, and overlays will also be
// included.
package tabula

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/nfnt/resize"
	"github.com/pdbogen/mapbot/common/cache"
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
	"io"
	"io/ioutil"
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

	// A list of lines appended to lines obtained from the context during rendering.
	Lines []mark.Line
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

func (t *Tabula) Delete(db anydb.AnyDb) error {
	_, err := db.Exec("DELETE FROM tabulas WHERE id=$1", int64(*t.Id))
	if err != nil {
		return fmt.Errorf("deleting table %d: %s", *t.Id, err)
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
		Dpi: 50,
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

	imgBuf := &bytes.Buffer{}
	if _, err := io.Copy(imgBuf, res.Body); err != nil {
		return fmt.Errorf("error reading from HTTP response: %s", err)
	}

	imgData := imgBuf.Bytes()
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		n := 16
		if len(imgData) < 16 {
			n = len(imgData)
		}
		return fmt.Errorf("received image data (%q…), but: %s", imgData[0:n], err)
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

	log.Debugf("adding grid to image with bounds %v at spacing %.02f", i.Bounds(), t.Dpi)

	//xOff := float32(t.OffsetX)
	//for xOff > 0 {
	//	xOff -= t.Dpi
	//}
	//
	//yOff := float32(t.OffsetY)
	//for yOff > 0 {
	//	yOff -= t.Dpi
	//}

	xOff := float32(0)
	yOff := float32(0)

	blackOnWhite := false
	var col color.Color = t.GridColor
	if col == nil {
		col = &color.Black
	}

	r, g, b, _ := col.RGBA()
	if r == 0 && b == 0 && g == 0 {
		blackOnWhite = true
	}

	for x := xOff; x < float32(bounds.Max.X); x += t.Dpi {
		if blackOnWhite {
			mbDraw.Line(gridded, image.Pt(int(x-1), 0), image.Pt(int(x-1), bounds.Max.Y), col)
			mbDraw.Line(gridded, image.Pt(int(x+1), 0), image.Pt(int(x+1), bounds.Max.Y), col)
		}
		mbDraw.Line(gridded, image.Pt(int(x), 0), image.Pt(int(x), bounds.Max.Y), col)
	}

	// Horizontal lines; Y at DPI intervals, all X
	for y := yOff; y < float32(bounds.Max.Y); y += t.Dpi {
		if blackOnWhite {
			mbDraw.Line(gridded, image.Pt(0, int(y-1)), image.Pt(bounds.Max.X, int(y-1)), col)
			mbDraw.Line(gridded, image.Pt(0, int(y+1)), image.Pt(bounds.Max.X, int(y+1)), col)
		}
		mbDraw.Line(gridded, image.Pt(0, int(y)), image.Pt(bounds.Max.X, int(y)), col)
	}

	if blackOnWhite {
		for x := xOff; x < float32(bounds.Max.X); x += t.Dpi {
			mbDraw.Line(gridded, image.Pt(int(x), 0), image.Pt(int(x), bounds.Max.Y), color.White)
		}
		for y := yOff; y < float32(bounds.Max.Y); y += t.Dpi {
			mbDraw.Line(gridded, image.Pt(0, int(y)), image.Pt(bounds.Max.X, int(y)), color.White)
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

	return Crop(i, min_x, min_y, max_x, max_y)
}

// Crops the given image to the given pixel dimensions and returns a new image.RGBA.
func Crop(i image.Image, min_x, min_y, max_x, max_y int) *image.RGBA {
	offset_x := 0
	if min_x < 0 {
		offset_x = -min_x
		min_x = 0
	}

	offset_y := 0
	if min_y < 0 {
		offset_y = -min_y
		min_y = 0
	}

	result := image.NewRGBA(image.Rect(0, 0, max_x-min_x, max_y-min_y))
	for x := min_x; x < max_x; x++ {
		srcX := x - offset_x
		if srcX < 0 || srcX > i.Bounds().Max.X {
			continue
		}
		for y := min_y; y < max_y; y++ {
			srcY := y - offset_y
			if srcY < 0 || srcY > i.Bounds().Max.Y {
				continue
			}
			result.Set(x-min_x, y-min_y, i.At(srcX, srcY))
		}
	}
	return result
}

func (t *Tabula) line(i draw.Image, fromX, fromY, toX, toY float32, col color.Color, offset image.Point) {
	iFromX := int(fromX*t.Dpi) + offset.X
	iFromY := int(fromY*t.Dpi) + offset.Y
	iToX := int(toX*t.Dpi) + offset.X
	iToY := int(toY*t.Dpi) + offset.Y
	log.Debugf("drawing line from (%d,%d) to (%d,%d)", iFromX, iFromY, iToX, iToY)
	mbDraw.Line(i, image.Pt(iFromX, iFromY), image.Pt(iToX, iToY), col)
}

func (t *Tabula) squareAtFloat(i draw.Image, minX, minY, maxX, maxY float32, inset int, col color.Color, offset image.Point) {
	for x := int(minX*t.Dpi) + offset.X + inset; x < int(maxX*t.Dpi)+offset.X-inset; x++ {
		if x < 0 {
			continue
		}
		for y := int(minY*t.Dpi) + offset.Y + inset; y < int(maxY*t.Dpi)+offset.Y-inset; y++ {
			if y < 0 {
				continue
			}
			mbDraw.BlendAt(i, x, y, col)
		}
	}
}

func (t *Tabula) squareAt(i draw.Image, bounds image.Rectangle, inset int, col color.Color, offset image.Point) {
	t.squareAtFloat(i, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Max.X), float32(bounds.Max.Y), inset, col, offset)
}

// drawAt *modifies* the image given by `i` so that the string given by `what` is printed in the square at tabula
// coordinates x,y (not image coordinates), scaled so that the string occupies a rectangle described by (width,height) tabula squares
func (t *Tabula) printAt(i draw.Image, what string, x float32, y float32, width float32, height float32, valign VerticalAlignment, halign HorizontalAlignment, offset image.Point) {
	g := glyph(what, t.Dpi*width, t.Dpi*height, valign, halign)
	draw.Draw(
		i,
		image.Rect(
			int(x*t.Dpi)+offset.X, int(y*t.Dpi)+offset.Y,
			int((x+width)*t.Dpi)+offset.X, int((y+height)*t.Dpi)+offset.Y,
		),
		g,
		image.Pt(0, 0),
		draw.Over,
	)
}

func toLetter(p int) string {
	letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
		"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	if p < 0 {
		return "-" + toLetter(-p-1)
	}
	c := letters[p%26] // 0..25
	tmp := p
	for tmp > 25 {
		tmp = int(tmp/26) - 1
		c = letters[tmp%26] + c
	}
	return c
}

func (t *Tabula) addCoordinates(i draw.Image, first_x, first_y int, offset image.Point) draw.Image {
	result := i //copyImage(i)

	rows := int(float32(i.Bounds().Max.Y)/t.Dpi + 0.2)
	cols := int(float32(i.Bounds().Max.X)/t.Dpi + 0.2)
	// 0 1 2 3 4 ... 25 26 27 28
	// A B C D E ... Y  Z  AA AB
	for x := first_x; x < first_x+cols; x++ {
		t.printAt(result, toLetter(x), float32(x), float32(first_y), 1, 0.5, Middle, Left, offset)
	}

	for y := first_y; y < first_y+rows; y++ {
		if y < 0 {
			t.printAt(result, strconv.Itoa(y), float32(first_x), float32(y)+0.5, 1, 0.5, Middle, Right, offset)
		} else {
			t.printAt(result, strconv.Itoa(y+1), float32(first_x), float32(y)+0.5, 1, 0.5, Middle, Right, offset)
		}
	}

	return result
}

// Returns the tabula BackgroundImage, scaled so that its largest dimension is 2000px.
func (t *Tabula) BackgroundImage(sendStatusMessage func(string)) (image.Image, error) {
	if t.Background == nil {
		bg, ok := cache.Get(t.Url)
		if ok && bg.Version == t.Version {
			t.Background = copyImage(bg.Image)
		} else {
			if sendStatusMessage != nil {
				sendStatusMessage("I have to retrieve the background image; this could take a moment.")
			}
			if err := t.Hydrate(); err != nil {
				return nil, fmt.Errorf("retrieving background: %s", err)
			}
			cache.Put(t.Url, &cache.CacheEntry{t.Version, copyImage(t.Background)})
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
	return resized, nil
}

func (t *Tabula) Render(ctx context.Context, sendStatusMessage func(string)) (image.Image, error) {
	if sendStatusMessage == nil {
		sendStatusMessage = func(string) {}
	}

	if ctx == nil {
		return nil, fmt.Errorf("render of tabula %d received nil context", t.Id)
	}
	if t.Dpi == 0 {
		return nil, errors.New("cannot render tabula with zero DPI")
	}

	minx, miny, maxx, maxy := ctx.GetZoom()

	log.Debugf("map with bounds from (%d,%d) to (%d,%d)", minx, miny, maxx, maxy)

	imgOffset := image.Point{
		int(float32(minx)*t.Dpi) + t.OffsetX,
		int(float32(miny)*t.Dpi) + t.OffsetY,
	}

	log.Debugf("calculated image offset %v", imgOffset)

	tokenOffset := image.Point{
		int(float32(minx)*t.Dpi) * -1,
		int(float32(miny)*t.Dpi) * -1,
	}
	log.Debugf("token offset %v", tokenOffset)

	cacheKey := fmt.Sprintf("%s|%fdpi+%dx%d-%dx%d", t.Url, t.Dpi, minx, miny, maxx, maxy)

	var gridded image.Image
	if cached, ok := cache.Get(cacheKey); ok && cached.Version == t.Version {
		gridded = copyImage(cached.Image)
	} else {
		log.Infof("Cache miss: %s", cacheKey)
		resized, err := t.BackgroundImage(sendStatusMessage)
		if err != nil {
			return nil, fmt.Errorf("retrieving background: %s", err)
		}

		drawable, ok := resized.(draw.Image)
		if !ok {
			panic("resize didn't return a drawable image?!")
		}

		if minx == maxx {
			maxx = int(float32(drawable.Bounds().Dx()+t.OffsetX) / t.Dpi)
		}

		if miny == maxy {
			maxy = int(float32(drawable.Bounds().Dy()+t.OffsetY) / t.Dpi)
		}

		r := image.Rect(
			imgOffset.X,
			imgOffset.Y,
			int(float32(maxx+1)*t.Dpi)+t.OffsetX+1,
			int(float32(maxy+1)*t.Dpi)+t.OffsetY+1,
		)
		log.Debugf("padding out to %v", r)
		padded := mbDraw.Pad(drawable, r)
		gridded = t.addGrid(padded)
		cache.Put(cacheKey, &cache.CacheEntry{t.Version, copyImage(gridded)})
	}

	log.Debugf("adding marks...")
	if err := t.addMarks(gridded, ctx, tokenOffset); err != nil {
		return nil, err
	}

	log.Debugf("adding lighting...")
	if err := t.addTokenLights(gridded, ctx, tokenOffset); err != nil {
		return nil, err
	}

	log.Debugf("adding tokens...")
	if err := t.addTokens(gridded, ctx, tokenOffset); err != nil {
		return nil, err
	}

	log.Debugf("adding lines...")
	if err := t.addLines(gridded, ctx, tokenOffset); err != nil {
		return nil, err
	}

	var coord image.Image
	if drawable, ok := gridded.(draw.Image); ok {
		coord = t.addCoordinates(drawable, minx, miny, tokenOffset)
	} else {
		panic("resize didn't return a drawable image?!")
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

func (t Tabula) PointToPixel(pt image.Point, dir string) image.Point {
	sX := 0
	sY := 0
	switch dir {
	case "nw":
	case "n":
		sX = int(t.Dpi/2 + 0.5)
	case "ne":
		sX = int(t.Dpi + 0.5)
	case "w":
		sY = int(t.Dpi/2 + 0.5)
	case "": // middle
		sX = int(t.Dpi/2 + 0.5)
		sY = int(t.Dpi/2 + 0.5)
	case "e":
		sY = int(t.Dpi/2 + 0.5)
		sX = int(t.Dpi + 0.5)
	case "sw":
		sY = int(t.Dpi + 0.5)
	case "s":
		sY = int(t.Dpi + 0.5)
		sX = int(t.Dpi/2 + 0.5)
	case "se":
		sY = int(t.Dpi + 0.5)
		sX = int(t.Dpi + 0.5)
	}
	return image.Pt(
		int(float32(pt.X)*t.Dpi+0.5)+sX+t.OffsetX,
		int(float32(pt.Y)*t.Dpi+0.5)+sY+t.OffsetY,
	)
}
