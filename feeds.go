package main

import (
	"encoding/xml"
	"net/http"

	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/markdown"
	"github.com/codahale/yellhole-go/view/atom"
)

type feedController struct {
	config  *config
	queries *db.Queries
}

func newFeedController(config *config, queries *db.Queries) *feedController {
	return &feedController{config, queries}
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
