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
			"list":    cmdproc.Subcommand{"", "lists known workflows", cmdList},
			"start":   cmdproc.Subcommand{"<workflow name> <choice>", "internal/debugging; manually initiates the named workflow with the given <choice> as the `enter` choice", cmdStart},
			"respond": cmdproc.Subcommand{"<workflow name> <choice>", "internal/debugging; calls the named workflow's current state's Resopnd function with the given choice.", cmdRespond},
			"clear":   cmdproc.Subcommand{"", "cancels any workflow associated with your user", cmdClear},
		},
	}
}

func cmdList(h *hub.Hub, c *hub.Command) {
	response := []string{
		"Known workflows:",
	}

	for wf := range workflow.Workflows {
		response = append(response, fmt.Sprintf("- %s", wf))
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(strings.Join(response, "\n")))
}

func cmdStart(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 1 {
		h.Error(c, "usage: workflow start <workflow name> <choice>")
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
	if state == "exit" {
		delete(c.User.Workflows, strings.ToLower(args[0]))
	} else {
		c.User.Workflows[strings.ToLower(args[0])] = user.WorkflowState{state, opaque}
	}
	c.User.Save(db.Instance)

	msg := wf.Challenge(args[0], state, opaque)
	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(msg))
}

func cmdRespond(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 1 {
		h.Error(c, "usage: workflow respond <workflow name> <choice>")
		return
	}

	wf, ok := workflow.Workflows[strings.ToLower(args[0])]
	if !ok {
		h.Error(c, fmt.Sprintf("error: workflow %q does not exist", args[0]))
		return
	}

	wfState, ok := c.User.Workflows[strings.ToLower(args[0])]
	if !ok {
		h.Error(c, fmt.Sprintf(
			"error: you don't have an active workflow %q; try `workflow start %s %s`?",
			args[0], args[1], strings.Join(args[1:], " "),
		))
		return
	}

	choice := strings.Join(args[1:], " ")
	state, opaque, err := wf.Response(wfState.State, wfState.Opaque, &choice)
	if err != nil {
		h.Error(c, fmt.Sprintf("action not accepted: %s", err))
		return
	}
	if state == "exit" {
		delete(c.User.Workflows, strings.ToLower(args[0]))
	} else {
		c.User.Workflows[strings.ToLower(args[0])] = user.WorkflowState{state, opaque}
	}
	msg := wf.Challenge(args[0], state, opaque)
	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(msg))
	c.User.Save(db.Instance)
}

func cmdClear(h *hub.Hub, c *hub.Command) {}
