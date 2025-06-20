package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/internal/db"
	"github.com/codahale/yellhole-go/internal/markdown"
	"github.com/gorilla/feeds"
	"github.com/valyala/bytebufferpool"
)

func handleHomePage(queries *db.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		n, err := strconv.ParseInt(r.FormValue("n"), 10, 8)
		if err != nil {
			n = 10
		}

		var notes []db.Note
		noteID := r.FormValue("id")
		if noteID == "" {
			notes, err = queries.RecentNotes(r.Context(), n)
			if err != nil {
				return fmt.Errorf("failed to retrieve recent notes: %w", err)
			}
		} else {
			notes, err = queries.RecentNotesOlderThan(r.Context(), noteID, n)
			if err != nil {
				return fmt.Errorf("failed to retrieve recent notes older than note %q: %w", noteID, err)
			}
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			return fmt.Errorf("failed to retrieve weeks with notes: %w", err)
		}

		return htmlResponse(w, t, "feed.gohtml", &feedPage{false, notes, weeks})
	}
}

func handleWeekPage(queries *db.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		start, err := time.ParseInLocation("2006-01-02", r.PathValue("start"), time.Local)
		if err != nil {
			http.NotFound(w, r)
			return nil //nolint:nilerr // the error is handled here
		}
		end := start.AddDate(0, 0, 7)

		n, err := strconv.ParseInt(r.FormValue("n"), 10, 8)
		if err != nil {
			n = 10
		}

		var notes []db.Note
		noteID := r.FormValue("id")
		if noteID == "" {
			notes, err = queries.NotesByDate(r.Context(), start, end, n)
			if err != nil {
				return fmt.Errorf("failed to retrieve notes by date: %w", err)
			}
		} else {
			notes, err = queries.NotesByDateOlderThan(r.Context(), start, end, noteID, n)
			if err != nil {
				return fmt.Errorf("failed to retrieve notes by date older than note %q: %w", noteID, err)
			}
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			return fmt.Errorf("failed to retrieve weeks with notes for week page: %w", err)
		}

		return htmlResponse(w, t, "feed.gohtml", &feedPage{false, notes, weeks})
	}
}

func handleNotePage(queries *db.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		note, err := queries.NoteByID(r.Context(), r.PathValue("id"))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return nil
			}
			return fmt.Errorf("failed to retrieve note by ID: %w", err)
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			return fmt.Errorf("failed to retrieve weeks with notes for note page: %w", err)
		}

		return htmlResponse(w, t, "feed.gohtml", &feedPage{true, []db.Note{note}, weeks})
	}
}

func handleAtomFeed(queries *db.Queries, author, title, description string, baseURL *url.URL) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		notes, err := queries.RecentNotes(r.Context(), 20)
		if err != nil {
			return fmt.Errorf("failed to retrieve recent notes for atom feed: %w", err)
		}

		feed := feeds.Feed{
			Title:       title,
			Link:        &feeds.Link{Href: baseURL.String()},
			Description: description,
			Author:      &feeds.Author{Name: author},
		}

		if len(notes) > 0 {
			feed.Updated = notes[0].CreatedAt
		}

		for _, note := range notes {
			html, err := markdown.HTML(note.Body)
			if err != nil {
				return fmt.Errorf("failed to convert markdown to HTML for note %s: %w", note.NoteID, err)
			}

			noteURL := baseURL.JoinPath("note", note.NoteID).String()
			feed.Items = append(feed.Items, &feeds.Item{
				Id:      note.NoteID,
				Title:   note.NoteID,
				Link:    &feeds.Link{Href: noteURL},
				Content: string(html),
				Created: note.CreatedAt,
			})
		}

		b := bytebufferpool.Get()
		defer bytebufferpool.Put(b)

		if err := feeds.WriteXML(&feeds.Atom{Feed: &feed}, b); err != nil {
			return fmt.Errorf("failed to write XML for atom feed: %w", err)
		}

		w.Header().Set("Content-Type", "application/atom+xml")
		_, err = w.Write(b.B)
		if err != nil {
			return fmt.Errorf("failed to write atom feed response: %w", err)
		}
		return nil
	}
}

type feedPage struct {
	Single bool
	Notes  []db.Note
	Weeks  []db.WeeksWithNotesRow
}

func (p *feedPage) LastNoteID() string {
	if len(p.Notes) == 0 {
		return ""
	}
	return p.Notes[len(p.Notes)-1].NoteID
}
