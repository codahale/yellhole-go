package storage

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type jsonStore[T any] struct {
	root    *os.Root
	indexes map[string]func(*T) string
}

func newJSONStore[T any](root *os.Root, dir string, indexes map[string]func(*T) string) (*jsonStore[T], error) {
	// Ensure that the base directory exists.
	_ = root.Mkdir(dir, 0755)

	// Open root/dir and limit the actions to that.
	root, err := root.OpenRoot(dir)
	if err != nil {
		return nil, err
	}

	// Ensure that each index directory exists.
	for index := range indexes {
		_ = root.Mkdir(index, 0755)
	}

	return &jsonStore[T]{root, indexes}, nil
}

func (s *jsonStore[T]) fetch(id string) (*T, error) {
	f, err := s.root.Open(fmt.Sprintf("%s.json", id))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var item T
	if err := json.NewDecoder(f).Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (s *jsonStore[T]) list(index string, n int) ([]*T, error) {
	items := make([]*T, 0, n)
	if err := fs.WalkDir(s.root.FS(), index, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		ext := filepath.Ext(path)
		id := strings.TrimSuffix(filepath.Base(path), ext)
		if ext == ".json" || ext == ".idx" {
			item, err := s.fetch(id)
			if err != nil {
				return err
			}
			items = append(items, item)
		}

		if len(items) == n {
			return fs.SkipAll
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *jsonStore[T]) listKeys(index string, n int) ([]string, error) {
	keys := make([]string, 0, n)
	if err := fs.WalkDir(s.root.FS(), index, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && d.Name() != index {
			keys = append(keys, d.Name())
			return fs.SkipDir
		}

		if len(keys) == n {
			return fs.SkipAll
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *jsonStore[T]) create(id string, item *T) error {
	// Write the item to the base directory.
	f, err := s.root.Create(fmt.Sprintf("%s.json", id))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(item); err != nil {
		return err
	}

	// Add index files.
	for index, keyFunc := range s.indexes {
		// Create a key for the index.
		key := keyFunc(item)

		// Construct the paths of the key directory and the index file.
		keyDir := filepath.Join(index, key)
		path := filepath.Join(keyDir, fmt.Sprintf("%s.idx", id))

		// Ensure the key directory exists.
		_ = s.root.Mkdir(keyDir, 0755)

		// Create an empty index file.
		f, err := s.root.Create(path)
		if err != nil {
			return err
		}
		_ = f.Close()
	}

	return nil
}

func (s *jsonStore[T]) remove(id string) error {
	item, err := s.fetch(id)
	if err != nil {
		return err
	}

	// Remove index files.
	for index, keyFunc := range s.indexes {
		// Create a key for the index.
		key := keyFunc(item)

		// Construct the paths of the key directory and the index file.
		keyDir := filepath.Join(index, key)
		path := filepath.Join(keyDir, fmt.Sprintf("%s.idx", id))

		// Remove the index file, if possible.
		_ = s.root.Remove(path)

		// Remove the key directory, if possible.
		_ = s.root.Remove(keyDir)
	}

	return s.root.Remove(fmt.Sprintf("%s.json", id))
}
