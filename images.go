package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

type imageController struct {
	config            *config.Config
	queries           *db.Queries
	root              *os.Root
	feedRoot          *os.Root
	origRoot          *os.Root
	thumbRoot         *os.Root
	feedImageHandler  http.Handler
	thumbImageHandler http.Handler
}

func newImageController(config *config.Config, dataRoot *os.Root, queries *db.Queries) (*imageController, error) {
	_ = dataRoot.Mkdir("images", 0755)
	root, err := dataRoot.OpenRoot("images")
	if err != nil {
		return nil, err
	}

	_ = root.Mkdir("feed", 0755)
	feedRoot, err := root.OpenRoot("feed")
	if err != nil {
		return nil, err
	}

	_ = root.Mkdir("original", 0755)
	origRoot, err := root.OpenRoot("original")
	if err != nil {
		return nil, err
	}

	_ = root.Mkdir("thumb", 0755)
	thumbRoot, err := root.OpenRoot("thumb")
	if err != nil {
		return nil, err
	}

	feedImageHandler := http.FileServerFS(feedRoot.FS())
	thumbImageHandler := http.FileServerFS(thumbRoot.FS())

	return &imageController{
		config,
		queries,
		root,
		feedRoot,
		origRoot,
		thumbRoot,
		feedImageHandler,
		thumbImageHandler,
	}, nil
}

func (ic *imageController) ServeFeedImage(w http.ResponseWriter, r *http.Request) {
	ic.feedImageHandler.ServeHTTP(w, r)
}

func (ic *imageController) ServeThumbImage(w http.ResponseWriter, r *http.Request) {
	ic.thumbImageHandler.ServeHTTP(w, r)
}

func (ic *imageController) DownloadImage(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	id := uuid.New()

	format, err := ic.processImage(id, resp.Body)
	if err != nil {
		panic(err)
	}

	if err := ic.queries.CreateImage(r.Context(), db.CreateImageParams{
		ImageID:  id.String(),
		Filename: url,
		Format:   format,
	}); err != nil {
		panic(err)
	}

	http.Redirect(w, r, "..", http.StatusSeeOther)
}

func (ic *imageController) UploadImage(w http.ResponseWriter, r *http.Request) {
	f, h, err := r.FormFile("image")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	id := uuid.New()

	format, err := ic.processImage(id, f)
	if err != nil {
		panic(err)
	}

	if err := ic.queries.CreateImage(r.Context(), db.CreateImageParams{
		ImageID:  id.String(),
		Filename: h.Filename,
		Format:   format,
	}); err != nil {
		panic(err)
	}

	http.Redirect(w, r, "..", http.StatusSeeOther)
}

func (ic *imageController) processImage(id uuid.UUID, r io.Reader) (string, error) {
	// Decode the image config, preserving the read part of the image in a buffer.
	buf := new(bytes.Buffer)
	config, format, err := image.DecodeConfig(io.TeeReader(r, buf))
	if err != nil {
		return "", err
	}

	// Reassemble the image reader using the buffer.
	r = io.MultiReader(bytes.NewReader(buf.Bytes()), r)

	// Copy the original image data to disk.
	orig, err := ic.origRoot.Create(fmt.Sprintf("%s.%s", id, format))
	if err != nil {
		return "", err
	}
	defer orig.Close()
	r = io.TeeReader(r, orig)

	// Fully decode the image.
	origImg, _, err := image.Decode(r)
	if err != nil {
		return "", err
	}

	// Generate thumbnails in parallel.
	done := make(chan error, 2)
	go func() {
		done <- generateThumbnail(ic.feedRoot, origImg, id, 600, config)
	}()
	go func() {
		done <- generateThumbnail(ic.thumbRoot, origImg, id, 100, config)
	}()

	// Return the first error, if any.
	for range len(done) {
		if err := <-done; err != nil {
			return "", err
		}
	}

	// Return the image format.
	return format, nil
}

func generateThumbnail(root *os.Root, img image.Image, id uuid.UUID, maxDim int, config image.Config) error {
	f, err := root.Create(fmt.Sprintf("%s.png", id))
	if err != nil {
		return err
	}
	defer f.Close()

	img = imaging.Thumbnail(img, min(maxDim, config.Width), min(maxDim, config.Height), imaging.CatmullRom)
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("error encoding image %s: %w", id, err)
	}

	return nil
}
