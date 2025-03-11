package id

import (
	"slices"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	a := New(time.Date(2025, 3, 10, 10, 2, 0, 0, time.UTC))
	b := New(time.Date(2025, 3, 10, 10, 2, 1, 0, time.UTC))
	c := New(time.Date(2025, 3, 10, 10, 2, 2, 0, time.UTC))

	expected := []string{c, b, a}
	actual := []string{a, b, c}
	slices.Sort(actual)

	if !slices.Equal(expected, actual) {
		t.Errorf("expected %T but was %T", expected, actual)
	}
}
