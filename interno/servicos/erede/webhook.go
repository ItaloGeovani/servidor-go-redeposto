package erede

import (
	"encoding/json"
	"errors"
	"strings"
)

// Tipos de evento do webhook PIX e.Rede.
const (
	WebhookEventoIgnorado  = "ignorado"
	WebhookEventoPago      = "pago"
	WebhookEventoDevolucao = "devolucao"
)

// WebhookPayload notificação Pix e.Rede.
type WebhookPayload struct {
	CompanyNumber string   `json:"companyNumber"`
	Events        []string `json:"events"`
	Data          struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ParseWebhookResult resultado do parse do webhook PIX.
type ParseWebhookResult struct {
	TID    string
	Tipo   string // pago | devolucao | ignorado
}

// ParseWebhook extrai tid e classifica o evento (pago / devolução / ignorado).
func ParseWebhook(body []byte) (ParseWebhookResult, error) {
	out := ParseWebhookResult{Tipo: WebhookEventoIgnorado}
	if len(body) == 0 {
		return out, errors.New("corpo vazio")
	}
	var p WebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return out, err
	}
	out.TID = strings.TrimSpace(p.Data.ID)
	if out.TID == "" {
		return out, errors.New("webhook sem data.id (tid)")
	}
	temPago := false
	temDevolucao := false
	for _, ev := range p.Events {
		switch strings.TrimSpace(ev) {
		case "PV.UPDATE_TRANSACTION_PIX":
			temPago = true
		case "PV.REFUND_PIX":
			temDevolucao = true
		}
	}
	// Devolução tem prioridade se ambos vierem no mesmo payload.
	if temDevolucao {
		out.Tipo = WebhookEventoDevolucao
		return out, nil
	}
	if temPago {
		out.Tipo = WebhookEventoPago
		return out, nil
	}
	return out, nil
}
