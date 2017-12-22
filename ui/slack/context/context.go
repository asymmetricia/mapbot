package context

import (
	"encoding/json"
	"errors"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	contextModel "github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/context/databaseContext"
	"image"
	_ "image/png"
	"net/http"
	"net/url"
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
	"boat": "sailboat",
}

type emoji struct {
	Unicode string      `json:"unicode"`
	Image   image.Image `json:"-"`
	Aliases []string    `json:"aliases"`
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
		for _, a := range e.Aliases {
			EmojiOne[strings.Trim(a, ":")] = e
			expanded[a] = true
		}
		expanded[n] = true
	}

	for from, to := range staticAliases {
		if e, ok := EmojiOne[to]; ok {
			EmojiOne[from] = e
		}
	}

	log.Debugf("Parsing emojiOne payload took %0.2fs", time.Now().Sub(start).Seconds())
}

type EmojiNotFound error

func GetEmojiOne(name string) (image.Image, error) {
	e, ok := EmojiOne[name]
	if !ok {
		e, ok = EmojiOne[strings.Replace(name, "_", "-", -1)]
	}
	if !ok {
		e, ok = EmojiOne[strings.Replace(name, "-", "_", -1)]
	}

	if ok {
		if e.Image == nil {
			emojiUrl := fmt.Sprintf("https://cdnjs.cloudflare.com/ajax/libs/emojione/2.2.7/assets/png/%s.png", e.Unicode)
			c := http.Client{
				Timeout: 30 * time.Second,
			}

			res, err := c.Get(emojiUrl)
			if err != nil {
				return nil, fmt.Errorf("retrieving EmojiOne glyph %s (%s): %s", name, e.Unicode, err)
			}
			defer res.Body.Close()

			img, _, err := image.Decode(res.Body)
			if err != nil {
				return nil, fmt.Errorf("parsing EmojiOne glyph %s (%s): %s", name, e.Unicode, err)
			}
			e.Image = img

			return img, nil
		} else {
			return e.Image, nil
		}
	} else {
		return nil, EmojiNotFound(errors.New("not found"))
	}
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
