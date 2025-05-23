package main

import (
	"database/sql"
	"errors"
	"github.com/codahale/yellhole-go/markdown"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/gorilla/feeds"
	"github.com/valyala/bytebufferpool"
)

func handleHomePage(queries *db.Queries, t *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.ParseInt(r.FormValue("n"), 10, 8)
		if err != nil {
			n = 10
		}

		notes, err := queries.RecentNotes(r.Context(), n)
		if err != nil {
			panic(err)
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			panic(err)
		}

		htmlResponse(w, t, "feed.gohtml", feedPage{false, notes, weeks})
	})
}

func handleWeekPage(queries *db.Queries, t *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, err := time.ParseInLocation("2006-01-02", r.PathValue("start"), time.Local)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		end := start.AddDate(0, 0, 7)

		notes, err := queries.NotesByDate(r.Context(), start, end)
		if err != nil {
			panic(err)
		}

		if len(notes) == 0 {
			http.NotFound(w, r)
			return
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			panic(err)
		}

		htmlResponse(w, t, "feed.gohtml", feedPage{false, notes, weeks})
	})
}

func handleNotePage(queries *db.Queries, t *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		note, err := queries.NoteByID(r.Context(), r.PathValue("id"))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			panic(err)
		}

		weeks, err := queries.WeeksWithNotes(r.Context())
		if err != nil {
			panic(err)
		}

		htmlResponse(w, t, "feed.gohtml", feedPage{true, []db.Note{note}, weeks})
	})
}

func handleAtomFeed(queries *db.Queries, author, title, description string, baseURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notes, err := queries.RecentNotes(r.Context(), 20)
		if err != nil {
			panic(err)
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
				panic(err)
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
			panic(err)
		}

		w.Header().Set("content-type", "application/atom+xml")
		if _, err := w.Write(b.B); err != nil {
			panic(err)
		}
	})
}

type feedPage struct {
	Single bool
	Notes  []db.Note
	Weeks  []db.WeeksWithNotesRow
}
