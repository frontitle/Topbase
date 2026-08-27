package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func HashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func NewAPIKeySecret() (raw, prefix, hash string, err error) {
	buf := make([]byte, 18)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", err
	}
	raw = "tbk_" + hex.EncodeToString(buf)
	if len(raw) < 8 {
		return "", "", "", fmt.Errorf("generated key too short")
	}
	return raw, raw[:8], HashAPIKey(raw), nil
}
