package issue

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// RandHex returns a random lowercase hex string of length n*2 bytes.
func RandHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand.Read on modern OSes does not fail
	}
	return hex.EncodeToString(b)
}

// Timestamp returns the basic-form ISO 8601 UTC timestamp used in filenames
// and as the leading portion of an issue ID, e.g. "20260427T143022Z".
func Timestamp(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// RandHexLen is the number of hex characters in the random suffix of an ID.
// 8 hex = 32 bits of entropy: collisions become possible (~50%) around 65k
// issues, but 4-character prefixes remain unique well past typical issue
// counts, which is what matters for git-style short refs.
const RandHexLen = 8

// NewID returns "<timestamp>-<8-hex>", e.g. "20260427T143022Z-9f2a4b7c".
func NewID(t time.Time) string {
	return Timestamp(t) + "-" + RandHex(RandHexLen/2)
}
