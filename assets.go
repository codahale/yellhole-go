package main

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
)

func loadAssets(assetsDir fs.FS) (paths []string, hashes map[string]string, handler http.Handler, err error) {
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
