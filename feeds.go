package main

import (
	"database/sql"
	"errors"
	db2 "github.com/codahale/yellhole-go/internal/db"
	"github.com/codahale/yellhole-go/internal/markdown"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/feeds"
	"github.com/valyala/bytebufferpool"
)

func handleHomePage(queries *db2.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		n, err := strconv.ParseInt(r.FormValue("n"), 10, 8)
		if err != nil {
			n = 10
		}

		notes, err := queries.RecentNotes(r.Context(), n)
		if err != nil {
			return err
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			return err
		}

		return htmlResponse(w, t, "feed.gohtml", feedPage{false, notes, weeks})
	}
}

func handleWeekPage(queries *db2.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		start, err := time.ParseInLocation("2006-01-02", r.PathValue("start"), time.Local)
		if err != nil {
			http.NotFound(w, r)
			return nil
		}
		end := start.AddDate(0, 0, 7)

		notes, err := queries.NotesByDate(r.Context(), start, end)
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			http.NotFound(w, r)
			return nil
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			return err
		}

		return htmlResponse(w, t, "feed.gohtml", feedPage{false, notes, weeks})
	}
}

func handleNotePage(queries *db2.Queries, t *template.Template) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		note, err := queries.NoteByID(r.Context(), r.PathValue("id"))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return nil
			}
			return err
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			return err
		}

		return htmlResponse(w, t, "feed.gohtml", feedPage{true, []db2.Note{note}, weeks})
	}
}

func handleAtomFeed(queries *db2.Queries, author, title, description string, baseURL *url.URL) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		notes, err := queries.RecentNotes(r.Context(), 20)
		if err != nil {
			return err
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
				return err
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
			return err
		}

		w.Header().Set("content-type", "application/atom+xml")
		_, err = w.Write(b.B)
		return err
	}
}

type feedPage struct {
	Single bool
	Notes  []db2.Note
	Weeks  []db2.WeeksWithNotesRow
}
