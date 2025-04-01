package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
)

//go:embed public
var public embed.FS

type assetController struct {
	assets fs.FS
	paths  []string
	hashes map[string]string
	http.Handler
}

func newAssetController(root fs.FS, dir string) (*assetController, error) {
	assets, err := fs.Sub(root, dir)
	if err != nil {
		return nil, err
	}

	controller := &assetController{assets, nil, make(map[string]string), http.FileServerFS(assets)}

	if err := fs.WalkDir(assets, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		b, err := fs.ReadFile(assets, p)
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
