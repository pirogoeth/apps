package database

import (
	"context"
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
	_ "github.com/mattn/go-sqlite3"
)

type DbWrapper struct {
	*Queries
	db *sql.DB
}

func Open(ctx context.Context, path string) (*DbWrapper, error) {
	db, err := sql.Open("sqlite3", path+"?_fk=1")
	if err != nil {
		return nil, err
	}

	return &DbWrapper{
		Queries: New(db),
		db:      db,
	}, nil
}

func (dw *DbWrapper) Close() error {
	return dw.db.Close()
}

func (dw *DbWrapper) RunMigrations(migrationsFS embed.FS) error {
	goose.SetBaseFS(migrationsFS)
	
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(dw.db, "migrations"); err != nil {
		return err
	}

	return nil
}

func (dw *DbWrapper) DB() *sql.DB {
	return dw.db
}