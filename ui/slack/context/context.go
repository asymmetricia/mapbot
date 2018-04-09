package context

import (
	"encoding/json"
	"errors"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	contextModel "github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/context/databaseContext"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var log = mbLog.Log

type SlackContext struct {
	databaseContext.DatabaseContext
	Emoji      map[string]string
	EmojiCache map[string]image.Image
}

func (sc *SlackContext) IsEmoji(name string) bool {
	return name[0] == ':' && name[len(name)-1] == ':'
}

var staticAliases = map[string]string{
	"boat":              "sailboat",
	"large_blue_circle": "blue_circle",
}

type emojiCodePoints struct {
	Base              string `json:"base"`
	NonFullyQualified string `json:"non_fully_qualified"`
}

type emoji struct {
	CodePoints emojiCodePoints `json:"code_points"`
	Shortname  string          `json:"shortname"`
	Image      image.Image     `json:"-"`
	Aliases    []string        `json:"shortname_alternates"`
}

var EmojiOne map[string]*emoji = map[string]*emoji{}

func init() {
	start := time.Now()
	if err := json.Unmarshal([]byte(emojiJson), &EmojiOne); err != nil {
		panic(fmt.Sprintf("parsing emoji one JSON: %s", err))
	}

	expanded := map[string]bool{}
	for n, e := range EmojiOne {
		if _, ok := expanded[n]; ok {
			continue
		}
		EmojiOne[strings.Trim(e.Shortname, ":")] = e
		expanded[e.Shortname] = true
		for _, a := range e.Aliases {
			EmojiOne[strings.Trim(a, ":")] = e
			expanded[a] = true
		}
		expanded[n] = true
	}

	for from, to := range staticAliases {
		e, ok := EmojiOne[to]
		if !ok {
			log.Warningf("missing target for static alias %q -> %q", from, to)
			continue
		}
		EmojiOne[from] = e
	}

	log.Debugf("Parsing %d emoji from emojiOne payload took %0.2fs", len(EmojiOne), time.Now().Sub(start).Seconds())
}

type EmojiNotFound error

func getEmojiOneFile(codepoint string) (image.Image, error) {
	path := fmt.Sprintf("/emoji/%s.png", codepoint)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func getEmojiOneCdn(codepoint string) (image.Image, error) {
	emojiUrl := fmt.Sprintf(emojiUrl, codepoint)
	c := http.Client{
		Timeout: 30 * time.Second,
	}

	log.Debugf("trying to retrieve glyph %s from %s...", codepoint, emojiUrl)
	res, err := c.Get(emojiUrl)
	if err != nil {
		return nil, fmt.Errorf("retrieving EmojiOne glyph %s: %s", codepoint, err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		log.Warningf("%d retrieving %s", res.StatusCode, emojiUrl)
		return nil, nil
	}

	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing EmojiOne glyph %s: %s", codepoint, err)
	}

	return img, nil
}

func GetEmojiOne(name string) (image.Image, error) {
	e, ok := EmojiOne[name]
	if !ok {
		name = strings.Replace(name, "_", "-", -1)
		e, ok = EmojiOne[name]
	}
	if !ok {
		name = strings.Replace(name, "-", "_", -1)
		e, ok = EmojiOne[name]
	}

	if !ok {
		return nil, EmojiNotFound(errors.New("not found"))
	}

	if e.Image != nil {
		return e.Image, nil
	}

	for _, pt := range []string{e.CodePoints.NonFullyQualified, e.CodePoints.Base} {
		img, err := getEmojiOneFile(pt)
		if err != nil {
			log.Warningf("error checking file cache for emoji %s: %s", pt, err)
			return nil, err
		}
		if img != nil {
			EmojiOne[name].Image = img
			return img, nil
		}

		img, err = getEmojiOneCdn(pt)
		if err != nil {
			log.Warningf("error checking CDN for emoji %s: %s", pt, err)
			return nil, err
		}
		if img != nil {
			EmojiOne[name].Image = img
			return img, nil
		}
	}

	return nil, EmojiNotFound(errors.New("could not retrieve"))
}

func (sc *SlackContext) GetEmoji(baseName string) (image.Image, error) {
	if baseName[0] != ':' || baseName[len(baseName)-1] != ':' {
		return nil, fmt.Errorf("slack emoji are bounded with colons (:), not %q and %q", baseName[0], baseName[len(baseName)-1])
	}
	name := baseName[1 : len(baseName)-1]
	if i, ok := sc.EmojiCache[name]; ok {
		return i, nil
	}
	log.Debugf("emoji cache miss: %s", name)
	if e, ok := sc.Emoji[name]; ok {
		emojiUrl, err := url.Parse(e)
		if err != nil {
			return nil, err
		}

		if emojiUrl.Scheme == "alias" {
			log.Debugf("emoji %q is alias for %q, let's try that...", baseName, emojiUrl.Opaque)
			return sc.GetEmoji(":" + emojiUrl.Opaque + ":")
		}

		c := http.Client{
			Timeout: 30 * time.Second,
		}

		res, err := c.Get(e)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		switch hdr := res.Header.Get("content-type"); hdr {
		case "image/jpeg":
			fallthrough
		case "image/gif":
			fallthrough
		case "image/png":
			img, _, err := image.Decode(res.Body)
			if err != nil {
				return nil, err
			}
			sc.EmojiCache[name] = img
			return img, nil
		default:
			return nil, fmt.Errorf("emoji was type %q, which is not currently supported", hdr)
		}

	}
	// OK, this might not be a custom emoji.
	img, err := GetEmojiOne(name)
	if _, ok := err.(EmojiNotFound); err != nil && !ok {
		return nil, fmt.Errorf("error getting EmojiOne emoji: %s", err)
	}
	if img != nil {
		return img, nil
	}

	return nil, fmt.Errorf("no emoji named %q in this context", name)
}

var _ contextModel.Context = (*SlackContext)(nil)
