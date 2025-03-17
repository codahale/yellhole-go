package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/codahale/yellhole-go/config"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	os.Exit(m.Run())
}

type testApp struct {
	app *app
	t   *testing.T
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	baseURL, err := url.Parse("http://example.com/")
	if err != nil {
		t.Fatal(err)
	}

	config := &config.Config{
		Addr:        "localhost:8080",
		BaseURL:     baseURL,
		DataDir:     t.TempDir(),
		Title:       "Test Yell",
		Description: "Gotta go fast.",
		RequestLog:  false,
	}
	app, err := newApp(config)
	if err != nil {
		t.Fatal(err)
	}

	return &testApp{app, t}
}

func (e *testApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.t.Helper()
	e.app.ServeHTTP(w, r)
}

func (e *testApp) teardown() {
	e.t.Helper()

	if err := e.app.close(); err != nil {
		e.t.Fatal(err)
	}
}
