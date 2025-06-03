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
	t.Parallel()

	app := newTestApp(t)

	imageFilename := "b5621adf-c26c-4a3d-9793-5bb492afdab6.png"

	if err := os.WriteFile(filepath.Join(app.tempDir, "images", "feed", imageFilename), []byte("feed"), 0666); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/images/feed/"+imageFilename, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := string(body), "feed"; got != want {
		t.Errorf("body = %q, want = %q", got, want)
	}
}

func TestServeThumbImage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	imageFilename := "b5621adf-c26c-4a3d-9793-5bb492afdab6.png"

	if err := os.WriteFile(filepath.Join(app.tempDir, "images", "thumb", imageFilename), []byte("thumb"), 0666); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/images/thumb/"+imageFilename, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := string(body), "thumb"; got != want {
		t.Errorf("body = %q, want = %q", got, want)
	}
}
