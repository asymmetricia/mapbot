package workflow

import (
	"encoding/json"
	"fmt"
	"github.com/pdbogen/mapbot/common/db"
	mbDraw "github.com/pdbogen/mapbot/common/draw"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/model/context/databaseContext"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"github.com/pdbogen/mapbot/model/user"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
)

var log = mbLog.Log

var alignWorkflow = Workflow{
	States: map[string]WorkflowState{
		"enter": {
			Response: alignEnterResponse,
		},
		"confirm": {
			Challenge: alignConfirmChallenge,
			Response:  alignConfirmResponse,
		},
		"vertical_a": {
			Challenge: alignVerticalAChallenge,
			Response:  alignVerticalAResponse,
		},
		"vertical_b": {
			Challenge: alignVerticalBChallenge,
			Response:  alignVerticalBResponse,
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
	UserId               types.UserId
	User                 *user.User `json:"-"`
	TabulaId             types.TabulaId
	Tabula               *tabula.Tabula `json:"-"`
	Top, Left            int
	Min, Max             int // generically used for binary searching.
	MinF, MaxF           float32
	Error                bool // set when we're exiting due to error
	VerticalA, VerticalB int
	SavedDpi             float32
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

func VerticalLine(in draw.Image, x int, col ...color.NRGBA) (out draw.Image) {
	if len(col) == 0 {
		col = []color.NRGBA{{255, 0, 0, 255}}
	}
	mbDraw.Line(in, image.Pt(x, 0), image.Pt(x, in.Bounds().Max.Y), col[0])
	return in
}

func horizontalLine(in draw.Image, y int, col ...color.NRGBA) (out draw.Image) {
	if len(col) == 0 {
		col = []color.NRGBA{{255, 0, 0, 255}}
	}
	mbDraw.Line(in, image.Pt(0, y), image.Pt(in.Bounds().Max.X, y), col[0])
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

/////////////////////////////////////////////
/// Confirmation of Ready To Rock status ///
///////////////////////////////////////////

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
		state.Tabula.Dpi = 50
		state.Tabula.OffsetX = 0
		state.Tabula.OffsetY = 0
		if err := state.Tabula.Save(db.Instance); err != nil {
			return alignError("huh! couldn't save the table: %s", err)
		}

		state.Top = -50
		state.Left = -50
		state.Min = -50
		state.Max = 310
		state.VerticalA = (state.Max + state.Min) / 2
		return "vertical_a", state
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

//////////////////////////////////////
/// Mk2 Alignment, First Vertical ///
////////////////////////////////////

func alignVerticalAChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	img := state.MapImage(state.Left, state.Top, state.Left+360, state.Top+360)
	VerticalLine(img, state.VerticalA-state.Left)

	return &WorkflowMessage{
		Text:  "Choose a pair of vertical lines. Is the red line left-of or right-of the left line? _(If you don't see a good pair of map lines, you can use the Pan buttons to shift the view.)_",
		State: "vertical_a",
		Image: img,
		ChoiceSets: [][]string{
			{alignLeftOf, alignPerfect, alignRightOf},
			{alignUp, alignDown, alignLeft, alignRight},
			{alignRestart},
		},
	}
}

func alignVerticalAResponse(opaque interface{}, choice *string) (string, interface{}) {
	shift := 200
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
		state.Max = state.VerticalA
		state.VerticalA = (state.Max + state.Min) / 2
		if state.VerticalA < state.Left {
			state.Left = state.VerticalA - 50
		}
	case alignLeftOf:
		state.Min = state.VerticalA
		state.VerticalA = (state.Max + state.Min) / 2
		if state.VerticalA > state.Left+360 {
			state.Left = state.VerticalA - 310
		}
	case alignRestart:
		state.Min = -50
		state.Max = 310
		state.VerticalA = (state.Max + state.Min) / 2
		state.Left = state.VerticalA - 180
	case alignUp:
		state.Top -= shift
		if state.Top < -50 {
			state.Top = -50
		}
	case alignDown:
		state.Top += shift
	case alignRight:
		state.Left += shift
		state.VerticalA += shift
		state.Min += shift
		state.Max += shift
	case alignLeft:
		if state.Left-shift >= -50 {
			state.Left -= shift
			state.VerticalA -= shift
			state.Min -= shift
			state.Max -= shift
		}
	case alignPerfect:
		state.Min = state.VerticalA
		state.Max = state.VerticalA + 360
		state.VerticalB = state.VerticalA + 50
		return "vertical_b", state
	}

	log.Debugf("align vertical_a %q -- %d...%d...%d", state.Tabula.Name, state.Min, state.VerticalA, state.Max)
	if err := state.Tabula.Save(db.Instance); err != nil {
		return alignError("huh! couldn't save the table: %s", err)
	}

	return "vertical_a", state
}

///////////////////////////////////////
/// Mk2 Alignment, Second Vertical ///
/////////////////////////////////////

func alignVerticalBChallenge(opaque interface{}) *WorkflowMessage {
	state, ok := opaque.(*alignWorkflowOpaque)
	if !ok {
		return alignErrorChallenge(fmt.Sprintf("invalid opaque data (was a %T)", opaque))
	}
	if err := state.Hydrate(); err != nil {
		return alignErrorChallenge(fmt.Sprintf("could not hydrate opaque data: %s", err))
	}

	img := state.MapImage(state.Left, state.Top, state.Left+360, state.Top+360)
	VerticalLine(img, state.VerticalA-state.Left)
	VerticalLine(img, state.VerticalB-state.Left, color.NRGBA{0, 0, 255, 255})

	return &WorkflowMessage{
		Text:  "Choose a pair of vertical lines. Is the **blue** line left-of or right-of the **right** line? _(If you don't see a good pair of map lines, you can use the Pan buttons to shift the view.)_",
		State: "vertical_b",
		Image: img,
		ChoiceSets: [][]string{
			{alignLeftOf, alignPerfect, alignRightOf},
			{alignUp, alignDown, alignLeft, alignRight},
			{alignRestart},
		},
	}
}

func alignVerticalBResponse(opaque interface{}, choice *string) (string, interface{}) {
	shift := 200
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
		state.Max = state.VerticalB
		state.VerticalB = (state.Max + state.Min) / 2
		if state.VerticalB < state.Left {
			state.Left = state.VerticalB - 50
		}
	case alignLeftOf:
		state.Min = state.VerticalB
		state.VerticalB = (state.Max + state.Min) / 2
		if state.VerticalB > state.Left+360 {
			state.Left = state.VerticalB - 310
		}
	case alignRestart:
		state.Min = state.VerticalA
		state.Max = state.VerticalA + 360
		state.VerticalB = state.VerticalA + 50
		state.Left = state.VerticalB - 180
	case alignUp:
		state.Top -= shift
		if state.Top < -50 {
			state.Top = -50
		}
	case alignDown:
		state.Top += shift
	case alignRight:
		state.Left += shift
		state.Min += shift
		state.Max += shift
	case alignLeft:
		if state.Left-shift >= -50 {
			state.Left -= shift
			state.Min -= shift
			state.Max -= shift
		}
	case alignPerfect:
		log.Infof("rough dpi %d", state.VerticalB-state.VerticalA)
		state.Tabula.Dpi = float32(state.VerticalB - state.VerticalA)
		// Calculate many DPI our line is from the left side of the image, then shift it about over that many.
		state.Tabula.OffsetX = state.VerticalA - int(state.Tabula.Dpi*float32(int(float32(state.VerticalA)/state.Tabula.Dpi+0.2))+0.5)
		if err := state.Tabula.Save(db.Instance); err != nil {
			return alignError("huh! couldn't save the table: %s", err)
		}
		state.MinF = state.Tabula.Dpi * 0.9
		state.MaxF = state.Tabula.Dpi * 1.1
		return "fine", state
	}

	log.Debugf("align vertical_b %q -- %d...%d...%d", state.Tabula.Name, state.Min, state.VerticalB, state.Max)
	if err := state.Tabula.Save(db.Instance); err != nil {
		return alignError("huh! couldn't save the table: %s", err)
	}

	return "vertical_b", state
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

	img := state.MapImage(state.Left, state.Top, state.Left+360, state.Top+360)
	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	i := 0
	for {
		x := state.Tabula.OffsetX - state.Left + int(float32(i)*state.Tabula.Dpi)
		if x > 360 {
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
		state.MinF = state.Tabula.Dpi * .99
		state.MaxF = state.Tabula.Dpi * 1.01
		state.SavedDpi = state.Tabula.Dpi
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

	img := state.MapImage(state.Left, state.Top, state.Left+360, state.Top+360)
	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	i := 0
	for {
		x := state.Tabula.OffsetX - state.Left + int(float32(i)*state.Tabula.Dpi)
		if x > 360 {
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
		state.Tabula.Dpi = state.SavedDpi
		state.MinF = state.Tabula.Dpi * 0.99
		state.MaxF = state.Tabula.Dpi * 1.01
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
		state.Max = 310
		state.Tabula.OffsetY = (state.Min + state.Max) / 2
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

	img := state.MapImage(state.Left, state.Top, state.Left+360, state.Top+360)
	if img == nil {
		return alignErrorChallenge("sorry! something's wrong with the map image.")
	}

	return &WorkflowMessage{
		Text: "This is the home stretch; we just need to align the horizontal. Pan around until you find a good horizontal line. Is " +
			"the red line **above** or **below** that map line?",
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
	shift := 200
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
	case alignRestart:
		state.Top = -50
		state.Left = -50
		state.Min = -50
		state.Max = 310
		state.Tabula.OffsetY = (state.Min + state.Max) / 2
	case alignAbove:
		state.Min = state.Tabula.OffsetY
		state.Tabula.OffsetY = (state.Min + state.Max) / 2
	case alignBelow:
		state.Max = state.Tabula.OffsetY
		state.Tabula.OffsetY = (state.Min + state.Max) / 2
	case alignPerfect:
		// With a small fudge factor, calculate how many DPI below the top of the image we are, and then
		// shift back up that much.
		state.Tabula.OffsetY -= int(state.Tabula.Dpi*float32(int(float32(state.Tabula.OffsetY)/state.Tabula.Dpi+0.2)) + 0.5)
		if err := state.Tabula.Save(db.Instance); err != nil {
			return alignError("huh! couldn't save the table: %s", err)
		}
		return "exit", state
	case alignUp:
		if state.Top-shift >= -50 {
			state.Top -= shift
			state.Tabula.OffsetY -= shift
			state.Min -= shift
			state.Max -= shift
		}
	case alignDown:
		state.Top += shift
		state.Tabula.OffsetY += shift
		state.Min += shift
		state.Max += shift
	case alignRight:
		state.Left += shift
	case alignLeft:
		state.Left -= shift
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
		Text:  "Ok! We've done our best. Some maps don't have perfectly rectangular grids, but I hope this one turned out well.",
		Image: img,
		State: "exit",
	}
}
