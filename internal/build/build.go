package build

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// Tag returns the truncated SHA-256 hash of the current executable.
func Tag() (tag string, err error) {
	f, err := os.Open(os.Args[0])
	if err != nil {
		return "", fmt.Errorf("failed to open the current executable: %w", err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read the current executable: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)[:8]), nil
}
