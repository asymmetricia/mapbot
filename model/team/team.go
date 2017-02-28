package team

import (
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/user"
)

// A Team is a collection of `User`s and `Context`s, a Network Type, and opaque authentication information for the Network.
type Team struct {
	Users                []*user.User
	Contexts             []*context.Context
	NetworkType          string
	NetworkAuthenticator string
}
