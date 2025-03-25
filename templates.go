package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/markdown"
)

type templateSet struct {
	templates map[string]*template.Template
}

func newTemplateSet() (*templateSet, error) {
	templates := make(map[string]*template.Template)
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
		templates[strings.TrimPrefix(tmplPath, "templates/")] = t

		return nil
	}); err != nil {
		return nil, err
	}
	return &templateSet{templates}, nil
}

func (ts *templateSet) render(w http.ResponseWriter, name string, data any) error {
	t, ok := ts.templates[name]
	if !ok {
		return fmt.Errorf("unknown template %q", name)
	}
	w.Header().Set("content-type", "text/html")
	return t.ExecuteTemplate(w, "base.html", data)
}

func atomURL(c *config.Config) *url.URL {
	return c.BaseURL.JoinPath("atom.xml")
}

func notePageURL(c *config.Config, noteID string) *url.URL {
	return c.BaseURL.JoinPath("note", noteID)
}

var (
	//go:embed templates
	templatesDir embed.FS
	assetHashes  = new(sync.Map)
	funcs        = template.FuncMap{
		"markdownHTML":   markdown.HTML,
		"markdownText":   markdown.Text,
		"markdownImages": markdown.Images,
		"currentYear": func() int {
			return time.Now().Local().Year()
		},
		"rfc3339": func(t64 int64) string {
			return time.Unix(t64, 0).Format(time.RFC3339)
		},
		"localTime": func(t64 int64) string {
			return time.Unix(t64, 0).Local().String()
		},
		"atomURL": atomURL,
		"weekPageURL": func(c *config.Config, startDate string) *url.URL {
			return c.BaseURL.JoinPath("notes", startDate)
		},
		"notePageURL": notePageURL,
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
			q.Add("", buildTag)
			u.RawQuery = q.Encode()
			return u
		},
		"assetHash": func(elem ...string) (string, error) {
			assetPath := path.Join("public", path.Join(elem...))
			hash, ok := assetHashes.Load(assetPath)
			if ok {
				return hash.(string), nil
			}

			f, err := public.Open(assetPath)
			if err != nil {
				return "", err
			}
			defer func() {
				_ = f.Close()
			}()

			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				return "", err
			}

			hash = "sha256:" + hex.EncodeToString(h.Sum(nil))
			assetHashes.Store(assetPath, hash)

			return hash.(string), nil
		},
	}
)
