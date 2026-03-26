package repositorios

import (
	"errors"
	"strings"
	"sync"
	"time"

	"gaspass-servidor/interno/modelos"
)

var (
	ErrAdminNaoEncontrado = errors.New("administrador geral nao encontrado")
	ErrEmailJaCadastrado  = errors.New("email ja cadastrado")
)

type AdministradorGeralRepositorio interface {
	Criar(admin *modelos.AdministradorGeral) error
	Atualizar(id, nome, email string, ativo bool) (*modelos.AdministradorGeral, error)
	BuscarPorEmail(email string) (*modelos.AdministradorGeral, error)
}

type administradorGeralMemoria struct {
	mu      sync.RWMutex
	porID   map[string]*modelos.AdministradorGeral
	porMail map[string]string
}

func NovoAdministradorGeralMemoria() AdministradorGeralRepositorio {
	return &administradorGeralMemoria{
		porID:   make(map[string]*modelos.AdministradorGeral),
		porMail: make(map[string]string),
	}
}

func (r *administradorGeralMemoria) Criar(admin *modelos.AdministradorGeral) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	emailNorm := normalizarEmail(admin.Email)
	if _, ok := r.porMail[emailNorm]; ok {
		return ErrEmailJaCadastrado
	}

	copia := *admin
	copia.Email = emailNorm
	now := time.Now().UTC()
	copia.CriadoEm = now
	copia.Atualizado = now

	r.porID[copia.ID] = &copia
	r.porMail[emailNorm] = copia.ID
	return nil
}

func (r *administradorGeralMemoria) Atualizar(id, nome, email string, ativo bool) (*modelos.AdministradorGeral, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	admin, ok := r.porID[id]
	if !ok {
		return nil, ErrAdminNaoEncontrado
	}

	emailNovo := normalizarEmail(email)
	if emailNovo != admin.Email {
		if idExistente, ok := r.porMail[emailNovo]; ok && idExistente != id {
			return nil, ErrEmailJaCadastrado
		}
		delete(r.porMail, admin.Email)
		r.porMail[emailNovo] = id
		admin.Email = emailNovo
	}

	admin.Nome = strings.TrimSpace(nome)
	admin.Ativo = ativo
	admin.Atualizado = time.Now().UTC()
	return clonarAdmin(admin), nil
}

func (r *administradorGeralMemoria) BuscarPorEmail(email string) (*modelos.AdministradorGeral, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.porMail[normalizarEmail(email)]
	if !ok {
		return nil, ErrAdminNaoEncontrado
	}

	admin, ok := r.porID[id]
	if !ok {
		return nil, ErrAdminNaoEncontrado
	}
	return clonarAdmin(admin), nil
}

func normalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func clonarAdmin(orig *modelos.AdministradorGeral) *modelos.AdministradorGeral {
	copia := *orig
	return &copia
}
