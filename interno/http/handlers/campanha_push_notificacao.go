package handlers

import (
	"context"
	"log"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/notificacoes"
)

// notificarClientesPushNovaCampanha envia FCM (assincrono) quando a campanha fica ativa e valida no app.
func (h *Handlers) notificarClientesPushNovaCampanha(c *modelos.Campanha) {
	if c == nil {
		return
	}
	if c.Status != modelos.StatusCampanhaAtiva || !c.ValidaNoApp {
		return
	}
	idRede := strings.TrimSpace(c.IDRede)
	if idRede == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		tokens, err := h.usuarioRedeService.ListarTokensFCMClientesRede(idRede)
		if err != nil {
			log.Printf("fcm campanha: listar tokens rede %s: %v", idRede, err)
			return
		}
		if len(tokens) == 0 {
			return
		}
		titulo := strings.TrimSpace(c.TituloExibicao)
		if titulo == "" {
			titulo = strings.TrimSpace(c.Titulo)
		}
		if titulo == "" {
			titulo = strings.TrimSpace(c.Nome)
		}
		if titulo == "" {
			titulo = "Nova promocao"
		}
		notificacoes.EnviarNovaCampanhaNoApp(ctx, h.cfg.FcmCaminhoContaServico, tokens, c.ID, titulo, idRede)
	}()
}
