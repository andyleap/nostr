package common

import (
	"crypto/rand"
	"encoding/hex"
)

func RandID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
