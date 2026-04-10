package servicos

import (
	"errors"
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
	EditarUsuarioEquipe(in EditarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error)
	LoginPainel(email, senha string) (string, *modelos.UsuarioSessao, error)
	CadastrarClienteApp(in CadastroClienteAppInput) (string, *modelos.UsuarioSessao, error)
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

// CadastroClienteAppInput cadastro público de cliente final (app mobile) na rede.
type CadastroClienteAppInput struct {
	IDRede         string
	NomeCompleto   string
	Email          string
	Senha          string
	ConfirmarSenha string
	Telefone       string
}

// EditarUsuarioEquipeInput atualizacao de gerente ou frentista; senhas vazias mantem a senha atual.
type EditarUsuarioEquipeInput struct {
	IDRede         string
	IDUsuario      string
	IDPosto        string
	Papel          string
	Nome           string
	Email          string
	Senha          string
	ConfirmarSenha string
	Telefone       string
	Ativo          bool
}

var papeisEquipePosto = map[string]struct{}{
	"gerente_posto": {},
	"frentista":     {},
}

type usuarioRedePostgresRepo interface {
	ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, error)
	CriarUsuarioEquipe(idRede, idPosto, papel, nome, email, senhaHash, telefone string) (*modelos.UsuarioVinculoRede, error)
	CriarClienteSelfCadastro(idRede, nome, email, senhaHash, telefone string) (*modelos.UsuarioVinculoRede, error)
	AtualizarUsuarioEquipe(idRede, idUsuario string, nome, email, telefone string, ativo bool, papel, idPosto, senhaHashOuVazio string) (*modelos.UsuarioVinculoRede, error)
	BuscarPorEmailParaLoginPainel(email string) (*repositorios.UsuarioPainelLogin, error)
	PostoPertenceARede(idPosto, idRede string) (bool, error)
}

type servicoUsuarioRede struct {
	repoUsuarios usuarioRedePostgresRepo
	repoRede     repositorios.RedeRepositorio
	auth         *autenticadorToken
}

func NovoServicoUsuarioRede(repoUsuarios usuarioRedePostgresRepo, repoRede repositorios.RedeRepositorio, auth Autenticador) (ServicoUsuarioRede, error) {
	authToken, ok := auth.(*autenticadorToken)
	if !ok {
		return nil, errors.New("autenticador invalido para servico de usuario da rede")
	}
	return &servicoUsuarioRede{
		repoUsuarios: repoUsuarios,
		repoRede:     repoRede,
		auth:         authToken,
	}, nil
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

func (s *servicoUsuarioRede) EditarUsuarioEquipe(in EditarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.IDUsuario = strings.TrimSpace(in.IDUsuario)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	in.Papel = strings.TrimSpace(in.Papel)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Email = strings.TrimSpace(in.Email)
	in.Senha = strings.TrimSpace(in.Senha)
	in.ConfirmarSenha = strings.TrimSpace(in.ConfirmarSenha)
	in.Telefone = strings.TrimSpace(in.Telefone)

	if in.IDRede == "" || in.IDUsuario == "" || in.IDPosto == "" || in.Nome == "" || in.Email == "" || in.Papel == "" {
		return nil, ErrDadosInvalidos
	}
	if _, ok := papeisEquipePosto[in.Papel]; !ok {
		return nil, fmt.Errorf("%w: papel deve ser gerente_posto ou frentista", ErrDadosInvalidos)
	}
	if in.Senha != "" || in.ConfirmarSenha != "" {
		if in.Senha != in.ConfirmarSenha {
			return nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
		}
		if len(in.Senha) < 6 {
			return nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
		}
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return nil, err
	}

	senhaHash := ""
	if in.Senha != "" {
		senhaHash = utils.GerarHashSHA256(in.Senha)
	}
	u, err := s.repoUsuarios.AtualizarUsuarioEquipe(
		in.IDRede,
		in.IDUsuario,
		in.Nome,
		in.Email,
		in.Telefone,
		in.Ativo,
		in.Papel,
		in.IDPosto,
		senhaHash,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *servicoUsuarioRede) LoginPainel(email, senha string) (string, *modelos.UsuarioSessao, error) {
	email = strings.TrimSpace(email)
	senha = strings.TrimSpace(senha)
	if email == "" || senha == "" {
		return "", nil, ErrDadosInvalidos
	}

	u, err := s.repoUsuarios.BuscarPorEmailParaLoginPainel(email)
	if err != nil {
		if errors.Is(err, repositorios.ErrUsuarioPainelLoginNaoEncontrado) {
			return "", nil, ErrCredenciais
		}
		return "", nil, err
	}
	if !u.Ativo || u.SenhaHash != utils.GerarHashSHA256(senha) {
		return "", nil, ErrCredenciais
	}

	p := modelos.Papel(strings.TrimSpace(u.Papel))
	sessao := &modelos.UsuarioSessao{
		IDUsuario:    u.ID,
		NomeCompleto: u.Nome,
		IDRede:       u.IDRede,
		IDPosto:      u.IDPosto,
		Papel:        p,
	}
	token := s.auth.CriarSessao(sessao)
	return token, sessao, nil
}

func (s *servicoUsuarioRede) CadastrarClienteApp(in CadastroClienteAppInput) (string, *modelos.UsuarioSessao, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.NomeCompleto = strings.TrimSpace(in.NomeCompleto)
	in.Email = strings.TrimSpace(in.Email)
	in.Senha = strings.TrimSpace(in.Senha)
	in.ConfirmarSenha = strings.TrimSpace(in.ConfirmarSenha)
	in.Telefone = strings.TrimSpace(in.Telefone)

	if in.IDRede == "" || in.NomeCompleto == "" || in.Email == "" || in.Senha == "" {
		return "", nil, ErrDadosInvalidos
	}
	if in.Senha != in.ConfirmarSenha {
		return "", nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
	}
	if len(in.Senha) < 6 {
		return "", nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return "", nil, err
	}

	u, err := s.repoUsuarios.CriarClienteSelfCadastro(
		in.IDRede,
		in.NomeCompleto,
		in.Email,
		utils.GerarHashSHA256(in.Senha),
		in.Telefone,
	)
	if err != nil {
		return "", nil, err
	}

	sessao := &modelos.UsuarioSessao{
		IDUsuario:    u.ID,
		NomeCompleto: u.Nome,
		IDRede:       u.IDRede,
		IDPosto:      u.IDPosto,
		Papel:        modelos.PapelCliente,
	}
	token := s.auth.CriarSessao(sessao)
	return token, sessao, nil
}
