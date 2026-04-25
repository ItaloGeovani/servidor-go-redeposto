package servicos

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gaspass-servidor/interno/repositorios"
)

// NiveisCliente resposta API (gestor + public).
type NiveisCliente struct {
	Ativo             bool                      `json:"ativo"`
	MultDescontoAtivo bool                      `json:"mult_desconto_ativo"`
	Niveis            []repositorios.NivelClienteLinha `json:"niveis"`
}

var reCodigoNivel = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// NiveisPadraoQuatro niveis sugeridos quando ainda nao ha registo na base.
func NiveisPadraoQuatro() []repositorios.NivelClienteLinha {
	return []repositorios.NivelClienteLinha{
		{Codigo: "bronze", Nome: "Bronze", MultMoeda: 1, MultDesconto: 1, Ordem: 1},
		{Codigo: "prata", Nome: "Prata", MultMoeda: 1.1, MultDesconto: 1.1, Ordem: 2},
		{Codigo: "ouro", Nome: "Ouro", MultMoeda: 1.2, MultDesconto: 1.2, Ordem: 3},
		{Codigo: "diamante", Nome: "Diamante", MultMoeda: 1.3, MultDesconto: 1.3, Ordem: 4},
	}
}

// ServicoNiveisCliente regras e leitura da config de niveis.
type ServicoNiveisCliente struct {
	repo repositorios.NiveisClienteRepositorio
}

func NovoServicoNiveisCliente(repo repositorios.NiveisClienteRepositorio) *ServicoNiveisCliente {
	return &ServicoNiveisCliente{repo: repo}
}

// Buscar retorna a config persistida ou padrao (ativo=false) se ainda nao existir linha.
func (s *ServicoNiveisCliente) Buscar(redeID string) (*NiveisCliente, error) {
	if s.repo == nil {
		return nil, errors.New("repo indisponivel")
	}
	cfg, err := s.repo.Buscar(redeID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &NiveisCliente{
			Ativo:             false,
			MultDescontoAtivo: false,
			Niveis:            NiveisPadraoQuatro(),
		}, nil
	}
	n := cfg.Niveis
	if len(n) == 0 {
		n = NiveisPadraoQuatro()
	}
	return &NiveisCliente{
		Ativo:             cfg.Ativo,
		MultDescontoAtivo: cfg.MultDescontoAtivo,
		Niveis:            n,
	}, nil
}

// Salvar valida e grava.
func (s *ServicoNiveisCliente) Salvar(redeID string, ativo, multDesc bool, niveis []repositorios.NivelClienteLinha) error {
	if s.repo == nil {
		return errors.New("repo indisponivel")
	}
	if err := validarNiveis(niveis); err != nil {
		return fmt.Errorf("%w: %v", ErrDadosInvalidos, err)
	}
	for i := range niveis {
		niveis[i].Codigo = strings.ToLower(strings.TrimSpace(niveis[i].Codigo))
		niveis[i].Nome = strings.TrimSpace(niveis[i].Nome)
	}
	return s.repo.Salvar(redeID, ativo, multDesc, niveis)
}

func validarNiveis(niveis []repositorios.NivelClienteLinha) error {
	if len(niveis) < 1 || len(niveis) > 8 {
		return errors.New("informe entre 1 e 8 niveis")
	}
	cods := map[string]struct{}{}
	ordens := map[int]struct{}{}
	for _, ni := range niveis {
		cod := strings.ToLower(strings.TrimSpace(ni.Codigo))
		if !reCodigoNivel.MatchString(cod) {
			return errors.New("codigo de nivel: apenas a-z, 0-9 e _, comecando com letra (max 32)")
		}
		nome := strings.TrimSpace(ni.Nome)
		if nome == "" {
			return errors.New("cada nivel precisa de nome")
		}
		if ni.MultMoeda < 1.0-1e-9 {
			return errors.New("mult_moeda deve ser >= 1")
		}
		if ni.MultDesconto < 1.0-1e-9 {
			return errors.New("mult_desconto deve ser >= 1")
		}
		if ni.Ordem < 1 || ni.Ordem > 99 {
			return errors.New("ordem deve ser entre 1 e 99")
		}
		if _, ok := cods[cod]; ok {
			return errors.New("codigos de nivel duplicados")
		}
		cods[cod] = struct{}{}
		if _, ok := ordens[ni.Ordem]; ok {
			return errors.New("ordem duplicada entre niveis")
		}
		ordens[ni.Ordem] = struct{}{}
	}
	return nil
}

// FatorMultMoeda retorna o multiplicador de moeda para o codigo de nivel, ou 1 se desligado / codigo vazio.
func (s *ServicoNiveisCliente) FatorMultMoeda(redeID, codigoNivel string) float64 {
	c, err := s.Buscar(redeID)
	if err != nil || c == nil || !c.Ativo {
		return 1
	}
	return fatorParaCodigo(c.Niveis, strings.ToLower(strings.TrimSpace(codigoNivel)), true)
}

// FatorMultDesconto idem para desconto; se mult_desconto_ativo false, 1.
func (s *ServicoNiveisCliente) FatorMultDesconto(redeID, codigoNivel string) float64 {
	c, err := s.Buscar(redeID)
	if err != nil || c == nil || !c.Ativo || !c.MultDescontoAtivo {
		return 1
	}
	return fatorParaCodigo(c.Niveis, strings.ToLower(strings.TrimSpace(codigoNivel)), false)
}

func fatorParaCodigo(niveis []repositorios.NivelClienteLinha, cod string, moeda bool) float64 {
	if cod == "" {
		cod = "bronze"
	}
	for _, ni := range niveis {
		if strings.ToLower(strings.TrimSpace(ni.Codigo)) == cod {
			if moeda {
				return ni.MultMoeda
			}
			return ni.MultDesconto
		}
	}
	// codigo desconhecido: baseline bronze/ primeiro da lista
	if len(niveis) > 0 {
		if moeda {
			return niveis[0].MultMoeda
		}
		return niveis[0].MultDesconto
	}
	return 1
}
