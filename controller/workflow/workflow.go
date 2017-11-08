package workflow

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/controller/cmdproc"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/user"
	"github.com/pdbogen/mapbot/model/workflow"
	"reflect"
	"strings"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:workflow", processor.Route)
}

var processor *cmdproc.CommandProcessor

func init() {
	processor = &cmdproc.CommandProcessor{
		Command: "workflow",
		Commands: map[string]cmdproc.Subcommand{
			"start": cmdproc.Subcommand{"<workflow name> <choice>", "primarily for debugging; manually initiates the named workflow with the given <choice> as the `enter` choice", cmdStart},
			"clear": cmdproc.Subcommand{"", "cancels any workflow associated with your user", cmdClear},
		},
	}
}

func cmdStart(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 1 {
		h.Error(c, "usage: workflow start <workflow name> <opaque JSON>")
		return
	}

	wf, ok := workflow.Workflows[strings.ToLower(args[0])]
	if !ok {
		h.Error(c, fmt.Sprintf("workflow %q could not be found", args[0]))
		return
	}

	choice := strings.Join(args[1:], " ")
	state, opaque, err := wf.Response("enter", nil, &choice)
	if err != nil {
		h.Error(c, fmt.Sprintf("error initiating workflow %q: %s", args[0], err))
		return
	}
	c.User.Workflows[strings.ToLower(args[0])] = user.WorkflowState{state, opaque}
	c.User.Save(db.Instance)

	msg := wf.Challenge(state, opaque)
	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(msg))
}

func cmdClear(h *hub.Hub, c *hub.Command) {}
