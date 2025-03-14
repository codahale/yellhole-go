package main

import (
	"encoding/xml"
	"net/http"
	"strconv"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/markdown"
	"github.com/codahale/yellhole-go/view"
	"github.com/codahale/yellhole-go/view/atom"
)

type feedController struct {
	config  *config.Config
	queries *db.Queries
}

func newFeedController(config *config.Config, queries *db.Queries) *feedController {
	return &feedController{config, queries}
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

	w.Header().Set("content-type", "text/html")
	if err := view.Render(w, "feed.html", struct {
		Config *config.Config
		Notes  []db.Note
		Weeks  []db.Week
	}{
		fc.config,
		notes,
		weeks,
	}); err != nil {
		panic(err)
	}
}

func (fc *feedController) WeekPage(w http.ResponseWriter, r *http.Request) {
	// TODO get weeks for nav
	// TODO get notes for week
	http.NotFound(w, r)
}

func (fc *feedController) NotePage(w http.ResponseWriter, r *http.Request) {
	// TODO get weeks for nav
	// TODO get note
	http.NotFound(w, r)
}

func (fc *feedController) AtomFeed(w http.ResponseWriter, r *http.Request) {
	notes, err := fc.queries.RecentNotes(r.Context(), 20)
	if err != nil {
		panic(err)
	}

	feed := atom.Feed{
		Title:    fc.config.Title,
		Subtitle: fc.config.Description,
		ID:       fc.config.BaseURL.String(),
		Author: &atom.Person{
			Name: fc.config.Author,
		},
		Link: []atom.Link{{
			Href: view.AtomURL(fc.config).String(),
			Rel:  "alternate",
		}},
	}

	if len(notes) > 0 {
		feed.Updated = atom.Time(notes[0].CreatedAt)
	}

	for _, note := range notes {
		html, err := markdown.HTML(note.Body)
		if err != nil {
			panic(err)
		}

		entry := atom.Entry{
			ID:    view.NotePageURL(fc.config, note.NoteID).String(),
			Title: note.NoteID,
			Content: &atom.Text{
				Type: "html",
				Body: string(html),
			},
			Link: []atom.Link{{
				Href: view.NotePageURL(fc.config, note.NoteID).String(),
				Rel:  "alternate",
			}},
			Published: atom.Time(note.CreatedAt),
			Updated:   atom.Time(note.CreatedAt),
		}
		feed.Entry = append(feed.Entry, &entry)
	}

	w.Header().Set("content-type", atom.ContentType)
	if err := xml.NewEncoder(w).Encode(&feed); err != nil {
		panic(err)
	}
}
