package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/valyala/bytebufferpool"
)

type templateSet struct {
	config    *config
	templates map[string]*template.Template
}

func newTemplateSet(config *config, templates fs.FS, assets *assetController) (*templateSet, error) {
	ts := &templateSet{
		config:    config,
		templates: make(map[string]*template.Template),
	}

	funcs := template.FuncMap{
		"markdownHTML":   markdownHTML,
		"markdownText":   markdownText,
		"markdownImages": markdownImages,
		"currentYear": func() int {
			return time.Now().Local().Year()
		},
		"rfc3339": func(t time.Time) string {
			return t.UTC().Format(time.RFC3339)
		},
		"localTime": func(t time.Time) string {
			return t.Local().String()
		},
		"assetHash": assets.assetHash,
		"buildTag": func() string {
			return buildTag
		},
		"url": func(elem ...string) *url.URL {
			return ts.config.BaseURL.JoinPath(elem...)
		},
	}

	if err := fs.WalkDir(templates, "templates", func(p string, d fs.DirEntry, err error) error {
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
		parsePath := []string{p}
		for dir := path.Dir(p); dir != "templates"; dir = path.Dir(dir) {
			parsePath = append(parsePath, dir+".html")
		}
		parsePath = append(parsePath, "templates/base.html")

		// Parse the template in its inheritance path.
		t, err := template.New(d.Name()).Funcs(funcs).ParseFS(templatesDir, parsePath...)
		if err != nil {
			return err
		}

		// Add the template using its relative path in the templates directory (e.g. "a/b/c.html").
		ts.templates[strings.TrimPrefix(p, "templates/")] = t

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
