package servicos

import (
	"math"
	"net/url"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

type ServicoCampanha interface {
	ListarPorRedeID(idRede string) ([]*modelos.Campanha, error)
	Criar(sessaoCriador string, in CriarCampanhaInput) (*modelos.Campanha, error)
	Atualizar(in AtualizarCampanhaInput) error
}

type CriarCampanhaInput struct {
	IDRede                string
	Nome                  string
	Titulo                string
	Descricao             string
	ImagemURL             string
	IDPosto               string
	VigenciaInicio        *time.Time
	VigenciaFim           *time.Time
	Status                modelos.StatusCampanha
	ValidaNoApp           bool
	ValidaNoPostoFisico   bool
	ModalidadeDesconto    string
	BaseDesconto          string
	ValorDesconto         float64
	ValorMinimoCompra     float64
	MaxUsosPorCliente     *int
}

type AtualizarCampanhaInput struct {
	ID                  string
	IDRede              string
	Nome                string
	Titulo              string
	Descricao           string
	ImagemURL           string
	IDPosto             string
	VigenciaInicio      *time.Time
	VigenciaFim         *time.Time
	Status              modelos.StatusCampanha
	ValidaNoApp         bool
	ValidaNoPostoFisico bool
	ModalidadeDesconto  string
	BaseDesconto        string
	ValorDesconto       float64
	ValorMinimoCompra   float64
	MaxUsosPorCliente   *int
}

type campanhaRepo interface {
	ListarPorRedeID(idRede string) ([]*modelos.Campanha, error)
	Criar(sessaoCriador string, c *modelos.Campanha) error
	Atualizar(c *modelos.Campanha) error
	PostoPertenceARede(idPosto, idRede string) (bool, error)
}

type servicoCampanha struct {
	repo     campanhaRepo
	repoRede repositorios.RedeRepositorio
}

func NovoServicoCampanha(repo campanhaRepo, repoRede repositorios.RedeRepositorio) ServicoCampanha {
	return &servicoCampanha{repo: repo, repoRede: repoRede}
}

func validarURLImagem(s string) bool {
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

func statusCampanhaValido(s modelos.StatusCampanha) bool {
	switch s {
	case modelos.StatusCampanhaRascunho, modelos.StatusCampanhaAtiva,
		modelos.StatusCampanhaPausada, modelos.StatusCampanhaArquivada:
		return true
	default:
		return false
	}
}

func normalizarModalidade(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case modelos.ModalidadeDescontoNenhum:
		return modelos.ModalidadeDescontoNenhum
	case modelos.ModalidadeDescontoPercentual, modelos.ModalidadeDescontoValorFixo:
		return s
	default:
		return modelos.ModalidadeDescontoNenhum
	}
}

func normalizarBase(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case modelos.BaseDescontoLitro, modelos.BaseDescontoUnidade:
		return s
	default:
		return modelos.BaseDescontoValorCompra
	}
}

func validarDescontoEUso(
	validaApp, validaPosto bool,
	mod, base string,
	valorDesc, vmin float64,
	maxUsos *int,
) bool {
	// Exatamente um canal: app OU posto fisico (mutuamente exclusivos).
	if validaApp == validaPosto {
		return false
	}
	if math.IsNaN(valorDesc) || math.IsInf(valorDesc, 0) {
		return false
	}
	if math.IsNaN(vmin) || math.IsInf(vmin, 0) || vmin < 0 {
		return false
	}
	if maxUsos != nil && *maxUsos < 1 {
		return false
	}
	switch mod {
	case modelos.ModalidadeDescontoNenhum:
		if valorDesc != 0 && math.Abs(valorDesc) > 1e-9 {
			return false
		}
	case modelos.ModalidadeDescontoPercentual:
		if valorDesc <= 0 || valorDesc > 100 {
			return false
		}
	case modelos.ModalidadeDescontoValorFixo:
		if valorDesc <= 0 {
			return false
		}
	default:
		return false
	}
	switch base {
	case modelos.BaseDescontoValorCompra, modelos.BaseDescontoLitro, modelos.BaseDescontoUnidade:
	default:
		return false
	}
	return true
}

func (s *servicoCampanha) ListarPorRedeID(idRede string) ([]*modelos.Campanha, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.repoRede.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	return s.repo.ListarPorRedeID(idRede)
}

