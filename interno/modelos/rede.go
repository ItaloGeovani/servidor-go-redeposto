package modelos

import "time"

type Rede struct {
	ID                  string    `json:"id"`
	NomeFantasia        string    `json:"nome_fantasia"`
	RazaoSocial         string    `json:"razao_social"`
	CNPJ                string    `json:"cnpj"`
	EmailContato        string    `json:"email_contato"`
	Telefone            string    `json:"telefone"`
	ValorImplantacao    float64   `json:"valor_implantacao"`
	ValorMensalidade    float64   `json:"valor_mensalidade"`
	PrimeiroCobranca    time.Time `json:"primeiro_cobranca"`
	DiaCobranca         int       `json:"dia_cobranca"`
	MoedaVirtualNome    string    `json:"moeda_virtual_nome"`
	MoedaVirtualCotacao float64   `json:"moeda_virtual_cotacao"`
	// VoucherDiasValidadeResgate dias para usar o saldo no posto após o PIX aprovado.
	VoucherDiasValidadeResgate int `json:"voucher_dias_validade_resgate"`
	// VoucherMinutosExpiraPagamentoPix tempo para pagar a cobrança PIX antes de expirar.
	VoucherMinutosExpiraPagamentoPix int       `json:"voucher_minutos_expira_pagamento_pix"`
	Ativa                            bool      `json:"ativa"`
	CriadoEm                         time.Time `json:"criado_em"`
	AtualizadoEm                     time.Time `json:"atualizado_em"`
}
