package context

import "github.com/pdbogen/mapbot/model/types"

type ContextProviderFunc func(id types.ContextId) (Context, error)

// ContextProvider is used by asynchronous types of communication to rehydrate contexts.
type ContextProvider struct {
	ContextTypes map[types.ContextType]ContextProviderFunc
}
