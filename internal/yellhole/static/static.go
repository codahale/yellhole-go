package static

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed assets/*
var assets embed.FS

func Register(mux *http.ServeMux) {
	assets, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}

	handler := http.FileServerFS(assets)
	if err := fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		mux.Handle(fmt.Sprintf("GET /%s", path), handler)
		return nil
	}); err != nil {
		panic(err)
	}
}
