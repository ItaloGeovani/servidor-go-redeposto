package repositorios

import (
	"errors"
	"time"
)

var ErrCombustivelRedeNaoEncontrado = errors.New("combustivel nao encontrado nesta rede")

// CombustivelRedeRegistro linha em rede_combustiveis.
type CombustivelRedeRegistro struct {
	ID             string    `json:"id"`
	RedeID         string    `json:"id_rede"`
	Nome           string    `json:"nome"`
	Codigo         string    `json:"codigo"`
	Descricao      string    `json:"descricao"`
	PrecoPorLitro  float64   `json:"preco_por_litro"`
	Ativo          bool      `json:"ativo"`
	Ordem          int       `json:"ordem"`
	CriadoEm       time.Time `json:"criado_em"`
	AtualizadoEm   time.Time `json:"atualizado_em"`
}

// CombustivelRedeRepositorio CRUD de combustíveis da rede.
type CombustivelRedeRepositorio interface {
	ListarPorRede(redeID string) ([]*CombustivelRedeRegistro, error)
	BuscarPorID(id, redeID string) (*CombustivelRedeRegistro, error)
	Criar(x *CombustivelRedeRegistro) error
	Atualizar(id, redeID string, atualizar func(*CombustivelRedeRegistro) error) (*CombustivelRedeRegistro, error)
	Excluir(id, redeID string) error
}
