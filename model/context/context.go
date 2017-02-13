package context

import (
	"github.com/pdbogen/mapbot/model/types"
	"image"
)

type Context interface {
	Id() types.ContextId
	GetActiveTabulaId() *types.TabulaId
	SetActiveTabulaId(*types.TabulaId)
	GetEmoji(name string) (image.Image, error)
	IsEmoji(name string) bool
	Save() error
}