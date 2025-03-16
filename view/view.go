package view

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"time"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/markdown"
)

var (
	//go:embed *.html
	files          embed.FS
	buildTimestamp string // injected via ldflags, must be uninitialized
	funcs          = template.FuncMap{
		"markdownHTML":   markdown.HTML,
		"markdownText":   markdown.Text,
		"markdownImages": markdown.Images,
		"buildTimestamp": func() string {
			return buildTimestamp
		},
		"currentYear": func() int {
			return time.Now().Local().Year()
		},
		"rfc3339": func(t *time.Time) string {
			return t.UTC().Format(time.RFC3339)
		},
		"atomURL":     AtomURL,
		"weekPageURL": WeekPageURL,
		"notePageURL": NotePageURL,
		"feedImageURL": func(c *config.Config, imageID string) *url.URL {
			return c.BaseURL.JoinPath("images", "feed", fmt.Sprintf("%s.png", imageID))
		},
		"thumbImageURL": func(c *config.Config, imageID string) *url.URL {
			return c.BaseURL.JoinPath("images", "thumb", fmt.Sprintf("%s.png", imageID))
		},
		"newNoteURL": func(c *config.Config) *url.URL {
			return c.BaseURL.JoinPath("admin", "new")
		},
		"uploadImageURL": func(c *config.Config) *url.URL {
			return c.BaseURL.JoinPath("admin", "images", "upload")
		},
		"downloadImageURL": func(c *config.Config) *url.URL {
			return c.BaseURL.JoinPath("admin", "images", "download")
		},
		"assetURL": func(c *config.Config, elem ...string) *url.URL {
			u := c.BaseURL.JoinPath(elem...)
			q := u.Query()
			q.Add("", buildTimestamp)
			u.RawQuery = q.Encode()
			return u
		},
	}
	tmpls = make(map[string]*template.Template)
)

func AtomURL(c *config.Config) *url.URL {
	return c.BaseURL.JoinPath("atom.xml")
}

func WeekPageURL(c *config.Config, startDate string) *url.URL {
	return c.BaseURL.JoinPath("notes", startDate)
}

func NotePageURL(c *config.Config, noteID string) *url.URL {
	return c.BaseURL.JoinPath("note", noteID)
}

func init() {
	dir, err := fs.ReadDir(files, ".")
	if err != nil {
		panic(err)
	}

	// This is a lot of hassle to accomplish a single level of nested layouts.
	for _, f := range dir {
		if f.IsDir() || f.Name() == "base.html" {
			continue
		}

		t, err := template.New(f.Name()).Funcs(funcs).ParseFS(files, f.Name(), "base.html")
		if err != nil {
			panic(err)
		}

		tmpls[f.Name()] = t
	}
}

func Render(w io.Writer, name string, data any) error {
	t, ok := tmpls[name]
	if !ok {
		return fmt.Errorf("unknown template %q", name)
	}
	return t.ExecuteTemplate(w, "base.html", data)
}
