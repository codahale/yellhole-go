package imgstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg" // support JPEG images
	_ "image/png"  // support PNG images
	"io"
	"io/fs"
	"math"
	"os"

	"github.com/HugoSmits86/nativewebp"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
	"golang.org/x/sync/errgroup"
)

type Store struct {
	root   *os.Root
	images *os.Root
	feed   *os.Root
	orig   *os.Root
	thumb  *os.Root
}

func New(dataDir string) (store *Store, err error) {
	store = new(Store)

	store.root, err = os.OpenRoot(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open data directory: %w", err)
	}

	if err = store.root.Mkdir("images", 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to create directory: %w", err))
	}
	store.images, err = store.root.OpenRoot("images")
	if err != nil {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to open images directory: %w", err))
	}

	if err = store.images.Mkdir("feed", 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to create directory: %w", err))
	}
	store.feed, err = store.images.OpenRoot("feed")
	if err != nil {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to open feed images directory: %w", err))
	}

	if err = store.images.Mkdir("original", 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to create directory: %w", err))
	}
	store.orig, err = store.images.OpenRoot("original")
	if err != nil {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to open original images directory: %w", err))
	}

	if err = store.images.Mkdir("thumb", 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to create directory: %w", err))
	}
	store.thumb, err = store.images.OpenRoot("thumb")
	if err != nil {
		return nil, errors.Join(store.Close(), fmt.Errorf("failed to open thumbnail images directory: %w", err))
	}

	return store, nil
}

func (s *Store) Close() (err error) {
	if s.thumb != nil {
		err = errors.Join(err, s.thumb.Close())
	}
	if s.orig != nil {
		err = errors.Join(err, s.orig.Close())
	}
	if s.feed != nil {
		err = errors.Join(err, s.feed.Close())
	}
	if s.images != nil {
		err = errors.Join(err, s.images.Close())
	}
	if s.root != nil {
		err = errors.Join(err, s.root.Close())
	}
	return err
}

func (s *Store) FeedImages() fs.FS {
	return s.feed.FS()
}

func (s *Store) ThumbImages() fs.FS {
	return s.thumb.FS()
}

func (s *Store) Add(ctx context.Context, id uuid.UUID, r io.Reader) (filename string, format string, err error) {
	// Decode the image config, preserving the read part of the image in a buffer.
	buf := new(bytes.Buffer)
	_, format, err = image.DecodeConfig(io.TeeReader(r, buf))
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image configuration: %w", err)
	}

	// Reassemble the image reader using the buffer.
	r = io.MultiReader(bytes.NewReader(buf.Bytes()), r)

	// Copy the original image data to the disk as it's decoded.
	orig, err := s.orig.Create(fmt.Sprintf("%s.%s", id, format))
	if err != nil {
		return "", "", fmt.Errorf("failed to create original image file: %w", err)
	}
	defer func() {
		err = errors.Join(err, orig.Close())
	}()
	r = io.TeeReader(r, orig)

	filename = id.String() + ".webp"

	// If the image is a GIF, decode it as such. Animated GIFs need to be handled separately.
	if format == "gif" {
		return filename, format, s.processAnim(ctx, r, filename)
	}

	// Fully decode the image.
	origImg, _, err := image.Decode(r)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Generate thumbnails.
	return filename, format, s.processStatic(ctx, origImg, filename)
}

func (s *Store) processAnim(ctx context.Context, r io.Reader, filename string) error {
	// Decode all frames.
	img, err := gif.DecodeAll(r)
	if err != nil {
		return fmt.Errorf("failed to decode animated GIF: %w", err)
	}

	// If there's only one frame, treat it as a static image.
	if len(img.Image) == 1 {
		return s.processStatic(ctx, img.Image[0], filename)
	}

	// Generate thumbnails in parallel.
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return resizeAnim(s.feed, img, filename, 600)
	})
	eg.Go(func() error {
		return resizeAnim(s.thumb, img, filename, 100)
	})
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to resize animated image: %w", err)
	}

	return nil
}

func (s *Store) processStatic(ctx context.Context, img image.Image, filename string) error {
	// Generate thumbnails in parallel.
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return resizeStatic(s.feed, img, filename, 600)
	})
	eg.Go(func() error {
		return resizeStatic(s.thumb, img, filename, 100)
	})
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to resize static image: %w", err)
	}
	return nil
}

func resizeStatic(root *os.Root, src image.Image, filename string, maxWidth int) (err error) {
	f, err := root.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create resized static image file %s: %w", filename, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	thumbnail := resize(src, maxWidth)

	if err := nativewebp.Encode(f, thumbnail, nil); err != nil {
		return fmt.Errorf("failed to encode static image %s: %w", filename, err)
	}

	return nil
}

func resizeAnim(root *os.Root, src *gif.GIF, filename string, maxWidth int) (err error) {
	f, err := root.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create resized animated image file %s: %w", filename, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	thumbnail := nativewebp.Animation{
		Disposals: make([]uint, len(src.Disposal)),
		Durations: make([]uint, len(src.Delay)),
		Images:    make([]image.Image, len(src.Image)),
		LoopCount: uint16(src.LoopCount), //nolint:gosec // inconsequential
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
		thumbnail.Durations[i] = uint(v) //nolint:gosec // inconsequential
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
		return fmt.Errorf("failed to encode animated image %s: %w", filename, err)
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
