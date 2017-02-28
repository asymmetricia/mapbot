package persist

import (
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/team"
	"github.com/pdbogen/mapbot/model/types"
	"github.com/pdbogen/mapbot/model/user"
	"time"
)

type Persister interface {
	ListTeams() ([]*team.Team, error)

	SaveTabula(*tabula.Tabula) error
	LoadTabula(types.TabulaId) (*tabula.Tabula, error)
	DeleteTabula(types.TabulaId) error

	SaveUser(*user.User) error
	LoadUser(user.Id) (*user.User, error)
	DeleteUser(user.Id) error

	SaveNonce(nonce string, expiry time.Time) error
	CheckNonce(nonce string) (bool, error)
	DeleteNonce(nonce string) error

	SaveContext(*context.Context) error
	LoadContext(types.ContextId) (*context.Context, error)
	DeleteContext(types.ContextId) error
}

var Persisters map[string]func([]string) (Persister, error)
