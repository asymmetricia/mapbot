// A workflow is a series of states with a context, initially a user and map.
package workflow

import (
	"fmt"
	. "github.com/pdbogen/mapbot/common/log"
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

// State is called when we first enter a state (with nil `choice`) and when
// activity occurs on the state (non-nil `choice`).
//
// State looks up the state by name, and calls the appropriate callbacks, or
// else returns a non-nil WorkflowMessage will provide a message for the user
// describing the problem (but this typically indicates misuse of a debugging
// utility or a programming error).
//
// A nil WorkflowMessage is not an error, but indicates that there is no
// response for the user.
//
// newState and newOpaque should be updated in the user's entry for this
// workflow if they are non-nil.

func (wf *Workflow) State(workflowName string, state string, opaque interface{}, choice *string) (
	newState *string, newOpaque interface{}, msg *WorkflowMessage,
) {
	stateObj, ok := wf.States[strings.ToLower(state)]
	if !ok {
		return nil, nil, &WorkflowMessage{
			Workflow: workflowName,
			Text:     fmt.Sprintf("invalid state %q", state),
		}
	}

	if choice == nil {
		// we're handling a state entry
		if stateObj.Challenge == nil && stateObj.OnStateEnter == nil {
			return nil, nil, &WorkflowMessage{
				Workflow: workflowName,
				Text:     fmt.Sprintf("no on-enter associated with state %q", state),
			}
		}

		if stateObj.Challenge != nil {
			msg := stateObj.Challenge(opaque)
			if msg.State == "" {
				return nil, nil, msg
			}
			return &msg.State, nil, msg
		}

		return stateObj.OnStateEnter(opaque)
	}

	// we're handling state activity
	if stateObj.Response == nil && stateObj.OnUserAction == nil {
		return nil, nil, &WorkflowMessage{
			Workflow: workflowName,
			Text:     fmt.Sprintf("no on-action associated with state %q", state),
		}
	}

	if stateObj.Response != nil {
		newState, newOpaque := stateObj.Response(opaque, choice)
		return &newState, newOpaque, nil
	}

	return stateObj.OnUserAction(opaque, choice)
}

type WorkflowState struct {
	// Deprecated; use OnUserAction
	Challenge ChallengeFunc
	// Deprecated; use OnStateEnter
	Response ResponseFunc

	OnUserAction OnUserActionFunc
	OnStateEnter OnStateEnterFunc
}

// StateEnter idempotently retrieves the challenge for the named state with the given opaque data
type ChallengeFunc func(opaque interface{}) *WorkflowMessage

// OnStateEnter fires when a state is entered. All returns are optional.
//
// If `state` is non-nil, the given state will be entered.
//
// If `opaqueOut` is non-nil, the user's opaque data for this workflow will be
// replaced with the new value.
//
// If `message` is non-nil, the message will be sent to the user.
type OnStateEnterFunc func(opaqueIn interface{}) (state *string, opaqueOut interface{}, message *WorkflowMessage)

// OnUserAction fires when the workflow is in this state and the user takes some
// action on a previous workflow message. All returns are optional.
//
// If `state` is non-nil, the given state will be entered.
//
// If `opaqueOut` is non-nil, the user's opaque data for this workflow will be
// replaced with the new value.
//
// If `message` is non-nil, the message will be sent to the user.
type OnUserActionFunc func(opaqueIn interface{}, choice *string) (state *string, opaqueOut interface{}, message *WorkflowMessage)

// UserAction is idempotent from the state machine perspective but may have side
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

func String(s string) *string {
	ret := new(string)
	*ret = s
	return ret
}
