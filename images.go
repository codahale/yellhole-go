package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/codahale/yellhole-go/imgstore"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
)

func handleFeedImage(images *imgstore.Store) http.Handler {
	return cacheControl(http.FileServerFS(images.FeedImages()), "max-age=31536000,immutable")
}

func handleThumbImage(images *imgstore.Store) http.Handler {
	return cacheControl(http.FileServerFS(images.ThumbImages()), "max-age=31536000,immutable")
}

func handleDownloadImage(queries *db.Queries, images *imgstore.Store, baseURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imageURL := r.FormValue("url")

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, imageURL, nil)
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
			slog.ErrorContext(r.Context(), "unable to download image", "imageURL", imageURL, "statusCode", resp.StatusCode)
			http.Error(w, "unable to download image", http.StatusInternalServerError)
			return
		}

		id := uuid.New()

		filename, format, err := images.Add(r.Context(), id, resp.Body)
		if err != nil {
			panic(err)
		}

		if err := queries.CreateImage(r.Context(), id.String(), filename, imageURL, format, time.Now()); err != nil {
			panic(err)
		}

		http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
	})
}

func handleUploadImage(queries *db.Queries, images *imgstore.Store, baseURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, h, err := r.FormFile("image")
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = f.Close()
		}()

		id := uuid.New()

		filename, format, err := images.Add(r.Context(), id, f)
		if err != nil {
			panic(err)
		}

		if err := queries.CreateImage(r.Context(), id.String(), filename, h.Filename, format, time.Now()); err != nil {
			panic(err)
		}

		http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
	})
}
