package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"github.com/jackc/pgx/v5/pgconn"
)

type redePostgres struct {
	db *sql.DB
}

func NovoRedePostgres(db *sql.DB) RedeRepositorio {
	return &redePostgres{db: db}
}

func (r *redePostgres) Listar() ([]*modelos.Rede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  id::text,
  nome_fantasia,
  razao_social,
  COALESCE(cnpj, ''),
  COALESCE(email_contato, ''),
  COALESCE(telefone, ''),
  COALESCE(valor_implantacao, 0),
  COALESCE(valor_mensalidade, 0),
  primeiro_cobranca,
  COALESCE(dia_cobranca, 0),
  COALESCE(ativa, true),
  criado_em,
  atualizado_em
FROM redes
ORDER BY criado_em ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.Rede
	for rows.Next() {
		item, err := scanRede(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return lista, nil
}

func (r *redePostgres) BuscarPorID(id string) (*modelos.Rede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  id::text,
  nome_fantasia,
  razao_social,
  COALESCE(cnpj, ''),
  COALESCE(email_contato, ''),
  COALESCE(telefone, ''),
  COALESCE(valor_implantacao, 0),
  COALESCE(valor_mensalidade, 0),
  primeiro_cobranca,
  COALESCE(dia_cobranca, 0),
  COALESCE(ativa, true),
  criado_em,
  atualizado_em
FROM redes
WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	rede, err := scanRede(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRedeNaoEncontrada
		}
		return nil, err
	}
	return rede, nil
}

func (r *redePostgres) Criar(rede *modelos.Rede) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
INSERT INTO redes (
  nome_fantasia,
  razao_social,
  cnpj,
  email_contato,
  telefone,
  ativa,
  status,
  valor_implantacao,
  valor_mensalidade,
  primeiro_cobranca,
  dia_cobranca
)
VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, $9, $10, $11)
RETURNING id::text, criado_em, atualizado_em`

	status := "INATIVA"
	if rede.Ativa {
		status = "ATIVA"
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(rede.NomeFantasia),
		strings.TrimSpace(rede.RazaoSocial),
		strings.TrimSpace(rede.CNPJ),
		strings.TrimSpace(rede.EmailContato),
		strings.TrimSpace(rede.Telefone),
		rede.Ativa,
		status,
		rede.ValorImplantacao,
		rede.ValorMensalidade,
		rede.PrimeiroCobranca,
		rede.DiaCobranca,
	).Scan(&rede.ID, &rede.CriadoEm, &rede.AtualizadoEm)
	if err != nil {
		return mapearErroRedePostgres(err)
	}
	return nil
}

func (r *redePostgres) Atualizar(id string, atualizar func(*modelos.Rede) error) (*modelos.Rede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const queryBusca = `
SELECT
  id::text,
  nome_fantasia,
  razao_social,
  COALESCE(cnpj, ''),
  COALESCE(email_contato, ''),
  COALESCE(telefone, ''),
  COALESCE(valor_implantacao, 0),
  COALESCE(valor_mensalidade, 0),
  primeiro_cobranca,
  COALESCE(dia_cobranca, 0),
  COALESCE(ativa, true),
  criado_em,
  atualizado_em
FROM redes
WHERE id = $1
FOR UPDATE`

	rede, err := scanRede(tx.QueryRowContext(ctx, queryBusca, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRedeNaoEncontrada
		}
		return nil, err
	}

	if err := atualizar(rede); err != nil {
		return nil, err
	}

	const queryAtualiza = `
UPDATE redes
SET
  nome_fantasia = $2,
  razao_social = $3,
  cnpj = NULLIF($4, ''),
  email_contato = NULLIF($5, ''),
  telefone = NULLIF($6, ''),
  ativa = $7,
  status = $8,
  valor_implantacao = $9,
  valor_mensalidade = $10,
  primeiro_cobranca = $11,
  dia_cobranca = $12,
  atualizado_em = NOW()
WHERE id = $1
RETURNING atualizado_em`

	status := "INATIVA"
	if rede.Ativa {
		status = "ATIVA"
	}

	err = tx.QueryRowContext(
		ctx,
		queryAtualiza,
		id,
		strings.TrimSpace(rede.NomeFantasia),
		strings.TrimSpace(rede.RazaoSocial),
		strings.TrimSpace(rede.CNPJ),
		strings.TrimSpace(rede.EmailContato),
		strings.TrimSpace(rede.Telefone),
		rede.Ativa,
		status,
		rede.ValorImplantacao,
		rede.ValorMensalidade,
		rede.PrimeiroCobranca,
		rede.DiaCobranca,
	).Scan(&rede.AtualizadoEm)
	if err != nil {
		return nil, mapearErroRedePostgres(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return rede, nil
}

type scannerRede interface {
	Scan(dest ...any) error
}

func scanRede(s scannerRede) (*modelos.Rede, error) {
	var rede modelos.Rede
	var primeiro sql.NullTime

	err := s.Scan(
		&rede.ID,
		&rede.NomeFantasia,
		&rede.RazaoSocial,
		&rede.CNPJ,
		&rede.EmailContato,
		&rede.Telefone,
		&rede.ValorImplantacao,
		&rede.ValorMensalidade,
		&primeiro,
		&rede.DiaCobranca,
		&rede.Ativa,
		&rede.CriadoEm,
		&rede.AtualizadoEm,
	)
	if err != nil {
		return nil, err
	}
	if primeiro.Valid {
		rede.PrimeiroCobranca = primeiro.Time
	}
	return &rede, nil
}

func mapearErroRedePostgres(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code != "23505" {
		return err
	}

	switch pgErr.ConstraintName {
	case "uq_redes_cnpj":
		return ErrRedeCNPJDuplicado
	case "uq_redes_nome_fantasia_lower":
		return ErrRedeNomeDuplicado
	default:
		msg := strings.ToLower(pgErr.Message)
		if strings.Contains(msg, "cnpj") {
			return ErrRedeCNPJDuplicado
		}
		if strings.Contains(msg, "nome") {
			return ErrRedeNomeDuplicado
		}
		return err
	}
}
