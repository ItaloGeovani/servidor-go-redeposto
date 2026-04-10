package modelos

import "time"

// ConfiguracaoAppMobile linha unica com versoes dos apps para checagem de atualizacao.
type ConfiguracaoAppMobile struct {
	VersaoIOS              string    `json:"versao_ios"`
	VersaoAndroid          string    `json:"versao_android"`
	URLLojaIOS             string    `json:"url_loja_ios"`
	URLLojaAndroid         string    `json:"url_loja_android"`
	MensagemAtualizacao    string    `json:"mensagem_atualizacao"`
	AtualizacaoObrigatoria bool      `json:"atualizacao_obrigatoria"`
	AtualizadoEm           time.Time `json:"atualizado_em"`
}
