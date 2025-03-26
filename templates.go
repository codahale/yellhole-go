package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed templates
var templatesDir embed.FS

type templateSet struct {
	templates map[string]*template.Template
}

func newTemplateSet(assets *assetController) (*templateSet, error) {
	templates := make(map[string]*template.Template)

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
		"assetHash": func(elem ...string) string {
			return assets.AssetHash(elem...)
		},
	}

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
