package main

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
)

//go:embed templates
var templatesDir embed.FS

type templateSet struct {
	config    *config
	templates map[string]*template.Template
}

func newTemplateSet(config *config, assets *assetController) (*templateSet, error) {
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
		"rfc3339": func(t64 int64) string {
			return time.Unix(t64, 0).Format(time.RFC3339)
		},
		"localTime": func(t64 int64) string {
			return time.Unix(t64, 0).Local().String()
		},
		"assetHash": assets.assetHash,
		"buildTag": func() string {
			return buildTag
		},
		"url": func(elem ...string) *url.URL {
			return ts.config.BaseURL.JoinPath(elem...)
		},
	}

	if err := fs.WalkDir(templatesDir, "templates", func(p string, d fs.DirEntry, err error) error {
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
	w.Header().Set("content-type", "text/html")
	if err := t.ExecuteTemplate(w, "base.html", data); err != nil {
		panic(err)
	}
}
