package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// newID generates a 128-bit random hex identifier. Adequate for user and
// entity IDs; uniqueness relies on randomness (128 bits).
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}
