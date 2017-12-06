package context

import (
	"github.com/pdbogen/mapbot/model/types"
	"image"
	"image/color"
)

type Context interface {
	Id() types.ContextId
	GetActiveTabulaId() *types.TabulaId
	SetActiveTabulaId(*types.TabulaId)
	GetZoom() (MinX, MinY, MaxX, MaxY int)
	SetZoom(MinX, MinY, MaxX, MaxY int)
	GetEmoji(name string) (image.Image, error)
	IsEmoji(name string) bool
	Mark(types.TabulaId, image.Point, color.Color)
	GetMarks(types.TabulaId) map[image.Point]color.Color
	ClearMarks(types.TabulaId)
	Save() error
}
