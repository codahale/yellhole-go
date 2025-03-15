package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

//go:embed migrations/*.sql
var migrations embed.FS

func RunMigrations(db *sql.DB) error {
	// Enumerate all paths.
	paths, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	slices.Sort(paths)

	// Find the most recent migration number.
	row := db.QueryRow("PRAGMA user_version")
	if row == nil {
		return fmt.Errorf("unable to query user_version")
	}

	var userVersion int64
	if err := row.Scan(&userVersion); err != nil {
		return err
	}

	slog.Info("checking for migrations", "currentVersion", userVersion)

	// Go through the migrations and run the ones which haven't been run yet.
	for _, path := range paths {
		s := strings.TrimFunc(path, func(r rune) bool { return !unicode.IsNumber(r) })
		version, err := strconv.ParseInt(s, 10, 16)
		if err != nil {
			return err
		}

		if version > userVersion {
			slog.Info("applying migration", "version", version)

			b, err := fs.ReadFile(migrations, path)
			if err != nil {
				return fmt.Errorf("unable to read migration: %w", err)
			}

			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("unable to start transaction: %w", err)
			}

			if _, err := tx.Exec(string(b)); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("unable to run migration: %w", err)
			}

			if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("unable to update user_version: %w", err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("unable to commit transaction: %w", err)
			}
		}
	}

	return nil
}
