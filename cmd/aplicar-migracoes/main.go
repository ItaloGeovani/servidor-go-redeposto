package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gaspass-servidor/utils"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := utils.CarregarEnvAutomatico(); err != nil {
		log.Fatalf("falha ao carregar .env: %v", err)
	}

	dsn, err := utils.MontarDSNPostgres()
	if err != nil {
		log.Fatalf("configuracao de banco invalida: %v", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("erro ao abrir conexao: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("falha no ping do banco: %v", err)
	}

	if err := prepararTabelaMigracoes(ctx, db); err != nil {
		log.Fatalf("falha ao preparar tabela de migracoes: %v", err)
	}

	arquivos, err := listarMigracoes("banco/migracoes")
	if err != nil {
		log.Fatalf("falha ao listar migracoes: %v", err)
	}

	if len(arquivos) == 0 {
		log.Fatal("nenhuma migracao encontrada em banco/migracoes")
	}

	for _, arquivo := range arquivos {
		if err := aplicarMigracao(ctx, db, arquivo); err != nil {
			log.Fatalf("falha ao aplicar %s: %v", arquivo, err)
		}
	}

	fmt.Println("Migracoes aplicadas com sucesso.")
}

func prepararTabelaMigracoes(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  id BIGSERIAL PRIMARY KEY,
  nome_arquivo TEXT NOT NULL UNIQUE,
  checksum TEXT NOT NULL,
  aplicada_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

	_, err := db.ExecContext(ctx, ddl)
	return err
}

func listarMigracoes(pasta string) ([]string, error) {
	padrao := filepath.Join(pasta, "*.sql")
	arquivos, err := filepath.Glob(padrao)
	if err != nil {
		return nil, err
	}
	sort.Strings(arquivos)
	return arquivos, nil
}

func aplicarMigracao(ctx context.Context, db *sql.DB, caminhoArquivo string) error {
	conteudo, err := os.ReadFile(caminhoArquivo)
	if err != nil {
		return err
	}

	soma := sha256.Sum256(conteudo)
	checksum := fmt.Sprintf("%x", soma[:])

	var checksumAtual string
	err = db.QueryRowContext(
		ctx,
		"SELECT checksum FROM schema_migrations WHERE nome_arquivo = $1",
		filepath.Base(caminhoArquivo),
	).Scan(&checksumAtual)
	if err == nil {
		if checksumAtual != checksum {
			return fmt.Errorf("migracao %s ja aplicada com checksum diferente", caminhoArquivo)
		}
		fmt.Printf("Ignorando (ja aplicada): %s\n", filepath.Base(caminhoArquivo))
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	fmt.Printf("Aplicando: %s\n", filepath.Base(caminhoArquivo))
	if _, err := tx.ExecContext(ctx, string(conteudo)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (nome_arquivo, checksum) VALUES ($1, $2)",
		filepath.Base(caminhoArquivo),
		checksum,
	); err != nil {
		return err
	}

	return tx.Commit()
}
