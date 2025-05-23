package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
)

func handleAdminPage(queries *db.Queries, t *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		images, err := queries.RecentImages(r.Context(), 10)
		if err != nil {
			panic(err)
		}

		htmlResponse(w, t, "new.gohtml", images)
	})
}

func handleNewNote(queries *db.Queries, t *template.Template, baseURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("preview") == fmt.Sprint(true) {
			htmlResponse(w, t, "preview.gohtml", r.FormValue("body"))
			return
		}

		id := uuid.New().String()
		if err := queries.CreateNote(r.Context(), id, r.FormValue("body"), time.Now()); err != nil {
			panic(err)
		}

		http.Redirect(w, r, baseURL.JoinPath("note", id).String(), http.StatusSeeOther)
	})
}
