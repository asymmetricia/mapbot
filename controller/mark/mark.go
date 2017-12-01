package mark

import (
	//	"fmt"
	// "github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/hub"
	//"github.com/pdbogen/mapbot/model/tabula"
	//"reflect"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:mark", cmdMark)
}

func cmdMark(h *hub.Hub, c *hub.Command) {
}
