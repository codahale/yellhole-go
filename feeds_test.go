package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
)

func TestFeedsHomePageEmpty(t *testing.T) {
	env := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	env.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}
}

func TestFeedsHomePageNote(t *testing.T) {
	env := newTestApp(t)

	if err := env.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    uuid.NewString(),
		Body:      "It's a *test*.",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	env.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}

	if want := "It&rsquo;s a <em>test</em>."; !strings.Contains(string(body), want) {
		t.Log(string(body))
		t.Errorf("note not in body, want=%q", want)
	}
}

func TestFeedsWeeksPage(t *testing.T) {
	env := newTestApp(t)

	if err := env.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    uuid.NewString(),
		Body:      "This one's in March.",
		CreatedAt: time.Date(2025, 3, 10, 10, 2, 0, 0, time.Local),
	}); err != nil {
		t.Fatal(err)
	}

	if err := env.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    uuid.NewString(),
		Body:      "This one's in April.",
		CreatedAt: time.Date(2025, 4, 10, 10, 2, 0, 0, time.Local),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/notes/2025-03-09", nil)
	w := httptest.NewRecorder()
	env.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}

	if want := "March"; !strings.Contains(string(body), want) {
		t.Errorf("note not in body, want=%q", want)
	}

	if notWant := "April"; strings.Contains(string(body), notWant) {
		t.Errorf("note in body, not want=%q", notWant)
	}
}
