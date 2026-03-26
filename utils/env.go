package utils

import (
	"os"
	"strconv"
	"strings"
)

func ObterEnv(chave, padrao string) string {
	valor := strings.TrimSpace(os.Getenv(chave))
	if valor == "" {
		return padrao
	}
	return valor
}

func ObterEnvInt(chave string, padrao int) int {
	valor := strings.TrimSpace(os.Getenv(chave))
	if valor == "" {
		return padrao
	}

	conv, err := strconv.Atoi(valor)
	if err != nil {
		return padrao
	}
	return conv
}
