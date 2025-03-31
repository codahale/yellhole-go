package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/gorilla/feeds"
)

type feedController struct {
	config    *config
	queries   *db.Queries
	templates *templateSet
}

func newFeedController(config *config, queries *db.Queries, templates *templateSet) *feedController {
	return &feedController{config, queries, templates}
}

func (fc *feedController) homePage(w http.ResponseWriter, r *http.Request) {
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

	fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: false,
		Notes:  notes,
		Weeks:  weeks,
	})
}

func (fc *feedController) weekPage(w http.ResponseWriter, r *http.Request) {
	start, err := time.ParseInLocation("2006-01-02", r.PathValue("start"), time.Local)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	end := start.AddDate(0, 0, 7)

	notes, err := fc.queries.NotesByDate(r.Context(), start.Unix(), end.Unix())
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

	fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: false,
		Notes:  notes,
		Weeks:  weeks,
	})
}

func (fc *feedController) notePage(w http.ResponseWriter, r *http.Request) {
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

	fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: true,
		Notes:  []db.Note{note},
		Weeks:  weeks,
	})
}

func (fc *feedController) atomFeed(w http.ResponseWriter, r *http.Request) {
	notes, err := fc.queries.RecentNotes(r.Context(), 20)
	if err != nil {
		panic(err)
	}

	feed := feeds.Feed{
		Title:       fc.config.Title,
		Link:        &feeds.Link{Href: fc.config.BaseURL.String()},
		Description: fc.config.Description,
		Author:      &feeds.Author{Name: fc.config.Author},
	}

	if len(notes) > 0 {
		feed.Updated = time.Unix(notes[0].CreatedAt, 0)
	}

	for _, note := range notes {
		html, err := markdownHTML(note.Body)
		if err != nil {
			panic(err)
		}

		noteURL := fc.config.BaseURL.JoinPath("note", note.NoteID).String()
		feed.Items = append(feed.Items, &feeds.Item{
			Id:      note.NoteID,
			Title:   note.NoteID,
			Link:    &feeds.Link{Href: noteURL},
			Content: string(html),
			Created: time.Unix(note.CreatedAt, 0),
		})
	}

	w.Header().Set("content-type", "application/atom+xml")
	atom, err := feed.ToAtom()
	if err != nil {
		panic(err)
	}
	if _, err := w.Write([]byte(atom)); err != nil {
		panic(err)
	}
}

type feedPage struct {
	Config *config
	Single bool
	Notes  []db.Note
	Weeks  []db.Week
}
