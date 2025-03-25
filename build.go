package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

var buildTag string

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

	buildTag = hex.EncodeToString(h.Sum(nil)[:8])
}
