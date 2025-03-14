package view

import (
	"embed"
	"html/template"
	"io"

	"github.com/codahale/yellhole-go/markdown"
)

var (
	//go:embed *.html
	templateFiles embed.FS
	templates     = template.New("yellhole").Funcs(template.FuncMap{
		"markdownHTML": func(s string) template.HTML {
			v, err := markdown.HTML(s)
			if err != nil {
				panic(err)
			}
			return v
		},
	})
)

func init() {
	templates = template.Must(templates.ParseFS(templateFiles, "*.html"))
}

func Render(w io.Writer, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
