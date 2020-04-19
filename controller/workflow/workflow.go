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
			"list":       cmdproc.Subcommand{"", "lists known workflows", cmdList},
			"transition": cmdproc.Subcommand{"<workflow name> <state>", "internal/debugging; enter given workflow state", cmdTransition},
			"action":     cmdproc.Subcommand{"<workflow name> <choice>", "internal/debugging; simulate given user action", cmdAction},
			"start":      cmdproc.Subcommand{"<workflow name> <choice>", "internal/debugging; start the given workflow with empty opaque data", cmdStart},
			"clear":      cmdproc.Subcommand{"", "cancels any workflow associated with your user", cmdClear},
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
	userState.Hydrate(wf.OpaqueFromJson)

	transits := 1
	for {
		userState.State = strings.ToLower(wfStateName)
		newState, newOpaque, msg := wf.State(wfName, wfStateName, userState.Opaque, nil)
		if newOpaque != nil {
			userState.Opaque = newOpaque
		}

		c.User.Workflows[wfName] = userState

		if msg != nil {
			msg.Workflow = wfName
			h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(msg))
		}

		if newState == nil || wfStateName == strings.ToLower(*newState) {
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

func cmdStart(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 2 {
		h.Error(c, "usage: workflow start <workflow name> <choice>")
		return
	}

	wfName := strings.ToLower(args[0])

	c.User.Workflows[wfName] = user.WorkflowState{State: "enter"}
	cmdAction(h, c)
}

func cmdAction(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 2 {
		h.Error(c, "usage: workflow action <workflow name> <choice>")
		return
	}
	wfName := strings.ToLower(args[0])
	choice := strings.Join(args[1:], " ")

	wf, ok := workflow.Workflows[wfName]
	if !ok {
		h.Error(c, fmt.Sprintf("error: workflow %q does not exist", args[0]))
		return
	}

	userWorkflowState, ok := c.User.Workflows[wfName]
	if !ok {
		userWorkflowState.State = "enter"
	}
	userWorkflowState.Hydrate(wf.OpaqueFromJson)

	newState, opaque, msg := wf.State(wfName, userWorkflowState.State, userWorkflowState.Opaque, &choice)

	if opaque != nil {
		userWorkflowState.Opaque = opaque
	}

	if newState != nil {
		userWorkflowState.State = *newState
	}

	if userWorkflowState.State == "exit" {
		delete(c.User.Workflows, userWorkflowState.State)
	} else {
		c.User.Workflows[wfName] = userWorkflowState
	}

	if err := c.User.Save(db.Instance); err != nil {
		h.Error(c, fmt.Sprintf("error saving opaque data: %s", err))
		return
	}

	if msg != nil {
		h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(msg))
	}

	if newState != nil && *newState != "exit" {
		log.Debugf("transitioning %q's %q to %q", c.User.Id, wfName, *newState)
		cmdTransition(h, c.WithPayload([]string{wfName, *newState}))
	}
}

func cmdClear(h *hub.Hub, c *hub.Command) {
	c.User.Workflows = map[string]user.WorkflowState{}
	if err := c.User.Save(db.Instance); err != nil {
		h.Error(c, fmt.Sprintf("error saving opaque data: %s", err))
		return
	}
	h.Publish(&hub.Command{
		Type:    hub.CommandType(c.From),
		Payload: "done!",
		Context: c.Context,
	})
}
