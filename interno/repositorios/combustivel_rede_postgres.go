package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type combustivelRedePostgres struct {
	db *sql.DB
}

func NovoCombustivelRedePostgres(db *sql.DB) CombustivelRedeRepositorio {
	return &combustivelRedePostgres{db: db}
}

func (r *combustivelRedePostgres) ListarPorRede(redeID string) ([]*CombustivelRedeRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT
  id::text, rede_id::text, TRIM(nome), COALESCE(TRIM(codigo), ''), COALESCE(descricao, ''),
  preco_por_litro::float8, ativo, ordem, criado_em, atualizado_em
FROM rede_combustiveis
WHERE rede_id = $1::uuid
ORDER BY ordem ASC, nome ASC`
	rows, err := r.db.QueryContext(ctx, q, strings.TrimSpace(redeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*CombustivelRedeRegistro
	for rows.Next() {
		var x CombustivelRedeRegistro
		if err := rows.Scan(
			&x.ID, &x.RedeID, &x.Nome, &x.Codigo, &x.Descricao,
			&x.PrecoPorLitro, &x.Ativo, &x.Ordem, &x.CriadoEm, &x.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		out = append(out, &x)
	}
	return out, rows.Err()
}

func (r *combustivelRedePostgres) BuscarPorID(id, redeID string) (*CombustivelRedeRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT
  id::text, rede_id::text, TRIM(nome), COALESCE(TRIM(codigo), ''), COALESCE(descricao, ''),
  preco_por_litro::float8, ativo, ordem, criado_em, atualizado_em
FROM rede_combustiveis
WHERE id = $1::uuid AND rede_id = $2::uuid`
	var x CombustivelRedeRegistro
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(id), strings.TrimSpace(redeID)).Scan(
		&x.ID, &x.RedeID, &x.Nome, &x.Codigo, &x.Descricao,
		&x.PrecoPorLitro, &x.Ativo, &x.Ordem, &x.CriadoEm, &x.AtualizadoEm,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCombustivelRedeNaoEncontrado
		}
		return nil, err
	}
	return &x, nil
}

func (r *combustivelRedePostgres) Criar(x *CombustivelRedeRegistro) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var cod, des sql.NullString
	if t := strings.TrimSpace(x.Codigo); t != "" {
		cod = sql.NullString{String: t, Valid: true}
	}
	if t := strings.TrimSpace(x.Descricao); t != "" {
		des = sql.NullString{String: t, Valid: true}
	}
	const q = `
INSERT INTO rede_combustiveis (rede_id, nome, codigo, descricao, preco_por_litro, ativo, ordem)
VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
RETURNING id::text, criado_em, atualizado_em`
	err := r.db.QueryRowContext(
		ctx, q,
		strings.TrimSpace(x.RedeID),
		strings.TrimSpace(x.Nome),
		cod,
		des,
		x.PrecoPorLitro,
		x.Ativo,
		x.Ordem,
	).Scan(&x.ID, &x.CriadoEm, &x.AtualizadoEm)
	if err != nil {
		return mapearErrCombustivelPG(err)
	}
	return nil
}

func (r *combustivelRedePostgres) Atualizar(id, redeID string, atualizar func(*CombustivelRedeRegistro) error) (*CombustivelRedeRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	const qBusca = `
SELECT
  id::text, rede_id::text, TRIM(nome), COALESCE(TRIM(codigo), ''), COALESCE(descricao, ''),
  preco_por_litro::float8, ativo, ordem, criado_em, atualizado_em
FROM rede_combustiveis
WHERE id = $1::uuid AND rede_id = $2::uuid
FOR UPDATE`
	var row CombustivelRedeRegistro
	err = tx.QueryRowContext(ctx, qBusca, strings.TrimSpace(id), strings.TrimSpace(redeID)).Scan(
		&row.ID, &row.RedeID, &row.Nome, &row.Codigo, &row.Descricao,
		&row.PrecoPorLitro, &row.Ativo, &row.Ordem, &row.CriadoEm, &row.AtualizadoEm,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCombustivelRedeNaoEncontrado
		}
		return nil, err
	}
	if err := atualizar(&row); err != nil {
		return nil, err
	}
	var codSet, desSet sql.NullString
	if t := strings.TrimSpace(row.Codigo); t != "" {
		codSet = sql.NullString{String: t, Valid: true}
	}
	if t := strings.TrimSpace(row.Descricao); t != "" {
		desSet = sql.NullString{String: t, Valid: true}
	}
	const qUp = `
UPDATE rede_combustiveis
SET
  nome = $3,
  codigo = $4,
  descricao = $5,
  preco_por_litro = $6,
  ativo = $7,
  ordem = $8,
  atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $2::uuid
RETURNING atualizado_em`
	err = tx.QueryRowContext(
		ctx, qUp,
		strings.TrimSpace(id),
		strings.TrimSpace(redeID),
		strings.TrimSpace(row.Nome),
		codSet,
		desSet,
		row.PrecoPorLitro,
		row.Ativo,
		row.Ordem,
	).Scan(&row.AtualizadoEm)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapearErrCombustivelPG(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *combustivelRedePostgres) Excluir(id, redeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := r.db.ExecContext(ctx, `
DELETE FROM rede_combustiveis
WHERE id = $1::uuid AND rede_id = $2::uuid`,
		strings.TrimSpace(id), strings.TrimSpace(redeID),
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCombustivelRedeNaoEncontrado
	}
	return nil
}

func mapearErrCombustivelPG(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	if strings.Contains(strings.ToLower(pgErr.ConstraintName), "uq_rede_combustivel_codigo") {
		return errors.New("ja existe combustivel com este codigo nesta rede")
	}
	return err
}
