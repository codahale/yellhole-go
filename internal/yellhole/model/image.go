package model

import "time"

type Image struct {
	ID               string    `json:"id"`
	FeedPath         string    `json:"feed_path"`
	OriginalPath     string    `json:"original_path"`
	ThumbnailPath    string    `json:"thumbnail_path"`
	OriginalFilename string    `json:"original_filename"`
	CreatedAt        time.Time `json:"created_at"`
}
