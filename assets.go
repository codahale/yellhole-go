package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
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

	controller := &assetController{
		assets:  assets,
		hashes:  make(map[string]string),
		Handler: http.FileServerFS(assets),
	}

	if err := fs.WalkDir(assets, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		controller.paths = append(controller.paths, p)

		b, err := public.ReadFile(path.Join("public", p))
		if err != nil {
			return err
		}

		h := sha256.New()
		h.Write(b)
		controller.hashes[p] = "sha256:" + hex.EncodeToString(h.Sum(nil))

		return nil
	}); err != nil {
		return nil, err
	}

	return controller, nil
}

func (ac *assetController) AssetPaths() []string {
	return ac.paths
}

func (ac *assetController) AssetHash(elem ...string) string {
	return ac.hashes[path.Join(elem...)]
}
