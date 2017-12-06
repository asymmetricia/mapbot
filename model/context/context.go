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
	GetZoom(types.TabulaId) (MinX, MinY, MaxX, MaxY int)
	SetZoom(TabId types.TabulaId, MinX, MinY, MaxX, MaxY int)
	GetEmoji(name string) (image.Image, error)
	IsEmoji(name string) bool
	Mark(types.TabulaId, image.Point, color.Color)
	GetMarks(types.TabulaId)
	ClearMarks(types.TabulaId)
	Save() error
}
