package modelos

import "time"

type AdministradorGeral struct {
	ID         string    `json:"id"`
	Nome       string    `json:"nome"`
	Email      string    `json:"email"`
	SenhaHash  string    `json:"-"`
	Ativo      bool      `json:"ativo"`
	CriadoEm   time.Time `json:"criado_em"`
	Atualizado time.Time `json:"atualizado_em"`
}
