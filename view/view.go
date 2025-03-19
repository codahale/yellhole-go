package view

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/markdown"
)

var (
	//go:embed templates
	templatesDir   embed.FS
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
		"rfc3339": func(t64 int64) string {
			return time.Unix(t64, 0).Format(time.RFC3339)
		},
		"localTime": func(t64 int64) string {
			return time.Unix(t64, 0).Local().String()
		},
		"atomURL":     AtomURL,
		"weekPageURL": WeekPageURL,
		"notePageURL": NotePageURL,
		"feedImageURL": func(c *config.Config, imageID string) *url.URL {
			return c.BaseURL.JoinPath("images", "feed", imageID+".png")
		},
		"thumbImageURL": func(c *config.Config, imageID string) *url.URL {
			return c.BaseURL.JoinPath("images", "thumb", imageID+".png")
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
	if err := fs.WalkDir(templatesDir, "templates", func(tmplPath string, d fs.DirEntry, err error) error {
		// Ignore directories.
		if d.IsDir() {
			return nil
		}

		// Propagate errors.
		if err != nil {
			return err
		}

		// Convert templates/a/b/c.html into a template parse pattern of the following:
		//
		//   templates/a/b/c.html templates/a/b.html templates/a.html templates/base.html
		parsePath := []string{tmplPath}
		dir := path.Dir(tmplPath)
		for dir != "templates" {
			parsePath = append(parsePath, dir+".html")
			dir = path.Dir(dir)
		}
		parsePath = append(parsePath, "templates/base.html")

		// Parse the template in its inheritance path.
		t, err := template.New(d.Name()).Funcs(funcs).ParseFS(templatesDir, parsePath...)
		if err != nil {
			return err
		}

		// Add the template using its relative path in the templates directory (e.g. "a/b/c.html").
		tmpls[strings.TrimPrefix(tmplPath, "templates/")] = t

		return nil
	}); err != nil {
		panic(err)
	}
}

func Render(w http.ResponseWriter, name string, data any) error {
	t, ok := tmpls[name]
	if !ok {
		return fmt.Errorf("unknown template %q", name)
	}
	w.Header().Set("content-type", "text/html")
	return t.ExecuteTemplate(w, "base.html", data)
}
