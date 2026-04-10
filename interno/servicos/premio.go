package servicos

import (
	"math"
	"net/url"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

type ServicoPremio interface {
	ListarPorRedeID(idRede string) ([]*modelos.Premio, error)
	Criar(in CriarPremioInput) (*modelos.Premio, error)
	Atualizar(in AtualizarPremioInput) error
}

type CriarPremioInput struct {
	IDRede               string
	Titulo               string
	ImagemURL            string
	ValorMoeda           float64
	Ativo                bool
	VigenciaInicio       *time.Time
	VigenciaFim          *time.Time
	QuantidadeDisponivel *int
}

type AtualizarPremioInput struct {
	ID                   string
	IDRede               string
	Titulo               string
	ImagemURL            string
	ValorMoeda           float64
	Ativo                bool
	VigenciaInicio       *time.Time
	VigenciaFim          *time.Time
	QuantidadeDisponivel *int
}

type premioRepo interface {
	ListarPorRedeID(idRede string) ([]*modelos.Premio, error)
	Criar(p *modelos.Premio) error
	Atualizar(p *modelos.Premio) error
}

type servicoPremio struct {
	repo     premioRepo
	repoRede repositorios.RedeRepositorio
}

func NovoServicoPremio(repo premioRepo, repoRede repositorios.RedeRepositorio) ServicoPremio {
	return &servicoPremio{repo: repo, repoRede: repoRede}
}

func validarURLImagemPremio(s string) bool {
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

func (s *servicoPremio) ListarPorRedeID(idRede string) ([]*modelos.Premio, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.repoRede.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	return s.repo.ListarPorRedeID(idRede)
}

func (s *servicoPremio) Criar(in CriarPremioInput) (*modelos.Premio, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.ImagemURL = strings.TrimSpace(in.ImagemURL)
	if in.IDRede == "" || in.Titulo == "" {
		return nil, ErrDadosInvalidos
	}
	if in.VigenciaInicio == nil {
		return nil, ErrDadosInvalidos
	}
	if in.VigenciaFim != nil && in.VigenciaFim.Before(*in.VigenciaInicio) {
		return nil, ErrDadosInvalidos
	}
	if math.IsNaN(in.ValorMoeda) || math.IsInf(in.ValorMoeda, 0) || in.ValorMoeda <= 0 {
		return nil, ErrDadosInvalidos
	}
	if in.QuantidadeDisponivel != nil && *in.QuantidadeDisponivel < 0 {
		return nil, ErrDadosInvalidos
	}
	if !validarURLImagemPremio(in.ImagemURL) {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return nil, err
	}
	p := &modelos.Premio{
		IDRede:               in.IDRede,
		Titulo:               in.Titulo,
		ImagemURL:            in.ImagemURL,
		ValorMoeda:           in.ValorMoeda,
		Ativo:                in.Ativo,
		VigenciaInicio:       *in.VigenciaInicio,
		VigenciaFim:          in.VigenciaFim,
		QuantidadeDisponivel: in.QuantidadeDisponivel,
	}
	if err := s.repo.Criar(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *servicoPremio) Atualizar(in AtualizarPremioInput) error {
	in.ID = strings.TrimSpace(in.ID)
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.ImagemURL = strings.TrimSpace(in.ImagemURL)
	if in.ID == "" || in.IDRede == "" || in.Titulo == "" {
		return ErrDadosInvalidos
	}
	if in.VigenciaInicio == nil {
		return ErrDadosInvalidos
	}
	if in.VigenciaFim != nil && in.VigenciaFim.Before(*in.VigenciaInicio) {
		return ErrDadosInvalidos
	}
	if math.IsNaN(in.ValorMoeda) || math.IsInf(in.ValorMoeda, 0) || in.ValorMoeda <= 0 {
		return ErrDadosInvalidos
	}
	if in.QuantidadeDisponivel != nil && *in.QuantidadeDisponivel < 0 {
		return ErrDadosInvalidos
	}
	if !validarURLImagemPremio(in.ImagemURL) {
		return ErrDadosInvalidos
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return err
	}
	p := &modelos.Premio{
		ID:                   in.ID,
		IDRede:               in.IDRede,
		Titulo:               in.Titulo,
		ImagemURL:            in.ImagemURL,
		ValorMoeda:           in.ValorMoeda,
		Ativo:                in.Ativo,
		VigenciaInicio:       *in.VigenciaInicio,
		VigenciaFim:          in.VigenciaFim,
		QuantidadeDisponivel: in.QuantidadeDisponivel,
	}
	return s.repo.Atualizar(p)
}
