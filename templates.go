package main

import (
	"embed"
	"fmt"
	"github.com/codahale/yellhole-go/internal/markdown"
	"html/template"
	"io/fs"
	"net/url"
	"path"
	"time"

	"github.com/codahale/yellhole-go/internal/build"
)

var (
	// templatesFS embeds all the templates for the app.
	//go:embed internal/templates
	templatesFS embed.FS
)

// loadTemplates loads and parses all the embedded templates for the app.
func loadTemplates(author, title, description, lang string, baseURL *url.URL, assetHashes map[string]string) (*template.Template, error) {
	templatesDir, err := fs.Sub(templatesFS, "internal/templates")
	if err != nil {
		return nil, err
	}

	return template.New("yellhole").Funcs(template.FuncMap{
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
			return build.Tag
		},
		"description": func() string {
			return description
		},
		"host": func() string {
			return baseURL.Host
		},
		"lang": func() string {
			return lang
		},
		"markdownHTML":   markdown.HTML,
		"markdownText":   markdown.Text,
		"markdownImages": markdown.Images,
		"now": func() time.Time {
			return time.Now()
		},
		"title": func() string {
			return title
		},
		"url": func(elem ...string) template.URL {
			return template.URL(baseURL.JoinPath(elem...).String())
		},
	}).ParseFS(templatesDir, "partials/*.gohtml", "*.gohtml")
}
