package workflow

import (
	"encoding/json"
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	mbDraw "github.com/pdbogen/mapbot/common/draw"
	. "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/context/databaseContext"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"github.com/pdbogen/mapbot/model/user"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
)

var log = Log

var alignWorkflow = Workflow{
	States: map[string]WorkflowState{
		"enter": {
			Response: alignEnterResponse,
		},
		"confirm": {
			Challenge: alignConfirmChallenge,
			Response:  alignConfirmResponse,
		},
		"left": {
			Challenge: alignLeftChallenge,
			Response:  alignLeftResponse,
		},
		"rough": {
			Challenge: alignRoughDpiChallenge,
			Response:  alignRoughDpiResponse,
		},
		"fine": {
			Challenge: alignFineChallenge,
			Response:  alignFineResponse,
		},
		"fine_br": {
			Challenge: alignFineBRChallenge,
			Response:  alignFineBRResponse,
		},
		"top": {
			Challenge: alignTopChallenge,
			Response:  alignTopResponse,
		},
		"error": {
			Challenge: alignErrorChallenge,
		},
		"exit": {
			Challenge: alignExit,
		},
	},
	OpaqueFromJson: func(data []byte) (interface{}, error) {
		ret := &alignWorkflowOpaque{}
		if err := json.Unmarshal(data, ret); err != nil {
			return nil, err
		}
		return ret, nil
	},
}

type alignWorkflowOpaque struct {
	UserId     types.UserId
	User       *user.User `json:"-"`
	TabulaId   types.TabulaId
	Tabula     *tabula.Tabula `json:"-"`
	Top, Left  int
	Min, Max   int // generically used for binary searching.
	MinF, MaxF float32
	Error      bool // set when we're exiting due to error
}

func (a *alignWorkflowOpaque) MapImage(minX, minY, maxX, maxY int) draw.Image {
	//ctx := &databaseContext.DatabaseContext{}

	img, err := a.Tabula.BackgroundImage()
	if err != nil {
		log.Errorf("failed to render tabula: %s", err)
		return nil
	}

	img = tabula.Crop(img, minX, minY, maxX, maxY)

	if dimg, ok := img.(draw.Image); ok {
		return dimg
	}

	panic("inconceivably, tabula.Crop did not return a `draw.Image`")
	return nil
}

func VerticalLine(in draw.Image, x int) (out draw.Image) {
	mbDraw.Line(in, image.Pt(x, 0), image.Pt(x, in.Bounds().Max.Y), color.NRGBA{255, 0, 0, 255})
	return in
}

func horizontalLine(in draw.Image, y int) (out draw.Image) {
	mbDraw.Line(in, image.Pt(0, y), image.Pt(in.Bounds().Max.X, y), color.NRGBA{255, 0, 0, 255})
	return in
}

func (a *alignWorkflowOpaque) Hydrate() error {
	userObj, err := user.Get(db.Instance, a.UserId)
	if err != nil {
		return fmt.Errorf("hydrating user %q: %s", a.UserId, err)
	}
	a.User = userObj

	for _, t := range userObj.Tabulas {
		if *t.Id == a.TabulaId {
			a.Tabula = t
		}
	}

	if a.Tabula == nil {
		return fmt.Errorf("user %q does not have tabula id %d", a.UserId, a.TabulaId)
	}

	return nil
}

func alignError(err string, fields ...interface{}) (string, string) {
	log.Error(fmt.Sprintf(err, fields...))
	return "error", fmt.Sprintf(err, fields...)
}

func alignEnterResponse(opaque interface{}, choice *string) (string, interface{}) {
	if choice == nil {
		return alignError("invalid choice on enter state, expected <userid> <tabulaid>")
	}

	parts := strings.Split(*choice, " ")

	if len(parts) != 2 {
		return alignError("invalid choice on enter state, expected <userid> <tabulaid>")
	}

	tid, err := strconv.Atoi(parts[1])
	if err != nil {
		return alignError("could not parse tabula ID %q as integer: %s", parts[1], err)
	}

	state := &alignWorkflowOpaque{
		UserId:   types.UserId(parts[0]),
		TabulaId: types.TabulaId(tid),
	}

	if err := state.Hydrate(); err != nil {
		return alignError("could not hydrate initial opaque state: %s", err)
	}

	return "confirm", state
}

