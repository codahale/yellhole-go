package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"time"

	"github.com/valyala/bytebufferpool"
)

type templateSet struct {
	templates map[string]*template.Template
}

func newTemplateSet(config *config, templatesDir fs.FS, assets *assetController) (*templateSet, error) {
	ts := &templateSet{
		templates: make(map[string]*template.Template),
	}

	funcs := template.FuncMap{
		"assetHash": assets.assetHash,
		"author": func() string {
			return config.Author
		},
		"buildTag": func() string {
			return buildTag
		},
		"description": func() string {
			return config.Description
		},
		"host": func() string {
			return config.BaseURL.Host
		},
		"markdownHTML":   markdownHTML,
		"markdownText":   markdownText,
		"markdownImages": markdownImages,
		"now": func() time.Time {
			return time.Now()
		},
		"title": func() string {
			return config.Title
		},
		"url": func(elem ...string) template.URL {
			return template.URL(config.BaseURL.JoinPath(elem...).String())
		},
	}

	if err := fs.WalkDir(templatesDir, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		// Convert templates/a/b/c.html into a template parse pattern of the following:
		//
		//   templates/a/b/c.html templates/a/b.html templates/a.html templates/base.html
		parsePath := []string{p}
		for dir := path.Dir(p); dir != "."; dir = path.Dir(dir) {
			parsePath = append(parsePath, dir+".html")
		}
		parsePath = append(parsePath, "base.html")

		// Parse the template in its inheritance path.
		t, err := template.New(d.Name()).Funcs(funcs).ParseFS(templatesDir, parsePath...)
		if err != nil {
			return err
		}

		// Add the template using its relative path in the templates directory (e.g. "a/b/c.html").
		ts.templates[p] = t

		return nil
	}); err != nil {
		return nil, err
	}
	return ts, nil
}

func (ts *templateSet) render(w http.ResponseWriter, name string, data any) {
	t, ok := ts.templates[name]
	if !ok {
		panic(fmt.Sprintf("unknown template: %q", name))
	}

	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	if err := t.ExecuteTemplate(b, "base.html", data); err != nil {
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
