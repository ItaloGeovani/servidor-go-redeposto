package utils

import (
	"fmt"
	"strings"
)

func MontarDSNPostgres() (string, error) {
	if url := strings.TrimSpace(ObterEnv("DATABASE_URL", "")); url != "" {
		return url, nil
	}

	host := strings.TrimSpace(ObterEnv("DB_HOST", ""))
	port := strings.TrimSpace(ObterEnv("DB_PORT", ""))
	nome := strings.TrimSpace(ObterEnv("DB_NOME", ""))
	usuario := strings.TrimSpace(ObterEnv("DB_USUARIO", ""))
	senha := strings.TrimSpace(ObterEnv("DB_SENHA", ""))
	sslMode := strings.TrimSpace(ObterEnv("DB_SSLMODE", "disable"))

	if host == "" || port == "" || nome == "" || usuario == "" || senha == "" {
		return "", fmt.Errorf("defina DATABASE_URL ou DB_HOST/DB_PORT/DB_NOME/DB_USUARIO/DB_SENHA")
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		usuario,
		senha,
		host,
		port,
		nome,
		sslMode,
	), nil
}
