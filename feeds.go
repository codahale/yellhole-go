package main

import (
	"encoding/xml"
	"net/http"
	"strconv"

	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/markdown"
	"github.com/codahale/yellhole-go/view"
	"github.com/codahale/yellhole-go/view/atom"
)

type feedController struct {
	config  *config
	queries *db.Queries
}

func newFeedController(config *config, queries *db.Queries) *feedController {
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

	w.Header().Set("content-type", "text/html")
	if err := view.Render(w, "feed.html", struct {
		Config *config
		Notes  []db.Note
	}{
		fc.config,
		notes,
	}); err != nil {
		panic(err)
	}
	// TODO get config (title, description, base URL, etc.)
	// TODO get weeks for nav
	// TODO get current year
}

func (fc *feedController) WeekPage(w http.ResponseWriter, r *http.Request) {
	// TODO get config (title, description, base URL, etc.)
	// TODO get weeks for nav
	// TODO get current year
	// TODO get notes for week
	http.NotFound(w, r)
}

func (fc *feedController) NotePage(w http.ResponseWriter, r *http.Request) {
	// TODO get config (title, description, base URL, etc.)
	// TODO get weeks for nav
	// TODO get current year
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
			Href: fc.config.BaseURL.JoinPath("/atom.xml").String(),
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
			// TODO ID: noteURL.String(),
			Title: note.NoteID,
			Content: &atom.Text{
				Type: "html",
				Body: string(html),
			},
			Link: []atom.Link{{
				Rel: "alternate",
				// TODO Href: note.noteURL.String(),
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
