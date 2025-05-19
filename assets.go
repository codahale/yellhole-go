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

	handler = http.FileServerFS(assetsDir)
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
