package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
