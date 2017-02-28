package context

import (
	"github.com/pdbogen/mapbot/model/types"
	"image"
)

// A Context is a conceptual delineation that can have an active map, such as a channel, game session, or forum thread.
// Because, for Slack, Emoji can differe between teams, a Context also includes a mechanism to convert from emoji names
// to images.
type Context struct {
	Id           types.ContextId
	ActiveTabula *types.TabulaId
	GetEmoji     func(name string) (image.Image, error)
	IsEmoji      func(name string) bool
}
