package db

import (
	"context"
	"slices"
	"time"
)

type Week struct {
	StartDate string
	EndDate   string
}

func (db *Queries) WeeksWithNotes(ctx context.Context) ([]Week, error) {
	var weeks []Week
	timestamps, err := db.AllNoteTimestamps(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range timestamps {
		ts := time.Unix(v, 0)
		midnight := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
		start := midnight.AddDate(0, 0, int(time.Sunday-ts.Weekday()))
		end := start.AddDate(0, 0, 6)
		weeks = append(weeks, Week{start.Format("2006-01-02"), end.Format("2006-01-02")})
	}

	weeks = slices.Compact(weeks)

	return weeks, nil
}
