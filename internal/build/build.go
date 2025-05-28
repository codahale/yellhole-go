package build

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// Tag is the SHA-256 hash of the binary.
var Tag string

func init() {
	f, err := os.Open(os.Args[0])
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = f.Close()
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		panic(err)
	}

	Tag = hex.EncodeToString(h.Sum(nil)[:8])
}
