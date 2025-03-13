package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"

	"github.com/codahale/yellhole-go/db"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

type imageController struct {
	queries           *db.Queries
	root              *os.Root
	feedRoot          *os.Root
	origRoot          *os.Root
	thumbRoot         *os.Root
	feedImageHandler  http.Handler
	thumbImageHandler http.Handler
}

func newImageController(dataRoot *os.Root, queries *db.Queries) (*imageController, error) {
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

	_ = root.Mkdir("orig", 0755)
	origRoot, err := root.OpenRoot("orig")
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
	// TODO check session auth
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

	_, _ = fmt.Fprintf(w, "That was a %s image.", format)
}

func (ic *imageController) UploadImage(w http.ResponseWriter, r *http.Request) {
	// TODO check session auth

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

	_, _ = fmt.Fprintf(w, "That was a %s image.", format)
}

func (ic *imageController) processImage(id uuid.UUID, r io.Reader) (string, error) {
	// Decode the image config, preserving the read part of the image in a buffer.
	buf := new(bytes.Buffer)
	_, format, err := image.DecodeConfig(io.TeeReader(r, buf))
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
	go func() { done <- generateThumbnail(ic.feedRoot, origImg, id, 600) }()
	go func() { done <- generateThumbnail(ic.thumbRoot, origImg, id, 100) }()

	// Return the first error, if any.
	for range len(done) {
		if err := <-done; err != nil {
			return "", err
		}
	}

	// Return the image format.
	return format, nil
}

func (ic *imageController) Close() error {
	for _, root := range []*os.Root{ic.origRoot, ic.feedRoot, ic.thumbRoot, ic.root} {
		if err := root.Close(); err != nil {
			return err
		}
	}
	return nil
}

func generateThumbnail(root *os.Root, img image.Image, id uuid.UUID, maxDim int) error {
	f, err := root.Create(fmt.Sprintf("%s.png", id))
	if err != nil {
		return err
	}
	defer f.Close()

	img = imaging.Thumbnail(img, maxDim, maxDim, imaging.CatmullRom)
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("error encoding image %s: %w", id, err)
	}

	return nil
}
