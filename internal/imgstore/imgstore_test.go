package imgstore_test

import (
	"image"
	"os"
	"testing"

	"github.com/codahale/yellhole-go/internal/imgstore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"golang.org/x/image/webp"
)

func TestStore_Add_Static(t *testing.T) {
	t.Parallel()

	store, err := imgstore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	})

	f, err := os.Open("../../yellhole.webp")
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

	if got, want := feedImg.Bounds(), image.Rect(0, 0, 400, 400); !cmp.Equal(got, want) {
		t.Errorf("Bounds = %d, want %d", got, want)
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

	if got, want := thumbImg.Bounds(), image.Rect(0, 0, 100, 100); !cmp.Equal(got, want) {
		t.Errorf("Bounds = %d, want %d", got, want)
	}
}

func TestStore_Add_Animated(t *testing.T) {
	t.Parallel()

	store, err := imgstore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	})

	f, err := os.Open("banana.gif")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	id := uuid.UUID{185, 46, 26, 0, 209, 35, 64, 140, 159, 160, 25, 139, 189, 33, 99, 102}
	filename, format, err := store.Add(t.Context(), id, f)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := format, "gif"; got != want {
		t.Errorf("format = %q, want %q", got, want)
	}

	if got, want := filename, "b92e1a00-d123-408c-9fa0-198bbd216366.webp"; got != want {
		t.Errorf("filename = %q, want %q", got, want)
	}

	// TODO test bounds once animated WEBP decoding drops
}
