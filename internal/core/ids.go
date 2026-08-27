package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func NewID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf))
}
