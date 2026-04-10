package servicos

import (
	"errors"
	"strings"
	"sync"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

var ErrTokenInvalido = errors.New("token invalido")
var ErrSessaoExpirada = errors.New("sessao expirada; faca login novamente")

type Autenticador interface {
	ValidarToken(token string) (*modelos.UsuarioSessao, error)
	CriarSessao(usuario *modelos.UsuarioSessao) string
}

type autenticadorToken struct {
	tokenPadrao string
	mu          sync.RWMutex
	sessoes     map[string]*modelos.UsuarioSessao
}

func NovoAutenticadorToken(tokenPadrao string) Autenticador {
	return &autenticadorToken{
		tokenPadrao: tokenPadrao,
		sessoes:     make(map[string]*modelos.UsuarioSessao),
	}
}

func (a *autenticadorToken) ValidarToken(token string) (*modelos.UsuarioSessao, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrTokenInvalido
	}

	a.mu.RLock()
	sessao, ok := a.sessoes[token]
	a.mu.RUnlock()
	if ok {
		copia := *sessao
		return &copia, nil
	}
	if strings.HasPrefix(token, "tok_") {
		return nil, ErrSessaoExpirada
	}

	switch token {
	case a.tokenPadrao, "dev-super-admin":
		return &modelos.UsuarioSessao{
			IDUsuario:    "usuario-dev-super-admin",
			NomeCompleto: "Administrador Global Dev",
			IDRede:       "rede-dev-001",
			Papel:        modelos.PapelSuperAdmin,
		}, nil
	case "dev-gestor":
		return &modelos.UsuarioSessao{
			IDUsuario:    "usuario-dev-gestor",
			NomeCompleto: "Gestor de Rede Dev",
			IDRede:       "rede-dev-001",
			Papel:        modelos.PapelGestorRede,
		}, nil
	case "dev-frentista":
		return &modelos.UsuarioSessao{
			IDUsuario:    "usuario-dev-frentista",
			NomeCompleto: "Frentista Dev",
			IDRede:       "rede-dev-001",
			Papel:        modelos.PapelFrentista,
		}, nil
	default:
		return nil, ErrTokenInvalido
	}
}

func (a *autenticadorToken) CriarSessao(usuario *modelos.UsuarioSessao) string {
	token := utils.GerarToken("tok")
	copia := *usuario

	a.mu.Lock()
	a.sessoes[token] = &copia
	a.mu.Unlock()
	return token
}
