package modelos

import "time"

// AppCardRede card configuravel para o app (destaque ou promocao).
type AppCardRede struct {
	ID         string    `json:"id,omitempty"`
	IDRede     string    `json:"id_rede,omitempty"`
	Slot       int       `json:"slot"`
	Titulo     string    `json:"titulo"`
	ImagemURL  string    `json:"imagem_url"`
	LinkURL    string    `json:"link_url"`
	Ativo      bool      `json:"ativo"`
	CriadoEm   time.Time `json:"criado_em,omitempty"`
	AtualizadoEm time.Time `json:"atualizado_em,omitempty"`
}
