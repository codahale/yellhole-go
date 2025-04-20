package main

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"path"
)

type assetController struct {
	assets fs.FS
	paths  []string
	hashes map[string]string
	http.Handler
}

func newAssetController(assetsDir fs.FS) (*assetController, error) {
	controller := &assetController{assetsDir, nil, make(map[string]string), http.FileServerFS(assetsDir)}

	if err := fs.WalkDir(assetsDir, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		b, err := fs.ReadFile(assetsDir, p)
		if err != nil {
			return err
		}

		controller.paths = append(controller.paths, p)
		controller.hashes[p] = fmt.Sprintf("sha256:%x", sha256.Sum256(b))

		return nil
	}); err != nil {
		return nil, err
	}

	return controller, nil
}

func (ac *assetController) assetPaths() []string {
	return ac.paths
}

func (ac *assetController) assetHash(elem ...string) string {
	return ac.hashes[path.Join(elem...)]
}