func (s *servicoCampanha) Criar(sessaoCriador string, in CriarCampanhaInput) (*modelos.Campanha, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.Descricao = strings.TrimSpace(in.Descricao)
	in.ImagemURL = strings.TrimSpace(in.ImagemURL)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	mod := normalizarModalidade(in.ModalidadeDesconto)
	base := normalizarBase(in.BaseDesconto)
	valor := in.ValorDesconto
	if mod == modelos.ModalidadeDescontoNenhum {
		valor = 0
	}

	if in.IDRede == "" || in.Nome == "" {
		return nil, ErrDadosInvalidos
	}
	if in.VigenciaInicio == nil || in.VigenciaFim == nil {
		return nil, ErrDadosInvalidos
	}
	if in.VigenciaFim.Before(*in.VigenciaInicio) {
		return nil, ErrDadosInvalidos
	}
	if !validarURLImagem(in.ImagemURL) {
		return nil, ErrDadosInvalidos
	}
	if in.Status == "" {
		in.Status = modelos.StatusCampanhaAtiva
	}
	if !statusCampanhaValido(in.Status) {
		return nil, ErrDadosInvalidos
	}
	if !validarDescontoEUso(in.ValidaNoApp, in.ValidaNoPostoFisico, mod, base, valor, in.ValorMinimoCompra, in.MaxUsosPorCliente) {
		return nil, ErrDadosInvalidos
	}

	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return nil, err
	}

	if in.IDPosto != "" {
		ok, err := s.repo.PostoPertenceARede(in.IDPosto, in.IDRede)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, repositorios.ErrPostoNaoPertenceARede
		}
	}

	c := &modelos.Campanha{
		IDRede:               in.IDRede,
		Nome:                 in.Nome,
		Titulo:               in.Titulo,
		Descricao:            in.Descricao,
		ImagemURL:            in.ImagemURL,
		IDPosto:              in.IDPosto,
		Status:               in.Status,
		VigenciaInicio:       in.VigenciaInicio,
		VigenciaFim:          in.VigenciaFim,
		ValidaNoApp:          in.ValidaNoApp,
		ValidaNoPostoFisico:  in.ValidaNoPostoFisico,
		ModalidadeDesconto:   mod,
		BaseDesconto:         base,
		ValorDesconto:        valor,
		ValorMinimoCompra:    in.ValorMinimoCompra,
		MaxUsosPorCliente:    in.MaxUsosPorCliente,
	}
	if err := s.repo.Criar(sessaoCriador, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *servicoCampanha) Atualizar(in AtualizarCampanhaInput) error {
	in.ID = strings.TrimSpace(in.ID)
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.Descricao = strings.TrimSpace(in.Descricao)
	in.ImagemURL = strings.TrimSpace(in.ImagemURL)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	mod := normalizarModalidade(in.ModalidadeDesconto)
	base := normalizarBase(in.BaseDesconto)
	valor := in.ValorDesconto
	if mod == modelos.ModalidadeDescontoNenhum {
		valor = 0
	}

	if in.ID == "" || in.IDRede == "" || in.Nome == "" {
		return ErrDadosInvalidos
	}
	if in.VigenciaInicio == nil || in.VigenciaFim == nil {
		return ErrDadosInvalidos
	}
	if in.VigenciaFim.Before(*in.VigenciaInicio) {
		return ErrDadosInvalidos
	}
	if in.Status == "" {
		return ErrDadosInvalidos
	}
	if !validarURLImagem(in.ImagemURL) {
		return ErrDadosInvalidos
	}
	if !statusCampanhaValido(in.Status) {
		return ErrDadosInvalidos
	}
	if !validarDescontoEUso(in.ValidaNoApp, in.ValidaNoPostoFisico, mod, base, valor, in.ValorMinimoCompra, in.MaxUsosPorCliente) {
		return ErrDadosInvalidos
	}

	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return err
	}

	if in.IDPosto != "" {
		ok, err := s.repo.PostoPertenceARede(in.IDPosto, in.IDRede)
		if err != nil {
			return err
		}
		if !ok {
			return repositorios.ErrPostoNaoPertenceARede
		}
	}

	c := &modelos.Campanha{
		ID:                   in.ID,
		IDRede:               in.IDRede,
		Nome:                 in.Nome,
		Titulo:               in.Titulo,
		Descricao:            in.Descricao,
		ImagemURL:            in.ImagemURL,
		IDPosto:              in.IDPosto,
		Status:               in.Status,
		VigenciaInicio:       in.VigenciaInicio,
		VigenciaFim:          in.VigenciaFim,
		ValidaNoApp:          in.ValidaNoApp,
		ValidaNoPostoFisico:  in.ValidaNoPostoFisico,
		ModalidadeDesconto:   mod,
		BaseDesconto:         base,
		ValorDesconto:        valor,
		ValorMinimoCompra:    in.ValorMinimoCompra,
		MaxUsosPorCliente:    in.MaxUsosPorCliente,
	}
	return s.repo.Atualizar(c)
}
