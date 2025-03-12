package storage

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestNoteStoreCreateAndFetch(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	createdAt := time.Now()
	expected, err := store.Create("Hello world.", createdAt)
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

func TestNoteStoreRecent(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	recent, err := store.Recent(5)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 5, len(recent); expected != actual {
		t.Fatalf("expected %d items but was %d", expected, actual)
	}
}

func TestNoteStoreYear(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	year, err := store.Year(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 5)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 5, len(year); expected != actual {
		t.Fatalf("expected %d items but was %d", expected, actual)
	}
}

func TestNoteStoreYears(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	years, err := store.Years(10)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := []string{"2025"}, years; !slices.Equal(expected, actual) {
		t.Fatalf("expected %#v items but was %#v", expected, actual)
	}
}

func TestNoteStoreMonth(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	month, err := store.Month(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), 5)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 5, len(month); expected != actual {
		t.Fatalf("expected %d items but was %d", expected, actual)
	}
}

func TestNoteStoreMonths(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	months, err := store.Months(10)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := []string{"2025-03"}, months; !slices.Equal(expected, actual) {
		t.Fatalf("expected %#v items but was %#v", expected, actual)
	}
}

func TestNoteStoreWeek(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	week, err := store.Week(time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC), 5)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 5, len(week); expected != actual {
		t.Fatalf("expected %d items but was %d", expected, actual)
	}
}

func TestNoteStoreWeeks(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	weeks, err := store.Weeks(10)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := []string{"2025-11"}, weeks; !slices.Equal(expected, actual) {
		t.Fatalf("expected %#v items but was %#v", expected, actual)
	}
}

func TestNoteStoreDay(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	day, err := store.Day(time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC), 5)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 5, len(day); expected != actual {
		t.Fatalf("expected %d items but was %d", expected, actual)
	}
}

func TestNoteStoreDays(t *testing.T) {
	tc := newTestContext(t)
	defer tc.teardown()

	store, err := NewNoteStore(tc.root)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 10 {
		if _, err := store.Create("Hello world.", time.Date(2025, 3, 10, 10, 2, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}

	days, err := store.Days(10)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := []string{"2025-03-10"}, days; !slices.Equal(expected, actual) {
		t.Fatalf("expected %#v items but was %#v", expected, actual)
	}
}
