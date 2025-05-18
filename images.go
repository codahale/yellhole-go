package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/HugoSmits86/nativewebp"
	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
	"golang.org/x/sync/errgroup"
)

func handleFeedImage(images *imageStore) http.Handler {
	return http.FileServerFS(images.feedRoot.FS())
}

func handleThumbImage(images *imageStore) http.Handler {
	return http.FileServerFS(images.thumbRoot.FS())
}

func handleDownloadImage(config *config, queries *db.Queries, images *imageStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.FormValue("url")

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
		if err != nil {
			panic(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			slog.Error("unable to download image", "url", url, "statusCode", resp.StatusCode)
			http.Error(w, "unable to download image", http.StatusInternalServerError)
			return
		}

		id := uuid.New()

		filename, format, err := images.processStream(r.Context(), id, resp.Body)
		if err != nil {
			panic(err)
		}

		if err := queries.CreateImage(r.Context(), id.String(), filename, url, format, time.Now()); err != nil {
			panic(err)
		}

		http.Redirect(w, r, config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
	})
}

func handleUploadImage(config *config, queries *db.Queries, images *imageStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, h, err := r.FormFile("image")
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = f.Close()
		}()

		id := uuid.New()

		filename, format, err := images.processStream(r.Context(), id, f)
		if err != nil {
			panic(err)
		}

		if err := queries.CreateImage(r.Context(), id.String(), filename, h.Filename, format, time.Now()); err != nil {
			panic(err)
		}

		http.Redirect(w, r, config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
	})
}

type imageStore struct {
	root      *os.Root
	feedRoot  *os.Root
	origRoot  *os.Root
	thumbRoot *os.Root
}

func newImageStore(dataRoot *os.Root) (*imageStore, error) {
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

	return &imageStore{root, feedRoot, origRoot, thumbRoot}, nil
}

func (ic *imageStore) processStream(ctx context.Context, id uuid.UUID, r io.Reader) (filename string, format string, err error) {
	// Decode the image config, preserving the read part of the image in a buffer.
	buf := new(bytes.Buffer)
	_, format, err = image.DecodeConfig(io.TeeReader(r, buf))
	if err != nil {
		return "", "", err
	}

	// Reassemble the image reader using the buffer.
	r = io.MultiReader(bytes.NewReader(buf.Bytes()), r)

	// Copy the original image data to disk as it's decoded.
	orig, err := ic.origRoot.Create(fmt.Sprintf("%s.%s", id, format))
	if err != nil {
		return "", "", err
	}
	defer func() {
		_ = orig.Close()
	}()
	r = io.TeeReader(r, orig)

	filename = id.String() + ".webp"

	// If the image is a GIF, decode it as such. Animated GIFs need to be special-cased.
	if format == "gif" {
		err = ic.processAnim(ctx, r, filename)
		return
	}

	// Fully decode the image.
	origImg, _, err := image.Decode(r)
	if err != nil {
		return "", "", err
	}

	// Generate thumbnails.
	err = ic.processStatic(ctx, origImg, filename)
	return
}

func (ic *imageStore) processAnim(ctx context.Context, r io.Reader, filename string) error {
	// Decode all frames.
	img, err := gif.DecodeAll(r)
	if err != nil {
		return err
	}

	// If there's only one frame, treat it as a static image.
	if len(img.Image) == 1 {
		return ic.processStatic(ctx, img.Image[0], filename)
	}

	// Generate thumbnails in parallel.
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return generateAnimThumbnail(ic.feedRoot, img, filename, 600)
	})
	eg.Go(func() error {
		return generateAnimThumbnail(ic.thumbRoot, img, filename, 100)
	})
	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (ic *imageStore) processStatic(ctx context.Context, img image.Image, filename string) error {
	// Generate thumbnails in parallel.
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return generateStaticThumbnail(ic.feedRoot, img, filename, 600)
	})
	eg.Go(func() error {
		return generateStaticThumbnail(ic.thumbRoot, img, filename, 100)
	})
	return eg.Wait()
}

func generateStaticThumbnail(root *os.Root, src image.Image, filename string, maxWidth int) error {
	f, err := root.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	thumbnail := resize(src, maxWidth)

	if err := nativewebp.Encode(f, thumbnail, nil); err != nil {
		return fmt.Errorf("error encoding image %s: %w", filename, err)
	}

	return nil
}

func generateAnimThumbnail(root *os.Root, src *gif.GIF, filename string, maxWidth int) error {
	f, err := root.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	thumbnail := nativewebp.Animation{
		Disposals: make([]uint, len(src.Disposal)),
		Durations: make([]uint, len(src.Delay)),
		Images:    make([]image.Image, len(src.Image)),
		LoopCount: uint16(src.LoopCount),
	}

	for i, d := range src.Disposal {
		switch d {
		case gif.DisposalNone, gif.DisposalPrevious:
			thumbnail.Disposals[i] = 0
		case gif.DisposalBackground:
			thumbnail.Disposals[i] = 1
		}
	}

	for i, v := range src.Delay {
		thumbnail.Durations[i] = uint(v)
	}

	// Create a new RGBA image to hold the incremental frames.
	firstFrame := src.Image[0].Bounds()
	b := image.Rect(0, 0, firstFrame.Dx(), firstFrame.Dy())
	img := image.NewRGBA(b)

	// Resize each frame.
	for i, frame := range src.Image {
		bounds := frame.Bounds()
		previous := img
		draw.Draw(img, bounds, frame, bounds.Min, draw.Over)
		thumbnail.Images[i] = resize(img, maxWidth)

		switch src.Disposal[i] {
		case gif.DisposalBackground:
			// https://github.com/golang/go/issues/20694
			img = image.NewRGBA(b)
		case gif.DisposalPrevious:
			img = previous
		}
	}

	if err := nativewebp.EncodeAll(f, &thumbnail, nil); err != nil {
		return fmt.Errorf("error encoding image %s: %w", filename, err)
	}

	return nil

}

func resize(img image.Image, maxWidth int) image.Image {
	if img.Bounds().Max.X <= maxWidth {
		return img
	}

	ratio := (float64)(img.Bounds().Max.Y) / (float64)(img.Bounds().Max.X)
	height := int(math.Round(float64(maxWidth) * ratio))

	thumbnail := image.NewRGBA(image.Rect(0, 0, maxWidth, height))
	draw.CatmullRom.Scale(thumbnail, thumbnail.Rect, img, img.Bounds(), draw.Over, nil)
	return thumbnail
}
