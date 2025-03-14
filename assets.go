package main

import (
	"io/fs"
	"net/http"
)

type assetController struct {
	assets fs.FS
	http.Handler
}

func newAssetController(root fs.FS, dir string) assetController {
	assets, err := fs.Sub(root, dir)
	if err != nil {
		panic(err)
	}

	handler := http.FileServerFS(assets)
	return assetController{assets, handler}
}

func (ac *assetController) AssetPaths() []string {
	var paths []string
	if err := fs.WalkDir(ac.assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			paths = append(paths, path)
		}

		return nil
	}); err != nil {
		panic(err)
	}
	return paths
}
