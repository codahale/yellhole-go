package storage

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/codahale/yellhole-go/internal/yellhole/model"
	"github.com/codahale/yellhole-go/internal/yellhole/model/id"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

type ImageStore struct {
	images *jsonStore[model.Image]
}

func NewImageStore(root *os.Root) (*ImageStore, error) {
	// Create a new JSON store.
	images, err := newJSONStore[model.Image](root, "images", nil)
	if err != nil {
		return nil, err
	}

	//  Ensure that images/{feed,original,thumb} exists.
	for _, dir := range []string{"feed", "original", "thumb"} {
		_ = images.root.Mkdir(dir, 0755)
	}

	return &ImageStore{images}, nil
}

func (s *ImageStore) Fetch(id string) (*model.Image, error) {
	return s.images.fetch(id)
}

func (s *ImageStore) Recent(n int) ([]*model.Image, error) {
	return s.images.list(".", n)
}

func (s *ImageStore) Create(r io.Reader, filename string, createdAt time.Time) (*model.Image, error) {
	// Generate a new image ID.
	id := id.New(createdAt)

	// Decode the image config, preserving the read part of the image in a buffer.
	buf := new(bytes.Buffer)
	_, format, err := image.DecodeConfig(io.TeeReader(r, buf))
	if err != nil {
		return nil, err
	}

	// Reassemble the image reader using the buffer.
	r = io.MultiReader(bytes.NewReader(buf.Bytes()), r)

	// Copy the original image data to disk.
	origPath := filepath.Join("original", fmt.Sprintf("%s.%s", id, format))
	orig, err := s.images.root.Create(origPath)
	if err != nil {
		return nil, err
	}
	defer orig.Close()
	r = io.TeeReader(r, orig)

	// Fully decode the image.
	origImg, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	// Generate a feed PNG image.
	feedPath := filepath.Join("feed", fmt.Sprintf("%s.png", id))
	feed, err := s.images.root.Create(feedPath)
	if err != nil {
		return nil, err
	}
	defer feed.Close()

	feedImg := imaging.Thumbnail(origImg, 600, 600, imaging.CatmullRom)
	if err := png.Encode(feed, feedImg); err != nil {
		return nil, fmt.Errorf("error encoding main image: %w", err)
	}

	// Generate a thumbnail PNG image.
	thumbPath := filepath.Join("thumb", fmt.Sprintf("%s.png", id))
	thumb, err := s.images.root.Create(thumbPath)
	if err != nil {
		return nil, err
	}
	defer thumb.Close()

	thumbImg := imaging.Thumbnail(origImg, 100, 100, imaging.CatmullRom)
	if err := png.Encode(thumb, thumbImg); err != nil {
		return nil, fmt.Errorf("error encoding thumbnail image: %w", err)
	}

	// Create an Image model and write it to disk.
	img := model.Image{
		ID:               id,
		FeedPath:         feedPath,
		OriginalPath:     origPath,
		ThumbnailPath:    thumbPath,
		OriginalFilename: filename,
		CreatedAt:        createdAt,
	}
	if err := s.images.create(id, &img); err != nil {
		return nil, err
	}

	return &img, nil
}
