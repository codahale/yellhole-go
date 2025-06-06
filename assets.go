package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"slices"
)

var (
	// assetsFS embeds all the static assets used by the app.
	//go:embed internal/assets
	assetsFS embed.FS
)

// loadAssets returns a slice of asset paths, a map of asset paths to subresource integrity hashes, an HTTP handler for
// serving assets, or an error.
func loadAssets() (paths []string, hashes map[string]string, handler http.Handler, err error) {
	// Step into the ./assets directory of the embedded files.
	assetsDir, err := fs.Sub(assetsFS, "internal/assets")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to access internal/assets: %w", err)
	}

	// Create an HTTP handler for the assets with a max-age of one week.
	handler = cacheControl(http.FileServerFS(assetsDir), "public,max-age=604800")

	// Create a map of asset paths to subresource integrity hashes.
	hashes = make(map[string]string)
	if err := fs.WalkDir(assetsDir, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		b, err := fs.ReadFile(assetsDir, p)
		if err != nil {
			return fmt.Errorf("failed to read asset %s: %w", p, err)
		}

		hashes[p] = fmt.Sprintf("sha256:%x", sha256.Sum256(b))

		return nil
	}); err != nil {
		return nil, nil, nil, err
	}

	// Create a slice of asset paths.
	paths = slices.Collect(maps.Keys(hashes))

	return paths, hashes, handler, nil
}

// cacheControl returns a wrapper handler which sets the specified Cache-Control header in the response.
func cacheControl(h http.Handler, cacheControl string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		h.ServeHTTP(w, r)
	})
}
