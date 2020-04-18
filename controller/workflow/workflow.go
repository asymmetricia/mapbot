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

	c.User.Workflows[strings.ToLower(args[0])] = user.WorkflowState{State: "enter"}
	if err := c.User.Save(db.Instance); err != nil {
		h.Error(c, fmt.Sprintf("error saving user opaque data: %s", err))
		return
	}

	cmdTransition(h, c.WithPayload(append([]string{args[0], "enter"}, args[1:]...)))
}

// move to the named state, trigger any OnStateEnter, & apply its results
func cmdTransition(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) != 2 {
		h.Error(c, "usage: workflow transition <workflow name> <state name>")
		return
	}

	wfName := strings.ToLower(args[0])
	wf, ok := workflow.Workflows[wfName]
	if !ok {
		h.Error(c, fmt.Sprintf("workflow %q could not be found", args[0]))
		return
	}

	wfStateName := strings.ToLower(args[1])

	userState, ok := c.User.Workflows[wfName]
	if !ok {
		userState = user.WorkflowState{}
	}

	transits := 1
	for {
		userState.State = strings.ToLower(wfStateName)
		newState, newOpaque, msg := wf.StateEnter(wfName, wfStateName, userState.Opaque)
		if newOpaque != nil {
			userState.Opaque = newOpaque
		}

		c.User.Workflows[wfName] = userState

		if msg != nil {
			h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(msg))
		}

		if newState == nil {
			break
		}
		transits++
		if transits > 5 {
			h.Error(c, "exceeded limit of 5 automatic transits")
			log.Errorf("workflow %q transit to %q exceeded limit of 5 automatic transits", args[0], args[1])
			return
		}
		wfStateName = strings.ToLower(*newState)
	}

	if err := c.User.Save(db.Instance); err != nil {
		h.Error(c, fmt.Sprintf("error saving updated state: %s", err))
		return
	}
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

	wfName := strings.ToLower(args[0])
	wf, ok := workflow.Workflows[wfName]
	if !ok {
		h.Error(c, fmt.Sprintf("error: workflow %q does not exist", args[0]))
		return
	}

	wfStateName := strings.ToLower(args[0])
	wfState, ok := c.User.Workflows[wfStateName]
	if !ok {
		h.Error(c, fmt.Sprintf(
			"error: you don't have an active workflow %q; try `workflow start %s %s`?",
			args[0], args[1], strings.Join(args[1:], " "),
		))
		return
	}

	choice := strings.Join(args[1:], " ")
	newState, opaque, err := wf.Response(wfState, &choice)
	if err != nil {
		h.Error(c, fmt.Sprintf("action not accepted: %s", err))
		return
	}

	if newState == "exit" {
		delete(c.User.Workflows, strings.ToLower(args[0]))
	} else {
		c.User.Workflows[wfStateName] = user.WorkflowState{State: wfStateName, Opaque: opaque}
	}

	if err := c.User.Save(db.Instance); err != nil {
		h.Error(c, fmt.Sprintf("error saving opaque data: %s", err))
		return
	}

	if newState != "exit" {
		cmdTransition(h, c.WithPayload([]string{wfName, newState}))
	}
}

func cmdClear(h *hub.Hub, c *hub.Command) {}
