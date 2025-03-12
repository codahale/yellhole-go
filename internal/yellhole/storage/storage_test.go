package storage

import (
	"os"
	"testing"
)

type testContext struct {
	t    *testing.T
	tmp  string
	root *os.Root
}

func newTestContext(t *testing.T) *testContext {
	t.Helper()

	tmp, err := os.MkdirTemp("", "yellhole-storage-test")
	if err != nil {
		t.Fatal(err)
	}

	root, err := os.OpenRoot(tmp)
	if err != nil {
		t.Fatal(err)
	}

	return &testContext{t, tmp, root}
}

func (tc *testContext) teardown() {
	tc.t.Helper()

	if err := tc.root.Close(); err != nil {
		tc.t.Errorf("error closing root: %s", err)
	}

	if err := os.RemoveAll(tc.tmp); err != nil {
		tc.t.Errorf("error deleting temp dir: %s", err)
	}
}