var alignConfirmYes = "Yep! Let's go."
var alignConfirmNo = "Oh, maybe not..."

func alignConfirmChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}

	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	return &WorkflowMessage{
		Text:    fmt.Sprintf("Would you like to use the guided alignment tool? This process will reset the DPI and Offsets for your map `%s`. Is that ok?", state.Tabula.Name),
		Choices: []string{alignConfirmYes, alignConfirmNo},
	}
}

func alignConfirmResponse(opaque interface{}, choice *string) (string, interface{}) {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignError("invalid opaque data (was a %T)", opaque)
	}

	if err := state.Hydrate(); err != nil {
		return alignError("could not hydrate opaque data: %s", err)
	}

	if choice == nil {
		return alignError("huh, got a nil string pointer...")
	}
	switch *choice {
	case alignConfirmYes:
		state.Top = -50
		state.Left = -50
		state.Tabula.Dpi = 50
		state.Tabula.OffsetX = 0
		state.Tabula.OffsetY = 0
		if err := state.Tabula.Save(db.Instance); err != nil {
			return alignError("huh! couldn't save the table: %s", err)
		}
		state.Min = -50
		state.Max = 250
		return "left", state
	case alignConfirmNo:
		return "error", "Ok! See you later."
	default:
		return alignError("I don't know what you meant by %s", *choice)
	}
}

func alignErrorChallenge(opaque interface{}) *WorkflowMessage {
	errMsg, ok := opaque.(string)
	if !ok {
		errMsg = "error state received non-string opaque data"
	}
	return &WorkflowMessage{Text: errMsg, State: "exit"}
}

var (
	alignUp         = "Pan Up"
	alignDown       = "Pan Down"
	alignLeft       = "Pan Left"
	alignRight      = "Pan Right"
	alignUpRight    = "Up & Right"
	alignUpLeft     = "Up & Left"
	alignDownRight  = "Down & Right"
	alignDownLeft   = "Down & Left"
	alignSmaller    = "Smaller"
	alignRoughOk    = "About Right"
	alignBigger     = "Bigger"
	alignLeftOf     = "Left-of"
	alignRightOf    = "Right-of"
	alignPerfect    = "Perfect!"
	alignRestart    = "Restart"
	alignShiftLeft  = "Left 1px"
	alignShiftRight = "Right 1px"
	alignAbove      = "Above"
	alignBelow      = "Below"
)

/////////////////////////////
/// X-Offset Calibration ///
///////////////////////////

func alignLeftChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	img := state.MapImage(state.Left, state.Top, state.Left+250, state.Top+250)

	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	return &WorkflowMessage{
		Text: "First, we need to find the left edge of the map grid. Sometimes " +
			"this is the very edge of the image, but sometimes it's farther to " +
			"the right. _(If it's not visible, you can pan around to find it.)_",
		State: "left",
		Image: VerticalLine(img, state.Tabula.OffsetX-state.Left),
		ChoiceSets: [][]string{
			{alignLeftOf, alignPerfect, alignRightOf},
			{alignUp, alignDown, alignLeft, alignRight},
			{alignRestart},
		},
	}
}

