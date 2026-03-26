package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GerarToken(prefixo string) string {
	buf := make([]byte, 24)
	_, _ = rand.Read(buf)
	return prefixo + "_" + hex.EncodeToString(buf)
}
