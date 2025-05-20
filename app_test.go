package main

import (
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/codahale/yellhole-go/db"
)

type testApp struct {
	app     http.Handler
	queries *db.Queries
	tempDir string
	t       *testing.T
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	baseURL, err := url.Parse("http://example.com/")
	if err != nil {
		t.Fatal(err)
	}

	tempDir := t.TempDir()
	queries, err := db.NewWithMigrations(t.Context(), filepath.Join(tempDir, "yellhole.db"))
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApp(t.Context(), queries, tempDir, "Test Man", "Test Yell", "Gotta go fast.", baseURL, false)
	if err != nil {
		t.Fatal(err)
	}

	return &testApp{app, queries, tempDir, t}
}

func (e *testApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.t.Helper()
	e.app.ServeHTTP(w, r)
}
