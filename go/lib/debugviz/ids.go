package debugviz

import (
	"crypto/rand"
	"encoding/hex"
)

func newTraceID() string {
	return newID()
}

func newSpanID() string {
	return newID()
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback for environments without crypto/rand.
		for i := range b {
			b[i] = byte(i + 7)
		}
	}
	return hex.EncodeToString(b[:])
}
