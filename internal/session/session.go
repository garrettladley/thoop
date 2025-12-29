package session

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func NewID() string {
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	randomBytes := make([]byte, 3)
	if _, err := rand.Read(randomBytes); err != nil {
		// fallback to nanoseconds if random fails
		return timestamp + "-" + now.Format("000000")
	}

	return timestamp + "-" + hex.EncodeToString(randomBytes)
}