func alignLeftResponse(opaque interface{}, choice *string) (string, interface{}) {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignError("invalid opaque data (was a %T)", opaque)
	}

	if err := state.Hydrate(); err != nil {
		return alignError(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	if choice == nil {
		return alignError("huh, got a nil string pointer...")
	}
	switch *choice {
	case alignRightOf:
		state.Max = state.Tabula.OffsetX
		state.Tabula.OffsetX = (state.Max + state.Min) / 2
	case alignLeftOf:
		state.Min = state.Tabula.OffsetX
		state.Tabula.OffsetX = (state.Max + state.Min) / 2
	case alignRestart:
		state.Min = -int(math.Ceil(float64(state.Tabula.Dpi)))
		state.Max = int(math.Ceil(float64(state.Tabula.Dpi)))
		state.Top = -50
		state.Left = -50
		state.Tabula.OffsetX = 0
	case alignUp:
		state.Top -= 250
		if state.Top < -50 {
			state.Top = -50
		}
	case alignDown:
		state.Left += 250
	case alignRight:
		state.Left += 250
		state.Tabula.OffsetX += 250
		state.Min += 250
		state.Max += 250
	case alignLeft:
		state.Left -= 250
		state.Tabula.OffsetX -= 250
		state.Min -= 250
		state.Max -= 250
		if state.Left < -50 {
			state.Left += 250
			state.Tabula.OffsetX += 250
			state.Min += 250
			state.Max += 250
		}
	case alignPerfect:
		state.Min = 1
		state.Max = 300
		return "rough", state
	}

	if err := state.Tabula.Save(db.Instance); err != nil {
		return alignError("huh! couldn't save the table: %s", err)
	}

	return "left", state
}

//////////////////////////////
/// Rough DPI Calibration ///
////////////////////////////

func alignRoughDpiChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	vline := state.Tabula.OffsetX - state.Left + int(state.Tabula.Dpi)
	right := state.Left + 250

	if vline > right {
		right = vline + 50
	}

	img := state.MapImage(state.Left, state.Top, right, state.Top+250)
	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	return &WorkflowMessage{
		Text: "Now we can get our rough DPI. Is the red line left-of or right-of the *second* grid line?" +
			fmt.Sprintf(" _(current DPI: %d)_", int(state.Tabula.Dpi)),
		State: "rough",
		Image: VerticalLine(img, vline),
		ChoiceSets: [][]string{
			{alignLeftOf, alignPerfect, alignRightOf},
			{alignUp, alignDown, alignLeft, alignRight},
			{alignRestart},
		},
	}
}

