// A workflow is a series of states with a context, initially a user and map.
package workflow

import (
	"fmt"
	. "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/user"
	"github.com/pdbogen/mapbot/ui/slack/context"
	"image"
	"strings"
)

/* workflow is a state machine
each state is a named challenge/response
a default `enter` state starts things off
any transition to an `exit` state ends things
a transition includes an optional message to a user
state should be persisted to the database and lazy-loaded into a cache
*/

type Workflow struct {
	States         map[string]WorkflowState
	OpaqueFromJson func([]byte) (interface{}, error)
}

// Challenge is called upon entering a state; the state issues a challenge to
// the user. Challenge looks up the state by name, and if a state is found and
// it has a challenge, returns the result of calling that state's Challenge.
// If the state cannot be found or it does not have a defined Challenge, a non-nil WorkflowMessage will provide a
// message for the user describing the problem (but this typically indicates misuse of a debugging utility or a
// programming error).
// A nil WorkflowMessage is not an error, but indicates that there is no response for the user.

func (wf *Workflow) Challenge(key string, state string, opaque interface{}) *WorkflowMessage {
	stateObj, ok := wf.States[strings.ToLower(state)]
	if !ok {
		return &WorkflowMessage{Workflow: key, Text: fmt.Sprintf("invalid state %q", state)}
	}
	if stateObj.Challenge == nil {
		return &WorkflowMessage{Workflow: key, Text: fmt.Sprintf("no challenge associated with state %q", state)}
	}
	msg := stateObj.Challenge(opaque)
	if msg == nil {
		return &WorkflowMessage{Workflow: key}
	}

	msg.Workflow = key
	return msg
}

// Response is called when a user responds to a challenge, either via a message
// action or via debugging commands. Response looks up the state by name, and
// if a state is found and it has a Response, returns the result of calling
// that state's Response.
// Error will be non-nil if the state cannot be found or it does not have a Response.
func (wf *Workflow) Response(state user.WorkflowState, choice *string) (string, interface{}, error) {
	stateObj, ok := wf.States[strings.ToLower(state.State)]
	if !ok {
		return "", nil, fmt.Errorf("state %q not found", state)
	}
	if stateObj.Response == nil {
		return "", nil, fmt.Errorf("state %q does not have a response", state)
	}

	if wf.OpaqueFromJson != nil && state.Opaque == nil && len(state.OpaqueRaw) > 0 {
		var err error
		state.Opaque, err = wf.OpaqueFromJson(state.OpaqueRaw)
		if err != nil {
			return "", nil, fmt.Errorf("could not unmarshal opaque data: %s", err)
		}
	}

	newState, newOpaque := stateObj.Response(state.Opaque, choice)
	return newState, newOpaque, nil
}

type WorkflowState struct {
	Challenge ChallengeFunc
	Response  ResponseFunc
}

// Challenge idempotently retrieves the challenge for the named state with the given opaque data
type ChallengeFunc func(opaque interface{}) *WorkflowMessage

// Response is idempotent from the state machine perspective but may have side
// effects like modifying Tabula or tokens. It executes the action associated
// with a state for the given choice. Usually the 'choice' is one of the
// options provided in the corresponding state's challenge's WorkflowMessage,
// but especially for `enter` states can be potentially any string.
type ResponseFunc func(opaque interface{}, choice *string) (newState string, newOpaque interface{})

type WorkflowMessage struct {
	Workflow   string
	State      string
	Text       string
	Choices    []string
	ChoiceSets [][]string
	Image      image.Image
}

func (wfm *WorkflowMessage) Id() string {
	return wfm.Workflow
}

var Workflows = map[string]Workflow{
	"align": alignWorkflow,
	"demo": Workflow{
		States: map[string]WorkflowState{
			"enter": {
				Response: func(interface{}, *string) (string, interface{}) {
					return "greet", nil
				},
			},
			"greet": {
				Challenge: func(interface{}) *WorkflowMessage {
					grin, err := context.GetEmojiOne("grin")
					if err != nil {
						Log.Errorf("could not get grin emoji? wtf? %s", err)
					}
					return &WorkflowMessage{
						Text:    "This is a demo workflow. Click the button to exit.",
						Choices: []string{"exit"},
						Image:   grin,
					}
				},
				Response: func(interface{}, *string) (string, interface{}) {
					return "exit", nil
				},
			},
			"exit": {
				Challenge: func(interface{}) *WorkflowMessage {
					return &WorkflowMessage{
						Text: "All done.",
					}
				},
			},
		},
	},
}
