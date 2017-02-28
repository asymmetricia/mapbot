package psql

import "github.com/pdbogen/mapbot/persist"

type PsqlPersister struct{}

var _ persist.Persister = (*PsqlPersister)(nil)