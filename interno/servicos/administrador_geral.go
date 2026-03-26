package servicos

import (
	"errors"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"
)

var (
	ErrDadosInvalidos = errors.New("dados invalidos")
	ErrCredenciais    = errors.New("credenciais invalidas")
)

type ServicoAdministradorGeral interface {
	Criar(nome, email, senha string) (*modelos.AdministradorGeral, error)
	Editar(id, nome, email string, ativo bool) (*modelos.AdministradorGeral, error)
	Login(email, senha string) (string, *modelos.UsuarioSessao, error)
}

type servicoAdministradorGeral struct {
	repo repositorios.AdministradorGeralRepositorio
	auth *autenticadorToken
}

func NovoServicoAdministradorGeral(repo repositorios.AdministradorGeralRepositorio, auth Autenticador) (ServicoAdministradorGeral, error) {
	authToken, ok := auth.(*autenticadorToken)
	if !ok {
		return nil, errors.New("autenticador invalido para servico de administrador")
	}
	return &servicoAdministradorGeral{repo: repo, auth: authToken}, nil
}

func (s *servicoAdministradorGeral) Criar(nome, email, senha string) (*modelos.AdministradorGeral, error) {
	nome = strings.TrimSpace(nome)
	email = strings.TrimSpace(email)
	senha = strings.TrimSpace(senha)

	if nome == "" || email == "" || senha == "" || len(senha) < 6 {
		return nil, ErrDadosInvalidos
	}

	admin := &modelos.AdministradorGeral{
		ID:        utils.GerarToken("adm"),
		Nome:      nome,
		Email:     email,
		SenhaHash: utils.GerarHashSHA256(senha),
		Ativo:     true,
	}

	if err := s.repo.Criar(admin); err != nil {
		return nil, err
	}

	admin.SenhaHash = ""
	return admin, nil
}

func (s *servicoAdministradorGeral) Editar(id, nome, email string, ativo bool) (*modelos.AdministradorGeral, error) {
	id = strings.TrimSpace(id)
	nome = strings.TrimSpace(nome)
	email = strings.TrimSpace(email)

	if id == "" || nome == "" || email == "" {
		return nil, ErrDadosInvalidos
	}

	admin, err := s.repo.Atualizar(id, nome, email, ativo)
	if err != nil {
		return nil, err
	}
	admin.SenhaHash = ""
	return admin, nil
}

func (s *servicoAdministradorGeral) Login(email, senha string) (string, *modelos.UsuarioSessao, error) {
	email = strings.TrimSpace(email)
	senha = strings.TrimSpace(senha)
	if email == "" || senha == "" {
		return "", nil, ErrDadosInvalidos
	}

	admin, err := s.repo.BuscarPorEmail(email)
	if err != nil {
		return "", nil, ErrCredenciais
	}

	if !admin.Ativo || admin.SenhaHash != utils.GerarHashSHA256(senha) {
		return "", nil, ErrCredenciais
	}

	sessao := &modelos.UsuarioSessao{
		IDUsuario:    admin.ID,
		NomeCompleto: admin.Nome,
		IDRede:       "plataforma",
		Papel:        modelos.PapelSuperAdmin,
	}

	token := s.auth.CriarSessao(sessao)
	return token, sessao, nil
}
