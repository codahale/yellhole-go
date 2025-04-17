package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssetsPublic(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/favicon.ico", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := len(body), 15406; got != want {
		t.Errorf("len(body) = %d, want = %d", got, want)
	}
}
