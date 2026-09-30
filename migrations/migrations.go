package migrations

import (
	"context"
	"database/sql"
	"time"
)

type Result struct {
	Version  int
	Name     string
	Applied  bool
	Duration time.Duration
}

type Migration struct {
	Version int
	Name    string
	Up      func(ctx context.Context, tx *sql.Tx) error
	Down    func(ctx context.Context, tx *sql.Tx) error
}

var All = []Migration{
	createSitesMigration,
	createReactorsMigration,
}
