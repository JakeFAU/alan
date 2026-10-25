package openai

import (
	"crypto/sha256"
	"encoding/hex"
)

func clientID(key string) string {
	s := sha256.Sum256([]byte(key))
	return "alan-" + hex.EncodeToString(s[:])
}
