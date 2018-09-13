package web

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/webSession"
)

var log = mbLog.Log

func Register(h *hub.Hub, tls bool, domain string) {
	h.Subscribe("user:web", GenerateWebSession(tls, domain))
}

func GenerateWebSession(tls bool, domain string) func(h *hub.Hub, c *hub.Command) {
	proto := "http"
	if tls {
		proto = "https"
	}
	return func(h *hub.Hub, c *hub.Command) {
		s, err := webSession.NewWebSession(db.Instance, c.Context.Id(), c.Context.Type())
		if err != nil {
			h.Error(c, "sorry, unable to generate a web session for you")
			log.Errorf("unable to generate web session: %v", err)
			return
		}

		h.Reply(c, fmt.Sprintf("Looks good! <%s://%s/ui?id=%s|Click here.>", proto, domain, s.SessionId))
	}
}