func alignRoughDpiResponse(opaque interface{}, choice *string) (string, interface{}) {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignError("invalid opaque data (was a %T)", opaque)
	}

	if err := state.Hydrate(); err != nil {
		return alignError(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	if choice == nil {
		return alignError("huh, got a nil string pointer...")
	}
	switch *choice {
	case alignLeftOf:
		state.Min = int(state.Tabula.Dpi)
		state.Tabula.Dpi = float32((state.Min + state.Max) / 2)
	case alignRightOf:
		state.Max = int(state.Tabula.Dpi)
		state.Tabula.Dpi = float32((state.Min + state.Max) / 2)
	case alignRestart:
		state.Min = 1
		state.Max = 300
		state.Tabula.Dpi = 50
	case alignPerfect:
		state.Left = state.Tabula.OffsetX - 5
		state.MinF = state.Tabula.Dpi * 0.9
		state.MaxF = state.Tabula.Dpi * 1.1
		state.Min = int(state.Tabula.Dpi)
		state.Max = int(state.Tabula.Dpi)
		return "fine", state
	case alignUp:
		state.Top -= 250
		if state.Top < -50 {
			state.Top = -50
		}
	case alignDown:
		state.Top += 250
	case alignRight:
		state.Left += 250
	case alignLeft:
		state.Left -= 250
		if state.Left < -50 {
			state.Left = -50
		}
	}

	if err := state.Tabula.Save(db.Instance); err != nil {
		return alignError("huh! couldn't save the table: %s", err)
	}

	return "rough", state
}

/////////////////////////////
/// Fine DPI Calibration ///
///////////////////////////

func alignFineChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	img := state.MapImage(state.Left, state.Top, state.Left+400, state.Top+400)
	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	i := 0
	for {
		x := state.Tabula.OffsetX - state.Left + int(float32(i)*state.Tabula.Dpi)
		if x > 400 {
			break
		}
		VerticalLine(img, x)
		i++
	}

	return &WorkflowMessage{
		Text: fmt.Sprintf("Now we can fine-tune the DPI. Current DPI: %0.2f\n\n", state.Tabula.Dpi) +
			"Smaller -- If the red grid lines advance to the right of the map grid lines, they need to be smaller.\n" +
			"Bigger -- If the red grid lines shrink to the left of the map grid lines, they need to be bigger.\n" +
			"_(you can also shift the entire grid left or right one pixel at a time, if you want)_",
		State: "fine",
		Image: img,
		ChoiceSets: [][]string{
			{alignSmaller, alignPerfect, alignBigger},
			{alignShiftLeft, alignShiftRight},
			{alignUp, alignDown, alignLeft, alignRight},
			{alignRestart},
		},
	}
}
func alignFineResponse(opaque interface{}, choice *string) (string, interface{}) {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignError("invalid opaque data (was a %T)", opaque)
	}

	if err := state.Hydrate(); err != nil {
		return alignError(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	if choice == nil {
		return alignError("huh, got a nil string pointer...")
	}
	switch *choice {
	case alignSmaller:
		state.MaxF = state.Tabula.Dpi
		state.Tabula.Dpi = (state.MaxF + state.MinF) / 2
	case alignBigger:
		state.MinF = state.Tabula.Dpi
		state.Tabula.Dpi = (state.MaxF + state.MinF) / 2
	case alignRestart:
		state.MinF = float32(state.Min) * 0.9
		state.MaxF = float32(state.Max) * 1.1
		state.Tabula.Dpi = (state.MaxF + state.MinF) / 2
	case alignShiftLeft:
		state.Tabula.OffsetX--
	case alignShiftRight:
		state.Tabula.OffsetX++
	case alignUp:
		state.Top -= 250
		if state.Top < -50 {
			state.Top = -50
		}
	case alignDown:
		state.Top += 250
	case alignRight:
		state.Left += 250
	case alignLeft:
		state.Left -= 250
		if state.Left < -50 {
			state.Left = -50
		}
	case alignPerfect:
		bg, err := state.Tabula.BackgroundImage()
		if err != nil {
			return alignError("surprising that we should be unable now to get the background image, but: %s", err)
		}
		state.Top = bg.Bounds().Max.Y - 400
		state.Left = bg.Bounds().Max.X - 400
		//state.MinF = state.Tabula.Dpi * .99
		//state.MaxF = state.Tabula.Dpi * 1.01
		return "fine_br", state
	}
	if err := state.Tabula.Save(db.Instance); err != nil {
		return alignError("huh! couldn't save the table: %s", err)
	}

	return "fine", state
}

///////////////////////////////////////////
/// Fine DPI Calibration, Botton-Right ///
/////////////////////////////////////////

func alignFineBRChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	img := state.MapImage(state.Left, state.Top, state.Left+400, state.Top+400)
	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	i := 0
	for {
		x := state.Tabula.OffsetX - state.Left + int(float32(i)*state.Tabula.Dpi)
		if x > 400 {
			break
		}
		VerticalLine(img, x)
		i++
	}

	return &WorkflowMessage{
		Text: fmt.Sprintf("Last bit, let's check the spacing in the bottom-right corner. DPI: %0.2f\n\n", state.Tabula.Dpi) +
			"Smaller -- If the red grid lines advance to the right of the map grid lines, they need to be smaller.\n" +
			"Bigger -- If the red grid lines shrink to the left of the map grid lines, they need to be bigger.\n" +
			"_(you can still shift the entire grid left or right one pixel at a time, if you want)_",
		State: "fine_br",
		Image: img,
		ChoiceSets: [][]string{
			{alignSmaller, alignPerfect, alignBigger},
			{alignShiftLeft, alignShiftRight},
			{alignUp, alignDown, alignLeft, alignRight},
			{alignRestart},
		},
	}
}
func alignFineBRResponse(opaque interface{}, choice *string) (string, interface{}) {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignError("invalid opaque data (was a %T)", opaque)
	}

	if err := state.Hydrate(); err != nil {
		return alignError(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	if choice == nil {
		return alignError("huh, got a nil string pointer...")
	}
	switch *choice {
	case alignSmaller:
		state.MaxF = state.Tabula.Dpi
		state.Tabula.Dpi = (state.MaxF + state.MinF) / 2
	case alignBigger:
		state.MinF = state.Tabula.Dpi
		state.Tabula.Dpi = (state.MaxF + state.MinF) / 2
	case alignRestart:
		state.MinF = float32(state.Min) * 0.9
		state.MaxF = float32(state.Max) * 1.1
		state.Tabula.Dpi = (state.MaxF + state.MinF) / 2
	case alignShiftLeft:
		state.Tabula.OffsetX--
	case alignShiftRight:
		state.Tabula.OffsetX++
	case alignUp:
		state.Top -= 250
		if state.Top < -50 {
			state.Top = -50
		}
	case alignDown:
		state.Top += 250
	case alignRight:
		state.Left += 250
	case alignLeft:
		state.Left -= 250
		if state.Left < -50 {
			state.Left = -50
		}
	case alignPerfect:
		state.Top = -50
		state.Left = -50
		state.Min = -50
		state.Max = int(state.Tabula.Dpi)
		state.Tabula.OffsetY = 0
		return "top", state
	}
	if err := state.Tabula.Save(db.Instance); err != nil {
		return alignError("huh! couldn't save the table: %s", err)
	}

	return "fine_br", state
}

/////////////////////////////
/// Y-Offset Calibration ///
///////////////////////////

func alignTopChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	img := state.MapImage(state.Left, state.Top, state.Left+250, state.Top+250)
	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	return &WorkflowMessage{
		Text: "This is the home stretch; we just need to align the top line. Is " +
			"the red line **above** or **below** the top grid line?",
		State: "top",
		Image: horizontalLine(img, -state.Top+state.Tabula.OffsetY),
		ChoiceSets: [][]string{
			{alignAbove},
			{alignPerfect},
			{alignBelow},
			{alignUp, alignDown, alignLeft, alignRight},
			{alignRestart},
		},
	}
}

