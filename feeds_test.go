package main

import (
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFeedsHomePageEmpty(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestFeedsHomePageNote(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	if err := app.queries.CreateNote(t.Context(), uuid.NewString(), "It's a *test*.", time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := string(body), "It&rsquo;s a <em>test</em>."; !strings.Contains(got, want) {
		t.Errorf("body = %q, want = /.*%s.*/", got, want)
	}
}

func TestFeedsWeeksPage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	if err := app.queries.CreateNote(t.Context(), uuid.NewString(), "This one's in March.",
		time.Date(2025, 3, 10, 10, 2, 0, 0, time.Local)); err != nil {
		t.Fatal(err)
	}

	if err := app.queries.CreateNote(t.Context(), uuid.NewString(), "This one's in April.",
		time.Date(2025, 4, 10, 10, 2, 0, 0, time.Local),
	); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/notes/2025-03-09", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := string(body), "March"; !strings.Contains(got, want) {
		t.Errorf("body = %q, want = /.*%s.*/", got, want)
	}

	if got, notWant := string(body), "April"; strings.Contains(got, notWant) {
		t.Errorf("body = %q, notWant = /.*%s.*/", got, notWant)
	}
}

func TestFeedsWeeksPage404(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/notes/2025-03-09", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestFeedsNotePage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	noteID := uuid.NewString()
	if err := app.queries.CreateNote(t.Context(), noteID, "An example.", time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/note/"+noteID, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := string(body), "An example"; !strings.Contains(got, want) {
		t.Errorf("body = %q, want = /.*%s.*/", got, want)
	}
}

func TestFeedsNotePage404(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	noteID := uuid.NewString()

	req := httptest.NewRequest("GET", "http://example.com/note/"+noteID, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestFeedsAtomFeed(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	if err := app.queries.CreateNote(t.Context(), uuid.NewString(), "It's a *test*.", time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/atom.xml", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := string(body), html.EscapeString("It&rsquo;s a <em>test</em>."); !strings.Contains(got, want) {
		t.Errorf("body = %q, want = /.*%s.*/", got, want)
	}

	if got, want := resp.Header.Get("Content-Type"), "application/atom+xml"; got != want {
		t.Errorf("resp.Header.Get(\"content-type\") = %q, want = %q", got, want)
	}
}
