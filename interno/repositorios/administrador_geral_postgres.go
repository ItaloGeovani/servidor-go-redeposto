package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

type administradorGeralPostgres struct {
	db *sql.DB
}

func NovoAdministradorGeralPostgres(db *sql.DB) AdministradorGeralRepositorio {
	return &administradorGeralPostgres{db: db}
}

func (r *administradorGeralPostgres) Criar(admin *modelos.AdministradorGeral) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	emailNormalizado := normalizarEmail(admin.Email)
	existe, err := r.existeEmailSuperAdmin(ctx, emailNormalizado, "")
	if err != nil {
		return err
	}
	if existe {
		return ErrEmailJaCadastrado
	}

	const query = `
INSERT INTO usuarios (
  rede_id,
  papel,
  nome_completo,
  email,
  senha_hash,
  ativo
)
VALUES (NULL, 'super_admin', $1, $2, $3, $4)
RETURNING id::text, criado_em, atualizado_em`

	err = r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(admin.Nome),
		emailNormalizado,
		admin.SenhaHash,
		admin.Ativo,
	).Scan(&admin.ID, &admin.CriadoEm, &admin.Atualizado)
	if err != nil {
		return err
	}
	admin.Email = emailNormalizado
	return nil
}

func (r *administradorGeralPostgres) Atualizar(id, nome, email string, ativo bool) (*modelos.AdministradorGeral, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id = strings.TrimSpace(id)
	emailNormalizado := normalizarEmail(email)
	if id == "" {
		return nil, ErrAdminNaoEncontrado
	}

	existe, err := r.existeEmailSuperAdmin(ctx, emailNormalizado, id)
	if err != nil {
		return nil, err
	}
	if existe {
		return nil, ErrEmailJaCadastrado
	}

	const query = `
UPDATE usuarios
SET
  nome_completo = $2,
  email = $3,
  ativo = $4,
  atualizado_em = NOW()
WHERE id = $1
  AND papel = 'super_admin'
RETURNING id::text, nome_completo, email, senha_hash, ativo, criado_em, atualizado_em`

	admin := &modelos.AdministradorGeral{}
	err = r.db.QueryRowContext(
		ctx,
		query,
		id,
		strings.TrimSpace(nome),
		emailNormalizado,
		ativo,
	).Scan(
		&admin.ID,
		&admin.Nome,
		&admin.Email,
		&admin.SenhaHash,
		&admin.Ativo,
		&admin.CriadoEm,
		&admin.Atualizado,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAdminNaoEncontrado
		}
		return nil, err
	}
	return admin, nil
}

func (r *administradorGeralPostgres) BuscarPorEmail(email string) (*modelos.AdministradorGeral, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  id::text,
  nome_completo,
  email,
  senha_hash,
  ativo,
  criado_em,
  atualizado_em
FROM usuarios
WHERE papel = 'super_admin'
  AND LOWER(email) = LOWER($1)
LIMIT 1`

	admin := &modelos.AdministradorGeral{}
	err := r.db.QueryRowContext(ctx, query, normalizarEmail(email)).Scan(
		&admin.ID,
		&admin.Nome,
		&admin.Email,
		&admin.SenhaHash,
		&admin.Ativo,
		&admin.CriadoEm,
		&admin.Atualizado,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAdminNaoEncontrado
		}
		return nil, err
	}
	return admin, nil
}

func (r *administradorGeralPostgres) existeEmailSuperAdmin(ctx context.Context, email, excluirID string) (bool, error) {
	if email == "" {
		return false, nil
	}

	const query = `
SELECT 1
FROM usuarios
WHERE papel = 'super_admin'
  AND LOWER(email) = LOWER($1)
  AND ($2 = '' OR id::text <> $2)
LIMIT 1`

	var marcador int
	err := r.db.QueryRowContext(ctx, query, email, strings.TrimSpace(excluirID)).Scan(&marcador)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
