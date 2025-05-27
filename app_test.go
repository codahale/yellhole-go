package main

import (
	db2 "github.com/codahale/yellhole-go/internal/db"
	"net/http"
	"path/filepath"
	"testing"
)

type testApp struct {
	queries *db2.Queries
	tempDir string
	t       *testing.T
	http.Handler
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	tempDir := t.TempDir()
	conn, queries, err := db2.NewWithMigrations(filepath.Join(tempDir, "yellhole.db"))
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

	app, err := newApp(t.Context(), queries, "http://example.com", tempDir, "Test Man", "Test Yell", "Gotta go fast.", "en", false)
	if err != nil {
		t.Fatal(err)
	}

	return &testApp{queries, tempDir, t, app}
}
