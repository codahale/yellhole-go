package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/valyala/bytebufferpool"
)

var (
	//go:embed templates
	templatesFS embed.FS
)

type templateSet struct {
	templates map[string]*template.Template
}

func newTemplateSet(author, title, description string, baseURL *url.URL, assetHashes map[string]string) (*templateSet, error) {
	templates := make(map[string]*template.Template)
	funcs := template.FuncMap{
		"assetHash": func(elem ...string) (string, error) {
			p := path.Join(elem...)
			hash, ok := assetHashes[p]
			if !ok {
				return "", fmt.Errorf("unknown asset: %q", p)
			}
			return hash, nil

		},
		"author": func() string {
			return author
		},
		"buildTag": func() string {
			return buildTag
		},
		"description": func() string {
			return description
		},
		"host": func() string {
			return baseURL.Host
		},
		"markdownHTML":   markdownHTML,
		"markdownText":   markdownText,
		"markdownImages": markdownImages,
		"now": func() time.Time {
			return time.Now()
		},
		"title": func() string {
			return title
		},
		"url": func(elem ...string) template.URL {
			return template.URL(baseURL.JoinPath(elem...).String())
		},
	}

	templatesDir, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}

	// Parse the base template.
	base, err := template.New("base").Funcs(funcs).ParseFS(templatesDir, "base.gohtml")
	if err != nil {
		return nil, err
	}

	if err := fs.WalkDir(templatesDir, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		// Convert templates/a/b/c.gohtml into a template parse pattern of the following:
		//
		//   templates/a/b/c.gohtml templates/a/b.gohtml templates/a.gohtml
		parsePath := []string{p}
		for dir := path.Dir(p); dir != "."; dir = path.Dir(dir) {
			parsePath = append(parsePath, dir+".gohtml")
		}

		// Parse the template in its inheritance path.
		t, err := template.Must(base.Clone()).ParseFS(templatesDir, parsePath...)
		if err != nil {
			return err
		}

		// Add the template using its relative path in the templates directory (e.g. "a/b/c.gohtml").
		templates[p] = t

		return nil
	}); err != nil {
		return nil, err
	}

	return &templateSet{templates}, nil
}

func (ts *templateSet) render(w http.ResponseWriter, name string, data any) {
	t, ok := ts.templates[name]
	if !ok {
		panic(fmt.Sprintf("unknown template: %q", name))
	}

	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	if err := t.ExecuteTemplate(b, "base.gohtml", data); err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "text/html")
	if _, err := w.Write(b.B); err != nil {
		panic(err)
	}
}

func jsonResponse(w http.ResponseWriter, v any) {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	if err := json.NewEncoder(b).Encode(v); err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "application/json")
	if _, err := w.Write(b.B); err != nil {
		panic(err)
	}
}
