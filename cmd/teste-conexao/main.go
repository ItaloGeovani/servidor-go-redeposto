package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("falha no ping do banco: %v", err)
	}

	var versao string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&versao); err != nil {
		log.Fatalf("conexao OK, mas falhou ao consultar versao: %v", err)
	}

	fmt.Println("Conexao com PostgreSQL estabelecida com sucesso.")
	fmt.Printf("Versao do servidor: %s\n", versao)
}
