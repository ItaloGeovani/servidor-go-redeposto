package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func GerarHashSHA256(texto string) string {
	soma := sha256.Sum256([]byte(texto))
	return hex.EncodeToString(soma[:])
}
