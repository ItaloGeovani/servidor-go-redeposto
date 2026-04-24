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

// ObterEnvSimNao interpreta 1, true, sim, yes, on (minusculo) como verdade; vazio usa [padrao].
func ObterEnvSimNao(chave string, padrao bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(chave)))
	if v == "" {
		return padrao
	}
	return v == "1" || v == "true" || v == "sim" || v == "yes" || v == "on"
}
