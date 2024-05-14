package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

//go:embed schema.sql
var dbSchema string

func Open(ctx context.Context, path string) (*Queries, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is empty")
	}

	logrus.Infof("Connecting to sqlite3 database at %s", path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	logrus.Infof("Applying database schema")
	logrus.Debugf("Database schema being applied: %s", dbSchema)
	if _, err := db.ExecContext(ctx, dbSchema); err != nil {
		return nil, fmt.Errorf("could not run schema command: %w", err)
	}

	return New(db), nil
}