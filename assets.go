package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

var (
	//go:embed assets
	assetsFS embed.FS
)

func loadAssets() (paths []string, hashes map[string]string, handler http.Handler, err error) {
	assetsDir, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return
	}

	handler = cacheControl(http.FileServerFS(assetsDir), "max-age=604800")
	hashes = make(map[string]string)
	err = fs.WalkDir(assetsDir, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		b, err := fs.ReadFile(assetsDir, p)
		if err != nil {
			return err
		}

		paths = append(paths, p)
		hashes[p] = fmt.Sprintf("sha256:%x", sha256.Sum256(b))

		return nil
	})
	return
}

func cacheControl(h http.Handler, cacheControl string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cache-control", cacheControl)
		h.ServeHTTP(w, r)
	})
}
