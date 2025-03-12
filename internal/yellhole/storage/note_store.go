package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/codahale/yellhole-go/internal/yellhole/model"
	"github.com/codahale/yellhole-go/internal/yellhole/model/id"
)

type NoteStore struct {
	notes *jsonStore[model.Note]
}

func NewNoteStore(root *os.Root) (*NoteStore, error) {
	notes, err := newJSONStore(root, "notes",
		map[string]func(*model.Note) string{
			"year": func(note *model.Note) string {
				return note.CreatedAt.Format("2006")
			},
			"month": func(note *model.Note) string {
				return note.CreatedAt.Format("2006-01")
			},
			"week": func(note *model.Note) string {
				y, w := note.CreatedAt.ISOWeek()
				return fmt.Sprintf("%04d-%02d", y, w)
			},
			"day": func(note *model.Note) string {
				return note.CreatedAt.Format("2006-01-02")
			},
		})
	if err != nil {
		return nil, err
	}
	return &NoteStore{notes}, nil
}

func (s *NoteStore) Fetch(id string) (*model.Note, error) {
	return s.notes.fetch(id)
}

func (s *NoteStore) Recent(n int) ([]*model.Note, error) {
	return s.notes.list(".", n)
}

func (s *NoteStore) Years(n int) ([]string, error) {
	return s.notes.listKeys("year", n)
}

func (s *NoteStore) Months(n int) ([]string, error) {
	return s.notes.listKeys("month", n)
}

func (s *NoteStore) Weeks(n int) ([]string, error) {
	return s.notes.listKeys("week", n)
}

func (s *NoteStore) Days(n int) ([]string, error) {
	return s.notes.listKeys("day", n)
}

func (s *NoteStore) Year(t time.Time, n int) ([]*model.Note, error) {
	return s.notes.list(filepath.Join("year", t.Format("2006")), n)
}

func (s *NoteStore) Month(t time.Time, n int) ([]*model.Note, error) {
	return s.notes.list(filepath.Join("month", t.Format("2006-01")), n)
}

func (s *NoteStore) Week(t time.Time, n int) ([]*model.Note, error) {
	y, w := t.ISOWeek()
	return s.notes.list(filepath.Join("week", fmt.Sprintf("%04d-%02d", y, w)), n)
}

func (s *NoteStore) Day(t time.Time, n int) ([]*model.Note, error) {
	return s.notes.list(filepath.Join("day", t.Format("2006-01-02")), n)
}

func (s *NoteStore) Create(body string, createdAt time.Time) (*model.Note, error) {
	id := id.New(createdAt)
	note := model.Note{
		ID:        id,
		Body:      body,
		CreatedAt: createdAt,
	}

	if err := s.notes.create(id, &note); err != nil {
		return nil, err
	}

	return &note, nil
}
