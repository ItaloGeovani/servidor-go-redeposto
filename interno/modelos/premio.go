package modelos

import "time"

type Premio struct {
	ID                   string     `json:"id"`
	IDRede               string     `json:"id_rede"`
	Titulo               string     `json:"titulo"`
	ImagemURL            string     `json:"imagem_url"`
	ValorMoeda           float64    `json:"valor_moeda"`
	Ativo                bool       `json:"ativo"`
	VigenciaInicio       time.Time  `json:"vigencia_inicio"`
	VigenciaFim          *time.Time `json:"vigencia_fim,omitempty"`
	QuantidadeDisponivel *int       `json:"quantidade_disponivel,omitempty"`
	CriadoEm             time.Time  `json:"criado_em"`
	AtualizadoEm         time.Time  `json:"atualizado_em"`
}
