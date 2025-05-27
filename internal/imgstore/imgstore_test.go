package imgstore_test

import (
	"github.com/codahale/yellhole-go/internal/imgstore"
	"os"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/image/webp"
)

func TestStore_Add_Static(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = root.Close()
	})

	store, err := imgstore.New(root)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open("../yellhole.webp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	id := uuid.New()
	filename, format, err := store.Add(t.Context(), id, f)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := format, "webp"; got != want {
		t.Errorf("format = %q, want %q", got, want)
	}

	feed, err := store.FeedImages().Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = feed.Close()
	})

	feedImg, err := webp.Decode(feed)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := feedImg.Bounds().Dx(), 400; got != want {
		t.Errorf("Dx = %d, want %d", got, want)
	}

	if got, want := feedImg.Bounds().Dy(), 400; got != want {
		t.Errorf("Dy = %d, want %d", got, want)
	}

	thumb, err := store.ThumbImages().Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = thumb.Close()
	})

	thumbImg, err := webp.Decode(thumb)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := thumbImg.Bounds().Dx(), 100; got != want {
		t.Errorf("Dx = %d, want %d", got, want)
	}

	if got, want := thumbImg.Bounds().Dy(), 100; got != want {
		t.Errorf("Dy = %d, want %d", got, want)
	}
}

func TestStore_Add_Animated(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = root.Close()
	})

	store, err := imgstore.New(root)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open("banana.gif")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	id := uuid.New()
	filename, format, err := store.Add(t.Context(), id, f)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := format, "gif"; got != want {
		t.Errorf("format = %q, want %q", got, want)
	}

	if got, want := filename, id.String()+".webp"; got != want {
		t.Errorf("filename = %q, want %q", got, want)
	}

	// TODO test bounds once animated WEBP decoding drops
}
