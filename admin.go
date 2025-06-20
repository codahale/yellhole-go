package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/internal/db"
	"github.com/google/uuid"
)

// handleAdminPage renders the admin note creation page.
func handleAdminPage(queries *db.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Look up the most recent 10 images, if any.
		images, err := queries.RecentImages(r.Context(), 10)
		if err != nil {
			return fmt.Errorf("failed to retrieve recent images: %w", err)
		}

		// Render the admin page.
		return htmlResponse(w, t, "new.gohtml", images)
	}
}

// handleNewNote creates new notes or displays them as a preview.
func handleNewNote(queries *db.Queries, t *template.Template, baseURL *url.URL) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		body := r.FormValue("body")

		// If ?preview=true, render the note as it would appear if created.
		if preview, err := strconv.ParseBool(r.FormValue("preview")); preview && err == nil {
			return htmlResponse(w, t, "preview.gohtml", body)
		}

		// Otherwise, create a new note and redirect to it.
		id := uuid.New().String()
		if err := queries.CreateNote(r.Context(), id, body, time.Now()); err != nil {
			return fmt.Errorf("failed to create new note: %w", err)
		}

		// Redirect to the new note.
		http.Redirect(w, r, baseURL.JoinPath("note", id).String(), http.StatusSeeOther)
		return nil
	}
}
