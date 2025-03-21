package main

import (
	"fmt"
	"net/http"
	"time"

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
	images, err := ac.queries.RecentImages(r.Context(), 10)
	if err != nil {
		panic(err)
	}

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
	if r.FormValue("preview") == fmt.Sprint(true) {
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
		NoteID:    id,
		Body:      r.FormValue("body"),
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		panic(err)
	}

	http.Redirect(w, r, "../note/"+id, http.StatusSeeOther)
}
