package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/codahale/yellhole-go/internal/yellhole/model"
	"github.com/codahale/yellhole-go/internal/yellhole/model/id"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

type ImageStore struct {
	root *os.Root
}

func NewImageStore(root *os.Root) (*ImageStore, error) {
	// Ensure that root/images/{feed,original,thumb} exists.
	_ = root.Mkdir("images", 0755)
	_ = root.Mkdir("images/feed", 0755)
	_ = root.Mkdir("images/original", 0755)
	_ = root.Mkdir("images/thumb", 0755)

	// Open root/dir and limit the actions to that.
	dir, err := root.OpenRoot("images")
	if err != nil {
		return nil, err
	}

	return &ImageStore{dir}, nil
}

func (s *ImageStore) Fetch(id string) (*model.Image, error) {
	return s.load(fmt.Sprintf("%s.json", id))
}

func (s *ImageStore) Recent(n int) ([]*model.Image, error) {
	images := make([]*model.Image, 0, n)
	if err := fs.WalkDir(s.root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && path != "." {
			return fs.SkipDir
		}

		if filepath.Ext(path) == ".json" {
			f, err := s.root.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			var img model.Image
			if err := json.NewDecoder(f).Decode(&img); err != nil {
				return err
			}

			images = append(images, &img)
			if len(images) == n {
				return fs.SkipAll
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return images, nil
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
	orig, err := s.root.Create(origPath)
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
	feed, err := s.root.Create(feedPath)
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
	thumb, err := s.root.Create(thumbPath)
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
	f, err := s.root.Create(fmt.Sprintf("%s.json", id))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(&img); err != nil {
		return nil, err
	}

	return &img, nil
}

func (s *ImageStore) load(filename string) (*model.Image, error) {
	f, err := s.root.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var img model.Image
	if err := json.NewDecoder(f).Decode(&img); err != nil {
		return nil, err
	}

	return &img, nil
}
