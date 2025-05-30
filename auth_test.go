package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/codahale/yellhole-go/internal/db"
	"github.com/go-webauthn/webauthn/webauthn"
)

func TestRegisterPage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/register", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestRegisterPageWithCredential(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	if err := app.queries.CreateWebauthnCredential(t.Context(), &db.JSONCredential{
		Data: &webauthn.Credential{ID: []byte("test-id")},
	}, time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/register", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusSeeOther; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := resp.Header.Get("location"), "http://example.com/login"; got != want {
		t.Errorf("resp.Header.Get(\"location\") = %q, want = %q", got, want)
	}
}

func TestLoginPage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	if err := app.queries.CreateWebauthnCredential(t.Context(), &db.JSONCredential{
		Data: &webauthn.Credential{ID: []byte("test-id")},
	}, time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/login", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestLoginPageWithoutCredential(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/login", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusSeeOther; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := resp.Header.Get("location"), "http://example.com/register"; got != want {
		t.Errorf("resp.Header.Get(\"location\") = %q, want = %q", got, want)
	}
}
