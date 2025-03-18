package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssetsPublic(t *testing.T) {
	env := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/favicon.ico", nil)
	w := httptest.NewRecorder()
	env.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}

	if got, want := len(body), 15406; got != want {
		t.Errorf("body length=%d, want=%d", got, want)
	}
}
