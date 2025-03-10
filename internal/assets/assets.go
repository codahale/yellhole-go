package assets

import (
	"embed"
	"fmt"
	"net/http"
)

//go:embed css js *.png *.ico *.webmanifest *.br *.gz
var assets embed.FS

func Register(mux *http.ServeMux) {
	fs := http.FileServerFS(assets)
	dir, err := assets.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for _, f := range dir {
		if f.IsDir() {
			mux.Handle(fmt.Sprintf("/%s/", f.Name()), fs)
		} else {
			mux.Handle(fmt.Sprintf("/%s", f.Name()), fs)
		}
	}
}
