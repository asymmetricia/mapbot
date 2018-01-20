package context

import (
	"github.com/pdbogen/mapbot/model/mark"
	"github.com/pdbogen/mapbot/model/types"
	"image"
)

type Context interface {
	Id() types.ContextId
	GetActiveTabulaId() *types.TabulaId
	SetActiveTabulaId(*types.TabulaId)
	GetZoom() (MinX, MinY, MaxX, MaxY int)
	SetZoom(MinX, MinY, MaxX, MaxY int)
	GetEmoji(name string) (image.Image, error)
	IsEmoji(name string) bool
	Mark(types.TabulaId, mark.Mark)
	GetMarks(types.TabulaId) map[image.Point]map[string]mark.Mark
	ClearMarks(types.TabulaId)
	Save() error
	GetLastToken(UserId string) (TokenName string)
	SetLastToken(UserId string, TokenName string)
}
