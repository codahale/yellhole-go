package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
)

func TestFeedsHome(t *testing.T) {
	env := newTestApp(t)
	defer env.teardown()

	if err := env.app.queries.CreateNote(t.Context(), db.CreateNoteParams{
		NoteID: uuid.NewString(),
		Body:   "It's a *test*.",
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

	if want := "It's a <em>test</em>."; strings.Contains(string(body), want) {
		t.Errorf("note not in body, want=%q", want)
	}
}
