package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
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

type imageController struct {
	config            *Config
	queries           *db.Queries
	root              *os.Root
	feedRoot          *os.Root
	origRoot          *os.Root
	thumbRoot         *os.Root
	feedImageHandler  http.Handler
	thumbImageHandler http.Handler
}

func newImageController(config *Config, dataRoot *os.Root, queries *db.Queries) (*imageController, error) {
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

	filename, format, err := ic.processStream(r.Context(), id, resp.Body)
	if err != nil {
		panic(err)
	}

	if err := ic.queries.CreateImage(r.Context(), id.String(), filename, url, format, time.Now()); err != nil {
		panic(err)
	}

	http.Redirect(w, r, ic.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
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

	filename, format, err := ic.processStream(r.Context(), id, f)
	if err != nil {
		panic(err)
	}

	if err := ic.queries.CreateImage(r.Context(), id.String(), filename, h.Filename, format, time.Now()); err != nil {
		panic(err)
	}

	http.Redirect(w, r, ic.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
}

func (ic *imageController) processStream(ctx context.Context, id uuid.UUID, r io.Reader) (string, string, error) {
	// Decode the image config, preserving the read part of the image in a buffer.
	buf := new(bytes.Buffer)
	_, format, err := image.DecodeConfig(io.TeeReader(r, buf))
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

	// If the image is a GIF, decode it as such. Animated GIFs need to be special-cased.
	if format == "gif" {
		filename, err := ic.processGIF(ctx, id, r)
		return filename, format, err
	}

	// Fully decode the image.
	origImg, _, err := image.Decode(r)
	if err != nil {
		return "", "", err
	}

	// Generate thumbnails.
	filename, err := ic.processWEBP(ctx, id, origImg)
	return filename, format, err
}

func (ic *imageController) processGIF(ctx context.Context, id uuid.UUID, r io.Reader) (string, error) {
	// Decode all frames.
	img, err := gif.DecodeAll(r)
	if err != nil {
		return "", err
	}

	// If there's only one frame, treat it as a static image.
	if len(img.Image) == 1 {
		return ic.processWEBP(ctx, id, img.Image[0])
	}

	// Otherwise, resize all frames and keep it as a GIF.
	filename := id.String() + ".gif"

	// Generate thumbnails in parallel.
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return generateGIFThumbnail(ic.feedRoot, img, filename, 600)
	})
	eg.Go(func() error {
		return generateGIFThumbnail(ic.thumbRoot, img, filename, 100)
	})
	if err := eg.Wait(); err != nil {
		return "", err
	}

	return filename, nil
}

func (ic *imageController) processWEBP(ctx context.Context, id uuid.UUID, img image.Image) (string, error) {
	filename := id.String() + ".webp"

	// Generate thumbnails in parallel.
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return generateWEBPThumbnail(ic.feedRoot, img, filename, 600)
	})
	eg.Go(func() error {
		return generateWEBPThumbnail(ic.thumbRoot, img, filename, 100)
	})
	if err := eg.Wait(); err != nil {
		return "", err
	}

	return filename, nil
}

func generateWEBPThumbnail(root *os.Root, src image.Image, filename string, maxWidth int) error {
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

func generateGIFThumbnail(root *os.Root, src *gif.GIF, filename string, maxWidth int) error {
	f, err := root.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	// Copy the image metadata.
	thumbnail := *src
	thumbnail.Image = make([]*image.Paletted, len(src.Image))

	// Create a new RGBA image to hold the incremental frames.
	firstFrame := src.Image[0].Bounds()
	b := image.Rect(0, 0, firstFrame.Dx(), firstFrame.Dy())
	img := image.NewRGBA(b)

	// Resize each frame.
	for i := range src.Image {
		frame := src.Image[i]
		bounds := frame.Bounds()
		previous := img
		draw.Draw(img, bounds, frame, bounds.Min, draw.Over)
		thumbnail.Image[i] = imageToPaletted(resize(img, maxWidth), frame.Palette)

		switch src.Disposal[i] {
		case gif.DisposalBackground:
			// I'm just assuming that the gif package will apply the appropriate
			// background here, since there doesn't seem to be an easy way to
			// access the global color table.
			img = image.NewRGBA(b)
		case gif.DisposalPrevious:
			img = previous
		}
	}

	// Set image.Config to new height and width.
	thumbnail.Config.Width = thumbnail.Image[0].Bounds().Max.X
	thumbnail.Config.Height = thumbnail.Image[0].Bounds().Max.Y

	if err := gif.EncodeAll(f, &thumbnail); err != nil {
		return fmt.Errorf("error encoding image %s: %w", filename, err)
	}

	return nil

}

func imageToPaletted(img image.Image, p color.Palette) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(b, p)
	draw.FloydSteinberg.Draw(pm, b, img, image.Point{})
	return pm
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
