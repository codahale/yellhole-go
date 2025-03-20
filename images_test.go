package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeFeedImage(t *testing.T) {
	env := newTestApp(t)

	if err := os.WriteFile(filepath.Join(env.tempDir, "images", "feed", "b5621adf-c26c-4a3d-9793-5bb492afdab6.png"), []byte("feed"), 0666); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/images/feed/b5621adf-c26c-4a3d-9793-5bb492afdab6.png", nil)
	w := httptest.NewRecorder()
	env.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}

	if got, want := string(body), "feed"; got != want {
		t.Errorf("status=%q, want=%q", got, want)
	}
}

func TestServeThumbImage(t *testing.T) {
	env := newTestApp(t)

	if err := os.WriteFile(filepath.Join(env.tempDir, "images", "thumb", "b5621adf-c26c-4a3d-9793-5bb492afdab6.png"), []byte("thumb"), 0666); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/images/thumb/b5621adf-c26c-4a3d-9793-5bb492afdab6.png", nil)
	w := httptest.NewRecorder()
	env.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}

	if got, want := string(body), "thumb"; got != want {
		t.Errorf("status=%q, want=%q", got, want)
	}
}