func alignTopResponse(opaque interface{}, choice *string) (string, interface{}) {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignError("invalid opaque data (was a %T)", opaque)
	}

	if err := state.Hydrate(); err != nil {
		return alignError(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	if choice == nil {
		return alignError("huh, got a nil string pointer...")
	}
	switch *choice {
	case alignAbove:
		state.Min = state.Tabula.OffsetY
		state.Tabula.OffsetY = (state.Min + state.Max) / 2
	case alignBelow:
		state.Max = state.Tabula.OffsetY
		state.Tabula.OffsetY = (state.Min + state.Max) / 2
	case alignPerfect:
		return "exit", state
	case alignUp:
		state.Top -= 250
		if state.Top < -50 {
			state.Top = -50
		}
	case alignDown:
		state.Top += 250
	case alignRight:
		state.Left += 250
	case alignLeft:
		state.Left -= 250
		if state.Left < -50 {
			state.Left = -50
		}
	}

	if err := state.Tabula.Save(db.Instance); err != nil {
		return alignError("huh! couldn't save the table: %s", err)
	}

	return "top", state
}

func alignExit(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return nil
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	img, err := state.Tabula.Render(&databaseContext.DatabaseContext{}, nil)

	if err != nil {
		return alignErrorChallenge(fmt.Sprintf("during final map render: %s", err))
	}

	return &WorkflowMessage{
		Text:  "You're done! Thanks for playing!",
		Image: img,
		State: "exit",
	}
}
