package main

import (
	"database/sql"
	"encoding/xml"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/db"
)

type feedController struct {
	config    *config
	queries   *db.Queries
	templates *templateSet
}

func newFeedController(config *config, queries *db.Queries, templates *templateSet) *feedController {
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

	fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: false,
		Notes:  notes,
		Weeks:  weeks,
	})
}

func (fc *feedController) WeekPage(w http.ResponseWriter, r *http.Request) {
	start, err := time.ParseInLocation("2006-01-02", r.PathValue("start"), time.Local)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	notes, err := fc.queries.NotesByDate(r.Context(), start.Unix(), start.AddDate(0, 0, 7).Unix())
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

	fc.templates.render(w, "feed.html", feedPage{
		Config: fc.config,
		Single: true,
		Notes:  []db.Note{note},
		Weeks:  weeks,
	})
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
			Href: fc.config.BaseURL.JoinPath("atom.xml").String(),
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

		noteURL := fc.config.BaseURL.JoinPath("note", note.NoteID).String()
		entry := atomEntry{
			ID:    noteURL,
			Title: note.NoteID,
			Content: &atomText{
				Type: "html",
				Body: string(html),
			},
			Link: []atomLink{{
				Href: noteURL,
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
	Config *config
	Single bool
	Notes  []db.Note
	Weeks  []db.Week
}

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from encoding/xml/read_test.go.

const (
	atomContentType = "application/atom+xml"
)

type atomFeed struct {
	XMLName  xml.Name     `xml:"http://www.w3.org/2005/Atom feed"`
	Title    string       `xml:"title"`
	Subtitle string       `xml:"subtitle"`
	ID       string       `xml:"id"`
	Link     []atomLink   `xml:"link"`
	Updated  atomTimeStr  `xml:"updated"`
	Author   *atomPerson  `xml:"author"`
	Entry    []*atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string      `xml:"title"`
	ID        string      `xml:"id"`
	Link      []atomLink  `xml:"link"`
	Published atomTimeStr `xml:"published"`
	Updated   atomTimeStr `xml:"updated"`
	Author    *atomPerson `xml:"author"`
	Summary   *atomText   `xml:"summary"`
	Content   *atomText   `xml:"content"`
}

type atomLink struct {
	Rel      string `xml:"rel,attr,omitempty"`
	Href     string `xml:"href,attr"`
	Type     string `xml:"type,attr,omitempty"`
	HrefLang string `xml:"hreflang,attr,omitempty"`
	Title    string `xml:"title,attr,omitempty"`
	Length   uint   `xml:"length,attr,omitempty"`
}

type atomPerson struct {
	Name     string `xml:"name"`
	URI      string `xml:"uri,omitempty"`
	Email    string `xml:"email,omitempty"`
	InnerXML string `xml:",innerxml"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type atomTimeStr string

func atomTime(t time.Time) atomTimeStr {
	return atomTimeStr(t.Format("2006-01-02T15:04:05-07:00"))
}
