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
		log.Printf("fcm campanha: skip (campanha nil)")
		return
	}
	log.Printf("fcm campanha: verificar push campanha_id=%s status=%s valida_no_app=%v id_rede=%s", c.ID, c.Status, c.ValidaNoApp, c.IDRede)
	if c.Status != modelos.StatusCampanhaAtiva {
		log.Printf("fcm campanha: nao enviado (so campanha ATIVA dispara; criou RASCUNHO/PAUSADA/ARQUIVADA? status=%s)", c.Status)
		return
	}
	if !c.ValidaNoApp {
		log.Printf("fcm campanha: nao enviado (valida_no_app=false)")
		return
	}
	idRede := strings.TrimSpace(c.IDRede)
	if idRede == "" {
		log.Printf("fcm campanha: nao enviado (rede vazia)")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if strings.TrimSpace(h.cfg.FcmCaminhoContaServico) == "" {
			log.Printf("fcm campanha: nao enviado (defina FCM_SERVICE_ACCOUNT_PATH no .env e reinicie o servidor)")
			return
		}
		tokens, err := h.usuarioRedeService.ListarTokensFCMClientesRede(idRede)
		if err != nil {
			log.Printf("fcm campanha: listar tokens rede %s: %v", idRede, err)
			return
		}
		if len(tokens) == 0 {
			log.Printf("fcm campanha: nao enviado (0 tokens FCM para clientes ativos desta rede id_rede=%s; app cliente precisa de login e POST /v1/eu/push/fcm na mesma rede)", idRede)
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
		log.Printf("fcm campanha: a enviar para %d token(s) rede=%s", len(tokens), idRede)
		notificacoes.EnviarNovaCampanhaNoApp(ctx, h.cfg.FcmCaminhoContaServico, tokens, c.ID, titulo, idRede)
	}()
}
