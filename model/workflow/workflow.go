// A workflow is a series of states with a context, initially a user and map.
package workflow

import (
	"encoding/json"
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

// Challenge looks up the state by name, and if a state is found and it has a challenge, returns the result of calling
// that state's Challenge.
// If the state cannot be found or it does not have a defined Challenge, a non-nil WorkflowMessage will provide a
// message for the user describing the problem (but this typically indicates misuse of a debugging utility or a
// programming error).
// A nil WorkflowMessage is not an error, but indicates that there is no response for the user.
func (wf *Workflow) Challenge(state string, opaque interface{}) *WorkflowMessage {
	stateObj, ok := wf.States[strings.ToLower(state)]
	if !ok {
		return &WorkflowMessage{Text: fmt.Sprintf("invalid state %q", state)}
	}
	if stateObj.Challenge == nil {
		return &WorkflowMessage{Text: fmt.Sprintf("no challenge associated with state %q", state)}
	}
	return stateObj.Challenge(opaque)
}

// Response looks up the state by name, and if a state is found and it has a Response, returns the result of calling
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
	jsonBytes, err := json.Marshal([]string{wfm.Workflow, wfm.State})
	if err != nil {
		panic(fmt.Errorf("This was real simple, what happened? %s", err))
	}
	return string(jsonBytes)
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
		},
	},
}

//// Workflow is initiated by receiving a first Callback with an empty value. Thereafter the workflow can continue to
//// receive callbacks until it returns a `true` value for `done`, which indicates the workflow is complete and should
//// be removed from memory.
//type Workflow interface {
//	Callback(value string) (msg WorkflowMessage, done bool)
//	Id() int64
//}
//
//type RandomId struct {
//	id int64
//}
//
//type AlignGrid struct {
//	Tabula *tabula.Tabula
//	state  int
//	ctx    *context.Context
//	RandomId
//}
//
//func NewAlignGrid(h *hub.Hub, t *tabula.Tabula, ctx *context.Context) *AlignGrid {
//	ag := &AlignGrid{
//		Tabula: t,
//	}
//	h.Subscribe(hub.CommandType(fmt.Sprintf("workflow:%d", ag.ID())), ag.Callback)
//	return ag
//}
//
//func (ag *RandomId) ID() int64 {
//	if ag.id == 0 {
//		ag.id = rand.Int63()
//	}
//	return ag.id
//}
//
//func (ag *AlignGrid) Callback(h *hub.Hub, c *hub.Command) {
//	var res WorkflowMessage
//	var done bool
//	switch ag.state {
//	case 0:
//		res, done = ag.initiate()
//	}
//
//	if done {
//		h.Unsubscribe(hub.CommandType(fmt.Sprintf("workflow:%d", ag.ID())))
//	}
//
//	origin := strings.Split(c.From, ":")
//
//	if len(origin) < 4 {
//		Log.Errorf("%s: could not parse origin CommandType %s", ag.ctx, c.From)
//		return
//	}
//
//	h.Publish(&hub.Command{
//		From:    fmt.Sprintf("workflow:%d", ag.ID()),
//		Type:    hub.CommandType(fmt.Sprintf("internal:workflow:%s:%s", origin[2], origin[3])),
//		Payload: res,
//	})
//}
//
//func (ag *AlignGrid) initiate() (wf WorkflowMessage, done bool) {
//	return WorkflowMessage{
//		Id:   ag.ID(),
//		Text: "Awesome! A new map. Let's get the grid aligned. In the image below, is the red line to the left or to the right of the first grid line? (Or exactly aligned?)",
//		Choices: []string{
//			"Left",
//			"Aligned",
//			"Right",
//			"No Grid Visible",
//		},
//	}, false
//}
