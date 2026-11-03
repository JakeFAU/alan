package openai

import (
	"crypto/sha256"
	"encoding/hex"
)

// ClientRequestID returns the X-Client-Request-Id the adapter sends for a retained
// effect key. It correlates one request and provides no guarantee of deduplication.
func ClientRequestID(key string) string {
	s := sha256.Sum256([]byte(key))
	return "alan-" + hex.EncodeToString(s[:])
}
