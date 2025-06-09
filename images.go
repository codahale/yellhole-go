package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/codahale/yellhole-go/internal/db"
	"github.com/codahale/yellhole-go/internal/imgstore"
	"github.com/google/uuid"
)

const cacheControlImmutable = "public,immutable,max-age=31536000"

func handleFeedImage(images *imgstore.Store) http.Handler {
	return cacheControl(http.FileServerFS(images.FeedImages()), cacheControlImmutable)
}

func handleThumbImage(images *imgstore.Store) http.Handler {
	return cacheControl(http.FileServerFS(images.ThumbImages()), cacheControlImmutable)
}

func handleDownloadImage(logger *slog.Logger, queries *db.Queries, images *imgstore.Store, baseURL *url.URL) appHandler {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:       20,
			DisableCompression:    false,
		},
		Timeout: 60 * time.Second,
	}

	return func(w http.ResponseWriter, r *http.Request) (err error) {
		imageURL := r.FormValue("url")

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, imageURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for image download from %q: %w", imageURL, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download image from %q: %w", imageURL, err)
		}
		defer func() {
			err = errors.Join(err, resp.Body.Close())
		}()

		if resp.StatusCode != http.StatusOK {
			logger.ErrorContext(r.Context(), "unable to download image", "imageURL", imageURL, "statusCode", resp.StatusCode)
			http.Error(w, "unable to download image", http.StatusInternalServerError)
			return nil
		}

		id := uuid.New()

		filename, format, err := images.Add(r.Context(), id, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to add downloaded image to store: %w", err)
		}

		if err := queries.CreateImage(r.Context(), id.String(), filename, imageURL, format, time.Now()); err != nil {
			return fmt.Errorf("failed to create image record in database: %w", err)
		}

		http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return err
	}
}

func handleUploadImage(queries *db.Queries, images *imgstore.Store, baseURL *url.URL) appHandler {
	return func(w http.ResponseWriter, r *http.Request) (err error) {
		f, h, err := r.FormFile("image")
		if err != nil {
			return fmt.Errorf("failed to get uploaded image file: %w", err)
		}
		defer func() {
			err = errors.Join(err, f.Close())
		}()

		id := uuid.New()

		filename, format, err := images.Add(r.Context(), id, f)
		if err != nil {
			return fmt.Errorf("failed to add uploaded image to store: %w", err)
		}

		if err := queries.CreateImage(r.Context(), id.String(), filename, h.Filename, format, time.Now()); err != nil {
			return fmt.Errorf("failed to create image record in database: %w", err)
		}

		http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return nil
	}
}
