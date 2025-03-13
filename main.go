package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

//go:generate go tool sqlc generate -f db/sqlc.yaml

func main() {
	addr := ":8080"
	dataDir := "./data"

	// Open the data directory as a file system root.
	log.Printf("storing data in %s", dataDir)
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer root.Close()

	// Ensure that the images subdirectory exists and open it as a file system root.
	_ = root.Mkdir("images", 0755)
	imageRoot, err := root.OpenRoot("images")
	if err != nil {
		log.Fatal(err)
	}
	defer imageRoot.Close()

	// Ensure that the images/{feed,orig,thumb} subdirectories exist and open them as file system roots.
	_ = imageRoot.Mkdir("feed", 0755)
	feedImageRoot, err := imageRoot.OpenRoot("feed")
	if err != nil {
		log.Fatal(err)
	}
	defer feedImageRoot.Close()

	_ = imageRoot.Mkdir("thumb", 0755)
	thumbImageRoot, err := imageRoot.OpenRoot("thumb")
	if err != nil {
		log.Fatal(err)
	}
	defer thumbImageRoot.Close()

	_ = imageRoot.Mkdir("orig", 0755)
	origImageRoot, err := imageRoot.OpenRoot("orig")
	if err != nil {
		log.Fatal(err)
	}
	defer origImageRoot.Close()

	// Create FS handlers for the feed and thumbnail images.
	feedImageHandler := http.FileServerFS(feedImageRoot.FS())
	thumbImageHandler := http.FileServerFS(thumbImageRoot.FS())

	// Create a handler for downloading images from the internet.
	downloadImageHandler := func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get(r.FormValue("url"))
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		id := uuid.New()
		format, err := processImage(origImageRoot, feedImageRoot, thumbImageRoot, id, resp.Body)
		if err != nil {
			panic(err)
		}

		_, _ = fmt.Fprintf(w, "That was a %s image.", format)
	}

	// Create a handler for uploading images from the browser.
	uploadImageHandler := func(w http.ResponseWriter, r *http.Request) {
		f, _, err := r.FormFile("image")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		id := uuid.New()
		format, err := processImage(origImageRoot, feedImageRoot, thumbImageRoot, id, f)
		if err != nil {
			panic(err)
		}

		_, _ = fmt.Fprintf(w, "That was a %s image.", format)
	}

	// Construct a route map of handlers.
	mux := http.NewServeMux()
	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", feedImageHandler))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", thumbImageHandler))
	mux.Handle("POST /images/download", http.HandlerFunc(downloadImageHandler))
	mux.Handle("POST /images/upload", http.HandlerFunc(uploadImageHandler))
	if err := handlePublicAssets(mux); err != nil {
		log.Fatalln(err)
	}

	// Listen for HTTP requests.
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalln("error serving HTTP", err)
	}
}

//go:embed public
var public embed.FS

func handlePublicAssets(mux *http.ServeMux) error {
	public, err := fs.Sub(public, "public")
	if err != nil {
		return err
	}

	handler := http.FileServerFS(public)
	return fs.WalkDir(public, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			log.Println(path, d)
			mux.Handle(fmt.Sprintf("GET /%s", path), handler)
		}

		return nil
	})
}

func processImage(origRoot, feedRoot, thumbRoot *os.Root, id uuid.UUID, r io.Reader) (string, error) {
	// Decode the image config, preserving the read part of the image in a buffer.
	buf := new(bytes.Buffer)
	_, format, err := image.DecodeConfig(io.TeeReader(r, buf))
	if err != nil {
		return "", err
	}

	// Reassemble the image reader using the buffer.
	r = io.MultiReader(bytes.NewReader(buf.Bytes()), r)

	// Copy the original image data to disk.
	orig, err := origRoot.Create(fmt.Sprintf("%s.%s", id, format))
	if err != nil {
		return "", err
	}
	defer orig.Close()
	r = io.TeeReader(r, orig)

	// Fully decode the image.
	origImg, _, err := image.Decode(r)
	if err != nil {
		return "", err
	}

	// Generate thumbnails in parallel.
	done := make(chan error, 2)
	go func() { done <- generateThumbnail(feedRoot, origImg, id, 600) }()
	go func() { done <- generateThumbnail(thumbRoot, origImg, id, 100) }()

	// Return the first error, if any.
	for range len(done) {
		if err := <-done; err != nil {
			return "", err
		}
	}

	// Return the image format.
	return format, nil
}

func generateThumbnail(root *os.Root, img image.Image, id uuid.UUID, maxDim int) error {
	f, err := root.Create(fmt.Sprintf("%s.png", id))
	if err != nil {
		return err
	}
	defer f.Close()

	feedImg := imaging.Thumbnail(img, maxDim, maxDim, imaging.CatmullRom)
	if err := png.Encode(f, feedImg); err != nil {
		return fmt.Errorf("error encoding feed image: %w", err)
	}

	return nil
}
