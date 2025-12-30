package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const stateLength = 32

func GenerateState() (string, error) {
	b := make([]byte, stateLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func ValidateState(expected string, received string) bool {
	return expected != "" && expected == received
}
