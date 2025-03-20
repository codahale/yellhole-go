package main

import (
	"html"
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
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}
}

func TestFeedsHomePageNote(t *testing.T) {
	app := newTestApp(t)

	if err := app.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    uuid.NewString(),
		Body:      "It's a *test*.",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

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
	app := newTestApp(t)

	if err := app.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    uuid.NewString(),
		Body:      "This one's in March.",
		CreatedAt: time.Date(2025, 3, 10, 10, 2, 0, 0, time.Local).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	if err := app.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    uuid.NewString(),
		Body:      "This one's in April.",
		CreatedAt: time.Date(2025, 4, 10, 10, 2, 0, 0, time.Local).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/notes/2025-03-09", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

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

func TestFeedsWeeksPage404(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/notes/2025-03-09", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := http.StatusNotFound, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}
}

func TestFeedsNotePage(t *testing.T) {
	app := newTestApp(t)

	noteID := uuid.NewString()
	if err := app.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    noteID,
		Body:      "An example.",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/note/"+noteID, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}

	if want := "An example"; !strings.Contains(string(body), want) {
		t.Errorf("note not in body, want=%q", want)
	}
}

func TestFeedsNotePage404(t *testing.T) {
	app := newTestApp(t)

	noteID := uuid.NewString()

	req := httptest.NewRequest("GET", "http://example.com/note/"+noteID, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := http.StatusNotFound, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}
}

func TestFeedsAtomFeed(t *testing.T) {
	app := newTestApp(t)

	if err := app.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID:    uuid.NewString(),
		Body:      "It's a *test*.",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/atom.xml", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("status=%d, want=%d", got, want)
	}

	if want := html.EscapeString("It&rsquo;s a <em>test</em>."); !strings.Contains(string(body), want) {
		t.Log(string(body))
		t.Errorf("note not in body, want=%q", want)
	}

	if got, want := resp.Header.Get("content-type"), "application/atom+xml"; got != want {
		t.Errorf("content-type=%q, want=%q", got, want)
	}
}
