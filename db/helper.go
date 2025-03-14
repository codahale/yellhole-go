package db

import (
	"context"
)

type Week struct {
	StartDate string
	EndDate   string
}

func (db *Queries) WeeksWithNotes(ctx context.Context) ([]Week, error) {
	raw, err := db.WeeksWithNotesRaw(ctx)
	if err != nil {
		return nil, err
	}

	weeks := make([]Week, len(raw))
	for i := range raw {
		weeks[i] = Week{
			StartDate: raw[i].StartDate.(string),
			EndDate:   raw[i].EndDate.(string),
		}
	}
	return weeks, nil
}
