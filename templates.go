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

		"adminURL":          ts.adminURL,
		"assetURL":          ts.assetURL,
		"atomURL":           ts.atomURL,
		"baseURL":           ts.baseURL,
		"downloadImageURL":  ts.downloadImageURL,
		"feedImageURL":      ts.feedImageURL,
		"loginURL":          ts.loginURL,
		"loginStartURL":     ts.loginStartURL,
		"loginFinishURL":    ts.loginFinishURL,
		"newNoteURL":        ts.newNoteURL,
		"notePageURL":       ts.notePageURL,
		"registerURL":       ts.registerURL,
		"registerStartURL":  ts.registerStartURL,
		"registerFinishURL": ts.registerFinishURL,
		"thumbImageURL":     ts.thumbImageURL,
		"uploadImageURL":    ts.uploadImageURL,
		"weekPageURL":       ts.weekPageURL,
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

func (ts *templateSet) atomURL() *url.URL {
	return ts.config.BaseURL.JoinPath("atom.xml")
}

func (ts *templateSet) notePageURL(noteID string) *url.URL {
	return ts.config.BaseURL.JoinPath("note", noteID)
}

func (ts *templateSet) weekPageURL(startDate string) *url.URL {
	return ts.config.BaseURL.JoinPath("notes", startDate)
}

func (ts *templateSet) feedImageURL(imageID string) *url.URL {
	return ts.config.BaseURL.JoinPath("images", "feed", imageID+".png")
}

func (ts *templateSet) thumbImageURL(imageID string) *url.URL {
	return ts.config.BaseURL.JoinPath("images", "thumb", imageID+".png")
}

func (ts *templateSet) newNoteURL() *url.URL {
	return ts.config.BaseURL.JoinPath("admin", "new")
}

func (ts *templateSet) uploadImageURL() *url.URL {
	return ts.config.BaseURL.JoinPath("admin", "images", "upload")
}

func (ts *templateSet) downloadImageURL() *url.URL {
	return ts.config.BaseURL.JoinPath("admin", "images", "download")
}

func (ts *templateSet) baseURL() *url.URL {
	return ts.config.BaseURL
}

func (ts *templateSet) adminURL() *url.URL {
	return ts.config.BaseURL.JoinPath("admin")
}

func (ts *templateSet) loginURL() *url.URL {
	return ts.config.BaseURL.JoinPath("login")
}

func (ts *templateSet) loginStartURL() *url.URL {
	return ts.config.BaseURL.JoinPath("login", "start")
}

func (ts *templateSet) loginFinishURL() *url.URL {
	return ts.config.BaseURL.JoinPath("login", "finish")
}

func (ts *templateSet) registerURL() *url.URL {
	return ts.config.BaseURL.JoinPath("register")
}

func (ts *templateSet) registerStartURL() *url.URL {
	return ts.config.BaseURL.JoinPath("register", "start")
}

func (ts *templateSet) registerFinishURL() *url.URL {
	return ts.config.BaseURL.JoinPath("register", "finish")
}

func (ts *templateSet) assetURL(elem ...string) *url.URL {
	u := ts.config.BaseURL.JoinPath(elem...)
	q := u.Query()
	q.Add("", buildTag)
	u.RawQuery = q.Encode()
	return u
}
