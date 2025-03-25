package main

import (
	"database/sql"
	"encoding/xml"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
)

type feedController struct {
	config    *config.Config
	queries   *db.Queries
	templates *templateSet
}

func newFeedController(config *config.Config, queries *db.Queries, templates *templateSet) *feedController {
	return &feedController{config, queries, templates}
}

func (fc *feedController) HomePage(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.ParseInt(r.FormValue("n"), 10, 8)
	if err != nil {
		n = 10
	}

	notes, err := fc.queries.RecentNotes(r.Context(), n)
	if err != nil {
		panic(err)
	}

	weeks, err := fc.queries.WeeksWithNotes(r.Context())
	if err != nil {
		panic(err)
	}

	if err := fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: false,
		Notes:  notes,
		Weeks:  weeks,
	}); err != nil {
		panic(err)
	}
}

func (fc *feedController) WeekPage(w http.ResponseWriter, r *http.Request) {
	start, err := time.ParseInLocation("2006-01-02", r.PathValue("start"), time.Local)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	notes, err := fc.queries.NotesByDate(r.Context(), db.NotesByDateParams{
		Start: start.Unix(),
		End:   start.AddDate(0, 0, 7).Unix(),
	})
	if err != nil {
		panic(err)
	}

	if len(notes) == 0 {
		http.NotFound(w, r)
		return
	}

	weeks, err := fc.queries.WeeksWithNotes(r.Context())
	if err != nil {
		panic(err)
	}

	if err := fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: false,
		Notes:  notes,
		Weeks:  weeks,
	}); err != nil {
		panic(err)
	}
}

func (fc *feedController) NotePage(w http.ResponseWriter, r *http.Request) {
	note, err := fc.queries.NoteByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		panic(err)
	}

	weeks, err := fc.queries.WeeksWithNotes(r.Context())
	if err != nil {
		panic(err)
	}

	if err := fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: true,
		Notes:  []db.Note{note},
		Weeks:  weeks,
	}); err != nil {
		panic(err)
	}
}

func (fc *feedController) AtomFeed(w http.ResponseWriter, r *http.Request) {
	notes, err := fc.queries.RecentNotes(r.Context(), 20)
	if err != nil {
		panic(err)
	}

	feed := atomFeed{
		Title:    fc.config.Title,
		Subtitle: fc.config.Description,
		ID:       fc.config.BaseURL.String(),
		Author: &atomPerson{
			Name: fc.config.Author,
		},
		Link: []atomLink{{
			Href: atomURL(fc.config).String(),
			Rel:  "alternate",
		}},
	}

	if len(notes) > 0 {
		feed.Updated = atomTime(time.Unix(notes[0].CreatedAt, 0))
	}

	for _, note := range notes {
		html, err := markdownHTML(note.Body)
		if err != nil {
			panic(err)
		}

		entry := atomEntry{
			ID:    notePageURL(fc.config, note.NoteID).String(),
			Title: note.NoteID,
			Content: &atomText{
				Type: "html",
				Body: string(html),
			},
			Link: []atomLink{{
				Href: notePageURL(fc.config, note.NoteID).String(),
				Rel:  "alternate",
			}},
			Published: atomTime(time.Unix(note.CreatedAt, 0)),
			Updated:   atomTime(time.Unix(note.CreatedAt, 0)),
		}
		feed.Entry = append(feed.Entry, &entry)
	}

	w.Header().Set("content-type", atomContentType)
	if err := xml.NewEncoder(w).Encode(&feed); err != nil {
		panic(err)
	}
}

type feedPage struct {
	Config *config.Config
	Single bool
	Notes  []db.Note
	Weeks  []db.Week
}
