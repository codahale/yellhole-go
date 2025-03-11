package id

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"math"
	"time"
)

func New(now time.Time) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(now.Unix())<<32)
	if _, err := rand.Read(buf[4:]); err != nil {
		panic(err)
	}
	binary.BigEndian.PutUint64(buf, math.MaxUint64-binary.BigEndian.Uint64(buf))
	return hex.EncodeToString(buf)
}
