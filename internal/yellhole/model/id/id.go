package id

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"
)

func New() string {
	return newID(time.Now())
}

func newID(now time.Time) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(now.Unix())<<32)
	if _, err := rand.Read(buf[4:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
