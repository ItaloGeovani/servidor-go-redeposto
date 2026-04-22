package servicos

import (
	"strings"

	"gaspass-servidor/interno/repositorios"
)

// ServicoCombustivelRede catálogo de combustíveis e preço por litro (referência da rede).
type ServicoCombustivelRede struct {
	repo     repositorios.CombustivelRedeRepositorio
	redeRepo repositorios.RedeRepositorio
}

func NovoServicoCombustivelRede(
	repo repositorios.CombustivelRedeRepositorio,
	rede repositorios.RedeRepositorio,
) *ServicoCombustivelRede {
	return &ServicoCombustivelRede{repo: repo, redeRepo: rede}
}

type CriarCombustivelRedeInput struct {
	Nome          string
	Codigo        string
	Descricao     string
	PrecoPorLitro float64
	Ordem         int
	Ativo         bool
}

type AtualizarCombustivelRedeInput struct {
	ID            string
	Nome          string
	Codigo        string
	Descricao     string
	PrecoPorLitro float64
	Ordem         int
	Ativo         bool
}

func (s *ServicoCombustivelRede) Listar(idRede string) ([]*repositorios.CombustivelRedeRegistro, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.redeRepo.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	return s.repo.ListarPorRede(idRede)
}

func (s *ServicoCombustivelRede) Criar(idRede string, in CriarCombustivelRedeInput) (*repositorios.CombustivelRedeRegistro, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	nome := strings.TrimSpace(in.Nome)
	if nome == "" {
		return nil, ErrDadosInvalidos
	}
	if in.PrecoPorLitro < 0 {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.redeRepo.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	reg := &repositorios.CombustivelRedeRegistro{
		RedeID:        idRede,
		Nome:          nome,
		Codigo:        strings.TrimSpace(in.Codigo),
		Descricao:     strings.TrimSpace(in.Descricao),
		PrecoPorLitro: in.PrecoPorLitro,
		Ativo:         in.Ativo,
		Ordem:         in.Ordem,
	}
	if err := s.repo.Criar(reg); err != nil {
		if strings.Contains(err.Error(), "ja existe") {
			return nil, err
		}
		return nil, err
	}
	return reg, nil
}

func (s *ServicoCombustivelRede) Atualizar(idRede string, in AtualizarCombustivelRedeInput) (*repositorios.CombustivelRedeRegistro, error) {
	idRede = strings.TrimSpace(idRede)
	id := strings.TrimSpace(in.ID)
	if idRede == "" || id == "" {
		return nil, ErrDadosInvalidos
	}
	nome := strings.TrimSpace(in.Nome)
	if nome == "" {
		return nil, ErrDadosInvalidos
	}
	if in.PrecoPorLitro < 0 {
		return nil, ErrDadosInvalidos
	}
	return s.repo.Atualizar(id, idRede, func(r *repositorios.CombustivelRedeRegistro) error {
		r.Nome = nome
		r.Codigo = strings.TrimSpace(in.Codigo)
		r.Descricao = strings.TrimSpace(in.Descricao)
		r.PrecoPorLitro = in.PrecoPorLitro
		r.Ordem = in.Ordem
		r.Ativo = in.Ativo
		return nil
	})
}

func (s *ServicoCombustivelRede) Excluir(idCombustivel, idRede string) error {
	idCombustivel = strings.TrimSpace(idCombustivel)
	idRede = strings.TrimSpace(idRede)
	if idCombustivel == "" || idRede == "" {
		return ErrDadosInvalidos
	}
	return s.repo.Excluir(idCombustivel, idRede)
}
