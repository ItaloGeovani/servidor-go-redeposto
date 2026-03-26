package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CarregarEnvAutomatico() error {
	caminho, err := LocalizarArquivoEnv()
	if err != nil {
		return err
	}
	return CarregarEnv(caminho)
}

func LocalizarArquivoEnv() (string, error) {
	caminhoAtual, err := os.Getwd()
	if err != nil {
		return "", err
	}

	caminhoBusca := caminhoAtual
	for i := 0; i < 8; i++ {
		candidato := filepath.Join(caminhoBusca, ".env")
		if info, err := os.Stat(candidato); err == nil && !info.IsDir() {
			return candidato, nil
		}

		pai := filepath.Dir(caminhoBusca)
		if pai == caminhoBusca || pai == "." {
			break
		}
		caminhoBusca = pai
	}

	return "", fmt.Errorf("arquivo .env nao encontrado a partir de %s", caminhoAtual)
}

func CarregarEnv(caminho string) error {
	arquivo, err := os.Open(caminho)
	if err != nil {
		return err
	}
	defer arquivo.Close()

	scanner := bufio.NewScanner(arquivo)
	for scanner.Scan() {
		linha := strings.TrimSpace(scanner.Text())
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}

		partes := strings.SplitN(linha, "=", 2)
		if len(partes) != 2 {
			continue
		}

		chave := strings.TrimSpace(partes[0])
		valor := strings.TrimSpace(partes[1])
		valor = strings.Trim(valor, `"'`)

		if os.Getenv(chave) == "" {
			_ = os.Setenv(chave, valor)
		}
	}

	return scanner.Err()
}
