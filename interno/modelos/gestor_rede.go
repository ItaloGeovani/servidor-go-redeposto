package modelos

import "time"

type GestorRede struct {
	ID                 string    `json:"id"`
	IDRede             string    `json:"id_rede"`
	Nome               string    `json:"nome"`
	Email              string    `json:"email"`
	SenhaHash          string    `json:"-"`
	NovaSenhaHash      string    `json:"-"`
	Telefone           string    `json:"telefone"`
	Ativo              bool      `json:"ativo"`
	ValorImplantacao   float64   `json:"valor_implantacao"`
	ValorMensalidade   float64   `json:"valor_mensalidade"`
	PrimeiroVencimento time.Time `json:"primeiro_vencimento"`
	DiaVencimento      int       `json:"dia_vencimento"`
	CriadoEm           time.Time `json:"criado_em"`
	AtualizadoEm       time.Time `json:"atualizado_em"`
}
