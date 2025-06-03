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

func NewWithMigrations(logger *slog.Logger, filename string) (*sql.DB, *Queries, error) {
	// Connect to the database.
	conn, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open SQLite database %s: %w", filename, err)
	}

	// Initialize the database settings.
	_, err = conn.Exec(initSQL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize database settings: %w", err)
	}

	// Load migration files.
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load migration files: %w", err)
	}

	// Configure a migration driver.
	driver, err := sqlite3.WithInstance(conn, &sqlite3.Config{
		MigrationsTable: "migrations",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to configure migration driver: %w", err)
	}

	// Create a migrator.
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create migrator: %w", err)
	}
	m.Log = &slogger{logger}

	// Run all unapplied migrations.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return conn, New(conn), nil
}

type slogger struct {
	logger *slog.Logger
}

func (s *slogger) Printf(format string, v ...any) {
	s.logger.Info("running migration", "msg", fmt.Sprintf(format, v...))
}

func (s *slogger) Verbose() bool {
	return false
}
