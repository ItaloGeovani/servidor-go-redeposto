package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"gaspass-servidor/interno/app"
	"gaspass-servidor/utils"
)

func main() {
	if err := utils.CarregarEnvAutomatico(); err != nil {
		log.Fatalf("falha ao carregar .env: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	aplicacao, err := app.Nova()
	if err != nil {
		log.Fatalf("falha ao iniciar aplicacao: %v", err)
	}

	if err := aplicacao.Executar(ctx); err != nil {
		log.Fatalf("falha ao executar servidor: %v", err)
	}
}
