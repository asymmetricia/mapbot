package blobserv

import (
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/common/rand"
	"time"
	"net/http"
	"strings"
	"fmt"
	"errors"
	"sync"
)

var log = mbLog.Log

type Blob struct {
	Data   []byte
	Expiry time.Time
}

// UrlBase should be `/`-terminated.
type BlobServ struct {
	UrlBase     string
	Blobs       map[string]*Blob
	BlobsMu     sync.Mutex
	janitorOnce sync.Once
}

var Instance *BlobServ

func Upload(data []byte) (url string, err error) {
	if Instance == nil {
		return "", errors.New("Upload called with nil Instance")
	}

	Instance.janitorOnce.Do(Instance.janitor)

	key := rand.RandHex(32)

	Instance.BlobsMu.Lock()
	defer Instance.BlobsMu.Unlock()
	if Instance.Blobs == nil {
		Instance.Blobs = map[string]*Blob{}
	}
	Instance.Blobs[key] = &Blob{
		Data:   make([]byte, len(data)),
		Expiry: time.Now().Add(time.Hour),
	}

	copy(Instance.Blobs[key].Data, data)

	return Instance.UrlBase + key, nil
}

func (b *BlobServ) janitor() {
	go func() {
		for {
			b.clean()
			time.Sleep(time.Minute)
		}
	}()
}

func (b *BlobServ) clean() {
	b.BlobsMu.Lock()
	defer b.BlobsMu.Unlock()

	for key, blob := range b.Blobs {
		if blob.Expiry.Before(time.Now()) {
			delete(b.Blobs, key)
		}
	}
}

func clog(req *http.Request, status int) {
	log.Infof("%s - - [%s] \"%s %s %s\" %d -\n",
		req.RemoteAddr,
		time.Now().Format("02/Jan/2006:15:04:05 -0700"),
		req.Method, req.RequestURI, req.Proto,
		status,
	)
}

func (b *BlobServ) Serve(rw http.ResponseWriter, req *http.Request) {
	if b == nil {
		panic("Serve called on nil BlobServ")
	}

	parts := strings.Split(req.URL.Path, "/")
	if len(parts) == 0 {
		clog(req, http.StatusBadRequest)
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, "Somehow, invalid response to blob server")
		return
	}
	key := parts[len(parts)-1]

	b.BlobsMu.Lock()
	blob, ok := b.Blobs[key]
	b.BlobsMu.Unlock()

	if !ok {
		clog(req, http.StatusNotFound)
		http.NotFound(rw, req)
		return
	}

	clog(req, http.StatusOK)
	rw.WriteHeader(http.StatusOK)
	n, err := rw.Write(blob.Data)
	if err != nil {
		log.Errorf("writing blob %q: %s", key, err)
		return
	}
	if n != len(blob.Data) {
		log.Warningf("no error, but incomplete write of blob %q (wrote %d, expected %d)", key, n, len(blob.Data))
	}
}
