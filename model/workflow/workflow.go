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
	States map[string]WorkflowState
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
	msg.Workflow = key
	return msg
}

// Response is called when a user responds to a challenge, either via a message
// action or via debugging commands. Response looks up the state by name, and
// if a state is found and it has a Response, returns the result of calling
// that state's Response.
// Error will be non-nil if the state cannot be found or it does not have a Response.
func (wf *Workflow) Response(state string, opaque interface{}, choice *string) (string, interface{}, error) {
	stateObj, ok := wf.States[strings.ToLower(state)]
	if !ok {
		return "", nil, fmt.Errorf("state %q not found", state)
	}
	if stateObj.Response == nil {
		return "", nil, fmt.Errorf("state %q does not have a response", state)
	}

	newState, newOpaque := stateObj.Response(opaque, choice)
	return newState, newOpaque, nil
}

type WorkflowState struct {
	Challenge ChallengeFunc
	Response  ResponseFunc
}

// Challenge idempotently retrieves the challenge for the named state with the given opaque data
type ChallengeFunc func(opaque interface{}) *WorkflowMessage

// Response is idempotent from the state machine perspective but may have side effects like modifying Tabula or tokens.
// It executes the action associated with a state for the given choice.
// Usually the 'choice' is one of the options provided in the corresponding state's challenge's WorkflowMessage, but
// especially for `enter` states can be potentially any string.
type ResponseFunc func(opaque interface{}, choice *string) (newState string, newOpaque interface{})

type WorkflowMessage struct {
	Workflow string
	State    string
	Text     string
	Choices  []string
	Image    image.Image
}

func (wfm *WorkflowMessage) Id() string {
	/*	jsonBytes, err := json.Marshal([]string{wfm.Workflow, wfm.State})
		if err != nil {
			panic(fmt.Errorf("This was real simple, what happened? %s", err))
		}
		return string(jsonBytes)*/
	return wfm.Workflow
}

var Workflows = map[string]Workflow{
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
