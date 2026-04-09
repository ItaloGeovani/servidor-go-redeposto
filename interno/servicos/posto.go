package servicos

import (
	"net/url"
	"strings"
	"unicode"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

type ServicoPosto interface {
	ListarPorRedeID(idRede string) ([]*modelos.Posto, error)
	CriarPostoNaRede(p *modelos.Posto) (*modelos.Posto, error)
}

type postoRepositorio interface {
	ListarPorRedeID(idRede string) ([]*modelos.Posto, error)
	Criar(p *modelos.Posto) error
}

type servicoPosto struct {
	repoPosto postoRepositorio
	repoRede  repositorios.RedeRepositorio
}

func NovoServicoPosto(repoPosto postoRepositorio, repoRede repositorios.RedeRepositorio) ServicoPosto {
	return &servicoPosto{
		repoPosto: repoPosto,
		repoRede:  repoRede,
	}
}

func (s *servicoPosto) ListarPorRedeID(idRede string) ([]*modelos.Posto, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.repoRede.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	return s.repoPosto.ListarPorRedeID(idRede)
}

func apenasDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validarURLLogo(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *servicoPosto) CriarPostoNaRede(p *modelos.Posto) (*modelos.Posto, error) {
	if p == nil {
		return nil, ErrDadosInvalidos
	}
	p.IDRede = strings.TrimSpace(p.IDRede)
	p.Nome = strings.TrimSpace(p.Nome)
	p.Codigo = strings.TrimSpace(p.Codigo)
	p.NomeFantasia = strings.TrimSpace(p.NomeFantasia)
	p.CNPJ = apenasDigitos(p.CNPJ)
	p.LogoURL = strings.TrimSpace(p.LogoURL)
	p.Rua = strings.TrimSpace(p.Rua)
	p.Numero = strings.TrimSpace(p.Numero)
	p.Bairro = strings.TrimSpace(p.Bairro)
	p.Complemento = strings.TrimSpace(p.Complemento)
	p.CEP = apenasDigitos(p.CEP)
	p.Cidade = strings.TrimSpace(p.Cidade)
	p.Estado = strings.ToUpper(strings.TrimSpace(p.Estado))
	p.Telefone = strings.TrimSpace(p.Telefone)
	p.EmailContato = strings.TrimSpace(p.EmailContato)

	if p.IDRede == "" || p.Nome == "" || p.Codigo == "" {
		return nil, ErrDadosInvalidos
	}
	if len(p.CNPJ) != 0 && len(p.CNPJ) != 14 {
		return nil, ErrDadosInvalidos
	}
	if len(p.CEP) != 0 && len(p.CEP) != 8 {
		return nil, ErrDadosInvalidos
	}
	if p.Estado != "" && len(p.Estado) != 2 {
		return nil, ErrDadosInvalidos
	}
	if !validarURLLogo(p.LogoURL) {
		return nil, ErrDadosInvalidos
	}

	if _, err := s.repoRede.BuscarPorID(p.IDRede); err != nil {
		return nil, err
	}

	if err := s.repoPosto.Criar(p); err != nil {
		return nil, err
	}
	return p, nil
}
