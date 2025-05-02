package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAdminPage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	sessionID := uuid.NewString()
	if err := app.app.queries.CreateSession(t.Context(), sessionID, time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/admin", nil)
	req.AddCookie(&http.Cookie{
		Name:  "sessionID",
		Value: sessionID,
	})

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		t.Fatal(err)
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestAdminNotePreview(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	sessionID := uuid.NewString()
	if err := app.app.queries.CreateSession(t.Context(), sessionID, time.Now()); err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	body, _ := mw.CreateFormField("body")
	_, _ = body.Write([]byte("This is _interesting_."))
	preview, _ := mw.CreateFormField("preview")
	_, _ = preview.Write([]byte("true"))
	_ = mw.Close()

	req := httptest.NewRequest("POST", "http://example.com/admin/new", &b)
	req.Header.Set("content-type", mw.FormDataContentType())
	// req.Header.Set("sec-fetch-site", "none")
	req.AddCookie(&http.Cookie{
		Name:  "sessionID",
		Value: sessionID,
	})

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()
	response, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := string(response), "This is <em>interesting</em>."; !strings.Contains(got, want) {
		t.Errorf(`response = %v, want = /%v/`, got, want)
	}
}

func TestAdminNoteCreate(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	sessionID := uuid.NewString()
	if err := app.app.queries.CreateSession(t.Context(), sessionID, time.Now()); err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	body, _ := mw.CreateFormField("body")
	_, _ = body.Write([]byte("This is _interesting_."))
	_ = mw.Close()

	req := httptest.NewRequest("POST", "http://example.com/admin/new", &b)
	req.Header.Set("content-type", mw.FormDataContentType())
	req.Header.Set("sec-fetch-site", "none")
	req.AddCookie(&http.Cookie{
		Name:  "sessionID",
		Value: sessionID,
	})

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusSeeOther; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	notes, err := app.app.queries.RecentNotes(t.Context(), 10)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(notes), 1; got != want {
		t.Errorf(`len(notes) = %v, want = %v`, got, want)
	}

	if got, want := notes[0].Body, "This is _interesting_."; got != want {
		t.Errorf(`notes[0].Body = %v, want = %v`, got, want)
	}

	if got, want := resp.Header.Get("location"), "http://example.com/note/"+notes[0].NoteID; got != want {
		t.Errorf(`resp.Header.Get("location") = %v, want = %v`, got, want)
	}
}
