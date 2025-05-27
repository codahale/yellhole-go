package main

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
)

// handleAdminPage renders the admin note creation page.
func handleAdminPage(queries *db.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Look up the most recent 10 images, if any.
		images, err := queries.RecentImages(r.Context(), 10)
		if err != nil {
			return err
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
		if preview, _ := strconv.ParseBool(r.FormValue("preview")); preview {
			return htmlResponse(w, t, "preview.gohtml", body)
		}

		// Otherwise, create the new note and redirect to it.
		id := uuid.New().String()
		if err := queries.CreateNote(r.Context(), id, body, time.Now()); err != nil {
			return err
		}

		http.Redirect(w, r, baseURL.JoinPath("note", id).String(), http.StatusSeeOther)
		return nil
	}
}
