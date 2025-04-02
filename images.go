package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
	"golang.org/x/sync/errgroup"
)

type imageController struct {
	config            *config
	queries           *db.Queries
	root              *os.Root
	feedRoot          *os.Root
	origRoot          *os.Root
	thumbRoot         *os.Root
	feedImageHandler  http.Handler
	thumbImageHandler http.Handler
}

func newImageController(config *config, dataRoot *os.Root, queries *db.Queries) (*imageController, error) {
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

	return &imageController{config, queries, root, feedRoot, origRoot, thumbRoot, feedImageHandler, thumbImageHandler}, nil
}

func (ic *imageController) feedImage(w http.ResponseWriter, r *http.Request) {
	ic.feedImageHandler.ServeHTTP(w, r)
}

func (ic *imageController) thumbImage(w http.ResponseWriter, r *http.Request) {
	ic.thumbImageHandler.ServeHTTP(w, r)
}

func (ic *imageController) downloadImage(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	id := uuid.New()

	format, err := ic.processImage(r.Context(), id, resp.Body)
	if err != nil {
		panic(err)
	}

	if err := ic.queries.CreateImage(r.Context(), id.String(), url, format, time.Now()); err != nil {
		panic(err)
	}

	http.Redirect(w, r, "..", http.StatusSeeOther)
}

func (ic *imageController) uploadImage(w http.ResponseWriter, r *http.Request) {
	f, h, err := r.FormFile("image")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = f.Close()
	}()

	id := uuid.New()

	format, err := ic.processImage(r.Context(), id, f)
	if err != nil {
		panic(err)
	}

	if err := ic.queries.CreateImage(r.Context(), id.String(), h.Filename, format, time.Now()); err != nil {
		panic(err)
	}

	http.Redirect(w, r, "..", http.StatusSeeOther)
}

func (ic *imageController) processImage(ctx context.Context, id uuid.UUID, r io.Reader) (string, error) {
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
	defer func() {
		_ = orig.Close()
	}()
	r = io.TeeReader(r, orig)

	// Fully decode the image.
	origImg, _, err := image.Decode(r)
	if err != nil {
		return "", err
	}

	// Generate thumbnails in parallel.
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return generateThumbnail(ic.feedRoot, origImg, id, 600)
	})
	eg.Go(func() error {
		return generateThumbnail(ic.thumbRoot, origImg, id, 100)
	})
	if err := eg.Wait(); err != nil {
		return "", err
	}

	// Return the image format.
	return format, nil
}

func generateThumbnail(root *os.Root, img image.Image, id uuid.UUID, maxDim uint) error {
	f, err := root.Create(fmt.Sprintf("%s.png", id))
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	thumbnail := resize.Thumbnail(maxDim, maxDim, img, resize.Lanczos2)
	if err := png.Encode(f, thumbnail); err != nil {
		return fmt.Errorf("error encoding image %s: %w", id, err)
	}

	return nil
}
