package database

import (
	"context"
	"database/sql"
	"embed"
	_ "embed"
	"fmt"
	"io"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/sirupsen/logrus"
)

//go:embed migrations/*.sql
var dbSchema embed.FS

var _ io.Closer = (*DbWrapper)(nil)

type DbWrapper struct {
	*Queries

	db *sql.DB
}

func Wrap(db *sql.DB) *DbWrapper {
	// TODO: Check from config parameter if database debugging is turned on
	// and conditionally install the dbLogger wrapper
	return &DbWrapper{
		Queries: New(newDbLogger(db)),
		db:      db,
	}
}

func (w *DbWrapper) Close() error {
	return w.db.Close()
}

func (w *DbWrapper) Querier() *Queries {
	return w.Queries
}

func Open(ctx context.Context, path string) (*DbWrapper, error) {
	sqlite_vec.Auto()

	if path == "" {
		return nil, fmt.Errorf("database path is empty")
	}

	logrus.Infof("Connecting to sqlite3 database at %s", path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	logrus.Infof("Applying database migrations")

	goose.SetLogger(logrus.StandardLogger())
	goose.SetBaseFS(dbSchema)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("could not set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("could not apply database migrations: %w", err)
	}

	// Ensure sqlite_vec loaded
	var vecVersion string
	err = db.QueryRow("select vec_version()").Scan(&vecVersion)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Debugf("loaded sqlite-vec version=%s", vecVersion)

	return Wrap(db), nil
}
