package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

func RunMigrations(db *sql.DB) error {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}
	driver, err := sqlite.WithInstance(db, &sqlite.Config{
		MigrationsTable: "migrations",
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return err
	}
	m.Log = &slogger{}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

type slogger struct {
}

func (s *slogger) Printf(format string, v ...any) {
	slog.Info("running migration", "msg", fmt.Sprintf(format, v...))
}

func (s *slogger) Verbose() bool {
	return false
}
