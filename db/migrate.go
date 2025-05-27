package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

var (
	//go:embed migrations/*.sql
	migrationsFS embed.FS
)

func (q *Queries) Close() error {
	return q.db.(*sql.DB).Close()
}

func NewWithMigrations(filename string) (*sql.DB, *Queries, error) {
	// Connect to the database.
	conn, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, nil, err
	}

	// Load migration files.
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, nil, err
	}

	// Configure a migration driver.
	driver, err := sqlite3.WithInstance(conn, &sqlite3.Config{
		MigrationsTable: "migrations",
	})
	if err != nil {
		return nil, nil, err
	}

	// Create a migrator.
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return nil, nil, err
	}
	m.Log = &slogger{}

	// Run all unapplied migrations.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, nil, err
	}

	return conn, New(conn), nil
}

type slogger struct {
}

func (s *slogger) Printf(format string, v ...any) {
	slog.Info("running migration", "msg", fmt.Sprintf(format, v...))
}

func (s *slogger) Verbose() bool {
	return false
}
