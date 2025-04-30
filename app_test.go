package main

import (
	"net/http"
	"net/url"
	"testing"
)

type testApp struct {
	app     *app
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
	config := &config{
		Addr:        "localhost:8080",
		BaseURL:     baseURL,
		DataDir:     tempDir,
		Title:       "Test Yell",
		Description: "Gotta go fast.",
		requestLog:  false,
	}
	app, err := newApp(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := app.close(); err != nil {
			t.Fatal(err)
		}
	})

	return &testApp{app, tempDir, t}
}

func (e *testApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.t.Helper()
	e.app.ServeHTTP(w, r)
}
