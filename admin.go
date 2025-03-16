package main

import (
	"fmt"
	"net/http"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/view"
	"github.com/google/uuid"
)

type adminController struct {
	config  *config.Config
	queries *db.Queries
}

func newAdminController(config *config.Config, queries *db.Queries) *adminController {
	return &adminController{config, queries}
}

func (ac *adminController) AdminPage(w http.ResponseWriter, r *http.Request) {
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}

	if !auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("login").String(), http.StatusSeeOther)
		return
	}

	images, err := ac.queries.RecentImages(r.Context(), 10)
	if err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "text/html")
	if err := view.Render(w, "new.html", struct {
		Config *config.Config
		Images []db.Image
	}{
		ac.config,
		images,
	}); err != nil {
		panic(err)
	}
}

func (ac *adminController) NewNote(w http.ResponseWriter, r *http.Request) {
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}

	if !auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("login").String(), http.StatusSeeOther)
		return
	}

	if r.FormValue("preview") == fmt.Sprint(true) {
		w.Header().Set("content-type", "text/html")
		if err := view.Render(w, "preview.html", struct {
			Config *config.Config
			Body   string
		}{
			ac.config,
			r.FormValue("body"),
		}); err != nil {
			panic(err)
		}
		return
	}

	id := uuid.New().String()
	if err := ac.queries.CreateNote(r.Context(), db.CreateNoteParams{
		NoteID: id,
		Body:   r.FormValue("body"),
	}); err != nil {
		panic(err)
	}

	http.Redirect(w, r, fmt.Sprintf("../note/%s", id), http.StatusSeeOther)
}
