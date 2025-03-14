package view

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"strconv"
	"time"

	"github.com/codahale/yellhole-go/markdown"
)

var (
	//go:embed *.html
	files          embed.FS
	buildTimestamp string // injected via ldflags, must be uninitialized
	funcs          = template.FuncMap{
		"markdownHTML": func(s string) template.HTML {
			v, err := markdown.HTML(s)
			if err != nil {
				panic(err)
			}
			return v
		},
		"buildTimestamp": func() string {
			return buildTimestamp
		},
		"currentYear": func() int {
			return time.Now().Local().Year()
		},
	}
	tmpls = make(map[string]*template.Template)
)

func init() {
	if buildTimestamp == "" {
		buildTimestamp = strconv.FormatInt(time.Now().Unix(), 10)
	}
	log.Printf("build timestamp: %q", buildTimestamp)

	dir, err := fs.ReadDir(files, ".")
	if err != nil {
		panic(err)
	}

	// This is a lot of hassle to accomplish a single level of nested layouts.
	for _, f := range dir {
		if f.IsDir() || f.Name() == "base.html" {
			continue
		}

		t, err := template.New(f.Name()).Funcs(funcs).ParseFS(files, f.Name(), "base.html")
		if err != nil {
			panic(err)
		}

		tmpls[f.Name()] = t
	}
}

func Render(w io.Writer, name string, data any) error {
	t, ok := tmpls[name]
	if !ok {
		return fmt.Errorf("unknown template %q", name)
	}
	return t.ExecuteTemplate(w, "base.html", data)
}
