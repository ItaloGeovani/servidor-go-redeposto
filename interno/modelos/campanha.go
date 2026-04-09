package modelos

import "time"

// StatusCampanha espelha o enum status_campanha no Postgres.
type StatusCampanha string

const (
	StatusCampanhaRascunho  StatusCampanha = "RASCUNHO"
	StatusCampanhaAtiva     StatusCampanha = "ATIVA"
	StatusCampanhaPausada   StatusCampanha = "PAUSADA"
	StatusCampanhaArquivada StatusCampanha = "ARQUIVADA"
)

const (
	ModalidadeDescontoNenhum      = "NENHUM"
	ModalidadeDescontoPercentual = "PERCENTUAL"
	ModalidadeDescontoValorFixo   = "VALOR_FIXO"
)

const (
	BaseDescontoValorCompra = "VALOR_COMPRA"
	BaseDescontoLitro       = "LITRO"
	BaseDescontoUnidade     = "UNIDADE"
)

type Campanha struct {
	ID                   string         `json:"id"`
	IDRede               string         `json:"id_rede"`
	Nome                 string         `json:"nome"`
	Titulo               string         `json:"titulo"`
	TituloExibicao       string         `json:"titulo_exibicao"`
	Descricao            string         `json:"descricao"`
	ImagemURL            string         `json:"imagem_url"`
	IDPosto              string         `json:"id_posto"`
	Escopo               string         `json:"escopo"` // "rede" | "posto"
	Status               StatusCampanha `json:"status"`
	VigenciaInicio       *time.Time     `json:"vigencia_inicio,omitempty"`
	VigenciaFim          *time.Time     `json:"vigencia_fim,omitempty"`
	ValidaNoApp          bool           `json:"valida_no_app"`
	ValidaNoPostoFisico  bool           `json:"valida_no_posto_fisico"`
	ModalidadeDesconto   string         `json:"modalidade_desconto"`
	BaseDesconto         string         `json:"base_desconto"`
	ValorDesconto        float64        `json:"valor_desconto"`
	ValorMinimoCompra    float64        `json:"valor_minimo_compra"`
	MaxUsosPorCliente    *int           `json:"max_usos_por_cliente,omitempty"`
	CriadoPor            string         `json:"criado_por"`
	CriadoEm             time.Time      `json:"criado_em"`
	AtualizadoEm         time.Time      `json:"atualizado_em"`
}
