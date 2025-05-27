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

const initSQL = `
	PRAGMA journal_mode = WAL;
	PRAGMA synchronous = NORMAL;
	PRAGMA temp_store = MEMORY;
	PRAGMA mmap_size = 30000000000; -- 30GB
	PRAGMA busy_timeout = 5000;
	PRAGMA automatic_index = true;
	PRAGMA foreign_keys = ON;
	PRAGMA analysis_limit = 1000;
	PRAGMA trusted_schema = OFF;
`

func NewWithMigrations(filename string) (*sql.DB, *Queries, error) {
	// Connect to the database.
	conn, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, nil, err
	}

	// Initialize the database settings.
	_, err = conn.Exec(initSQL)
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
