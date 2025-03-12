package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestImageStoreCreate(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewImageStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	original, err := os.Open("../../../yellhole.webp")
	if err != nil {
		t.Fatal(err)
	}
	defer original.Close()

	img, err := store.Create(original, "yellhole.webp", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tc.root.Stat(filepath.Join("images", img.OriginalPath)); err != nil {
		t.Error(err)
	}

	if _, err := tc.root.Stat(filepath.Join("images", img.FeedPath)); err != nil {
		t.Error(err)
	}

	if _, err := tc.root.Stat(filepath.Join("images", img.ThumbnailPath)); err != nil {
		t.Error(err)
	}

	if expected, actual := "yellhole.webp", img.OriginalFilename; expected != actual {
		t.Errorf("expected %T but was %T", expected, actual)
	}
}

func TestImageStoreFetch(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewImageStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	original, err := os.Open("../../../yellhole.webp")
	if err != nil {
		t.Fatal(err)
	}
	defer original.Close()

	createdAt := time.Now()
	expected, err := store.Create(original, "yellhole.webp", createdAt)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := store.Fetch(expected.ID)
	if err != nil {
		t.Fatal(err)
	}
	actual.CreatedAt = createdAt // time.Time doesn't round trip through encoding/json well

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %#v but got %#v", expected, actual)
	}
}

func TestImageStoreRecent(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewImageStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		original, err := os.Open("../../../yellhole.webp")
		if err != nil {
			t.Fatal(err)
		}
		defer original.Close()

		if _, err := store.Create(original, fmt.Sprintf("yellhole-%d.webp", i), time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}

		original.Close()
	}

	recent, err := store.Recent(5)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 5, len(recent); expected != actual {
		t.Fatalf("expected %d items but was %d", expected, actual)
	}

	if expected, actual := "yellhole-9.webp", recent[0].OriginalFilename; expected != actual {
		t.Errorf("expected %q but was %q", expected, actual)
	}

	if expected, actual := "yellhole-8.webp", recent[1].OriginalFilename; expected != actual {
		t.Errorf("expected %q but was %q", expected, actual)
	}

	if expected, actual := "yellhole-7.webp", recent[2].OriginalFilename; expected != actual {
		t.Errorf("expected %q but was %q", expected, actual)
	}

	if expected, actual := "yellhole-6.webp", recent[3].OriginalFilename; expected != actual {
		t.Errorf("expected %q but was %q", expected, actual)
	}

	if expected, actual := "yellhole-5.webp", recent[4].OriginalFilename; expected != actual {
		t.Errorf("expected %q but was %q", expected, actual)
	}
}
