package servicos

import (
	"fmt"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"
)

const (
	usuarioRedeLimitePadrao  = 20
	usuarioRedeLimiteMaximo  = 100
	usuarioRedeLimiteMinimo  = 1
)

type ServicoUsuarioRede interface {
	ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, int, int, error)
	CriarUsuarioEquipe(in CriarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error)
}

// CriarUsuarioEquipeInput cadastro de gerente de posto ou frentista pelo admin global.
type CriarUsuarioEquipeInput struct {
	IDRede         string
	IDPosto        string
	Papel          string
	Nome           string
	Email          string
	Senha          string
	ConfirmarSenha string
	Telefone       string
}

var papeisEquipePosto = map[string]struct{}{
	"gerente_posto": {},
	"frentista":     {},
}

type usuarioRedePostgresRepo interface {
	ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, error)
	CriarUsuarioEquipe(idRede, idPosto, papel, nome, email, senhaHash, telefone string) (*modelos.UsuarioVinculoRede, error)
	PostoPertenceARede(idPosto, idRede string) (bool, error)
}

type servicoUsuarioRede struct {
	repoUsuarios usuarioRedePostgresRepo
	repoRede     repositorios.RedeRepositorio
}

func NovoServicoUsuarioRede(repoUsuarios usuarioRedePostgresRepo, repoRede repositorios.RedeRepositorio) ServicoUsuarioRede {
	return &servicoUsuarioRede{
		repoUsuarios: repoUsuarios,
		repoRede:     repoRede,
	}
}

func (s *servicoUsuarioRede) ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, int, int, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, 0, 0, 0, ErrDadosInvalidos
	}
	if limite < usuarioRedeLimiteMinimo {
		limite = usuarioRedeLimitePadrao
	}
	if limite > usuarioRedeLimiteMaximo {
		limite = usuarioRedeLimiteMaximo
	}
	if offset < 0 {
		offset = 0
	}
	papeis := repositorios.SanitizarPapeisFiltro(papeisFiltro)
	idPostoFiltro = strings.TrimSpace(idPostoFiltro)
	if _, err := s.repoRede.BuscarPorID(idRede); err != nil {
		return nil, 0, 0, 0, err
	}
	itens, total, err := s.repoUsuarios.ListarPorRedeIDPaginado(idRede, limite, offset, papeis, idPostoFiltro)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	return itens, total, limite, offset, nil
}

func (s *servicoUsuarioRede) CriarUsuarioEquipe(in CriarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	in.Papel = strings.TrimSpace(in.Papel)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Email = strings.TrimSpace(in.Email)
	in.Senha = strings.TrimSpace(in.Senha)
	in.ConfirmarSenha = strings.TrimSpace(in.ConfirmarSenha)
	in.Telefone = strings.TrimSpace(in.Telefone)

	if in.IDRede == "" || in.IDPosto == "" || in.Nome == "" || in.Email == "" || in.Senha == "" || in.Papel == "" {
		return nil, ErrDadosInvalidos
	}
	if in.Senha != in.ConfirmarSenha {
		return nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
	}
	if len(in.Senha) < 6 {
		return nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
	}
	if _, ok := papeisEquipePosto[in.Papel]; !ok {
		return nil, fmt.Errorf("%w: papel deve ser gerente_posto ou frentista", ErrDadosInvalidos)
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return nil, err
	}
	ok, err := s.repoUsuarios.PostoPertenceARede(in.IDPosto, in.IDRede)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, repositorios.ErrPostoNaoPertenceARede
	}

	return s.repoUsuarios.CriarUsuarioEquipe(
		in.IDRede,
		in.IDPosto,
		in.Papel,
		in.Nome,
		in.Email,
		utils.GerarHashSHA256(in.Senha),
		in.Telefone,
	)
}
