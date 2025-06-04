package main

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/codahale/yellhole-go/internal/db"
	"github.com/codahale/yellhole-go/internal/imgstore"
)

type testApp struct {
	queries *db.Queries
	tempDir string
	t       *testing.T
	http.Handler
}

func newTestApp(t *testing.T) *testApp {
	logger := slog.New(slog.DiscardHandler)
	t.Helper()

	tempDir := t.TempDir()
	conn, queries, err := db.NewWithMigrations(logger, filepath.Join(tempDir, "yellhole.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := queries.Close(); err != nil {
			t.Fatal(err)
		}

		if err := conn.Close(); err != nil {
			t.Fatal(err)
		}
	})

	images, err := imgstore.New(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := images.Close(); err != nil {
			t.Fatal(err)
		}
	})

	app, err := newApp(t.Context(), logger, queries, images, "http://example.com", "Test Man", "Test Yell", "Gotta go fast.", "en", "00000000", false)
	if err != nil {
		t.Fatal(err)
	}

	return &testApp{queries, tempDir, t, app}
}
