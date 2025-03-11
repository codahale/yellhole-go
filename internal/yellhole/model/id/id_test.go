package id

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	a := newID(time.Date(2025, 3, 10, 10, 2, 0, 0, time.UTC))
	b := newID(time.Date(2025, 3, 10, 10, 2, 1, 0, time.UTC))
	c := newID(time.Date(2025, 3, 10, 10, 2, 2, 0, time.UTC))

	if !(a < b && b < c) {
		t.Errorf("unexpected ordering of %q, %q, %q", a, b, c)
	}

}
