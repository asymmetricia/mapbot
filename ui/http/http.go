package http

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db/anydb"
	"github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/webSession"
	"image/png"
	"net/http"
)

type Http struct {
	db   anydb.AnyDb
	hub  *hub.Hub
	mux  *http.ServeMux
	prov *context.ContextProvider
}

func (h *Http) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.mux.ServeHTTP(rw, req)
}

func New(db anydb.AnyDb, hub *hub.Hub, prov *context.ContextProvider) *Http {
	ret := &Http{db: db, hub: hub, prov: prov}
	mux := http.NewServeMux()
	mux.HandleFunc("/", ret.GetMap)
	ret.mux = mux
	return ret
}

func (h *Http) GetSession(rw http.ResponseWriter, req *http.Request) (*webSession.WebSession, bool) {
	id := req.FormValue("id")
	if id == "" {
		http.NotFound(rw, req)
		return nil, false
	}
	ret, err := webSession.Load(h.db, id)
	if err != nil {
		switch err.(type) {
		case webSession.NotFound:
			http.NotFound(rw, req)
			return nil, false
		default:
			http.Error(rw, "internal server error", http.StatusInternalServerError)
			log.Errorf("non-notfound error loading web session with id %q: %v", id, err)
			return nil, false
		}
	}
	return ret, true
}

func (h *Http) GetMap(rw http.ResponseWriter, req *http.Request) {
	sess, ok := h.GetSession(rw, req)
	if !ok {
		return
	}
	ctx, err := sess.GetContext(h.prov)
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		log.Errorf("retrieving context for session %v: %v", sess, err)
		return
	}

	tabId := ctx.GetActiveTabulaId()
	if tabId == nil {
		fmt.Fprintln(rw, "No active map.")
		return
	}

	tab, err := tabula.Load(h.db, *tabId)
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		log.Errorf("loading tabula with id %q: %v", *tabId, err)
		return
	}

	img, err := tab.Render(ctx, nil)
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		log.Errorf("loading tabula bg with id %q: %v", *tabId, err)
		return
	}
	png.Encode(rw, img)
}
