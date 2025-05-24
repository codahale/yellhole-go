package main

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/codahale/yellhole-go/db"
)

type testApp struct {
	queries *db.Queries
	tempDir string
	t       *testing.T
	http.Handler
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	tempDir := t.TempDir()
	queries, err := db.NewWithMigrations(filepath.Join(tempDir, "yellhole.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := queries.Close(); err != nil {
			t.Fatal(err)
		}
	})

	app, err := newApp(t.Context(), queries, "http://example.com", tempDir, "Test Man", "Test Yell", "Gotta go fast.", "en", false)
	if err != nil {
		t.Fatal(err)
	}

	return &testApp{queries, tempDir, t, app}
}
