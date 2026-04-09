package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

type campanhaPostgres struct {
	db *sql.DB
}

func NovoCampanhaPostgres(db *sql.DB) *campanhaPostgres {
	return &campanhaPostgres{db: db}
}

var ErrCampanhaNaoEncontrada = errors.New("campanha nao encontrada")

func (r *campanhaPostgres) resolverCriadoPor(sessaoID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessaoID = strings.TrimSpace(sessaoID)
	var id string
	err := r.db.QueryRowContext(ctx, `
SELECT id::text FROM usuarios WHERE id::text = $1 LIMIT 1`, sessaoID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = r.db.QueryRowContext(ctx, `
SELECT id::text FROM usuarios WHERE papel = 'super_admin' ORDER BY criado_em ASC LIMIT 1`).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("nenhum usuario super_admin no banco para registrar campanha")
		}
		return "", err
	}
	return id, nil
}

func (r *campanhaPostgres) PostoPertenceARede(idPosto, idRede string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var um int
	err := r.db.QueryRowContext(ctx, `
SELECT 1 FROM postos WHERE id = $1::uuid AND rede_id = $2::uuid LIMIT 1`, idPosto, idRede).Scan(&um)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func preencherTituloExibicaoEscopo(c *modelos.Campanha) {
	if strings.TrimSpace(c.Titulo) != "" {
		c.TituloExibicao = strings.TrimSpace(c.Titulo)
	} else {
		c.TituloExibicao = strings.TrimSpace(c.Nome)
	}
	if strings.TrimSpace(c.IDPosto) == "" {
		c.Escopo = "rede"
	} else {
		c.Escopo = "posto"
	}
}

func (r *campanhaPostgres) ListarPorRedeID(idRede string) ([]*modelos.Campanha, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
SELECT
  c.id::text,
  c.rede_id::text,
  c.nome,
  COALESCE(c.titulo, ''),
  COALESCE(c.descricao, ''),
  COALESCE(c.imagem_url, ''),
  COALESCE(c.posto_id::text, ''),
  c.status::text,
  c.vigencia_inicio,
  c.vigencia_fim,
  c.valida_no_app,
  c.valida_no_posto_fisico,
  c.modalidade_desconto,
  c.base_desconto,
  c.valor_desconto::float8,
  c.valor_minimo_compra::float8,
  c.max_usos_por_cliente,
  c.criado_por::text,
  c.criado_em,
  c.atualizado_em
FROM campanhas c
WHERE c.rede_id = $1::uuid
ORDER BY c.vigencia_inicio DESC NULLS LAST, c.criado_em DESC`

	rows, err := r.db.QueryContext(ctx, query, strings.TrimSpace(idRede))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.Campanha
	for rows.Next() {
		var c modelos.Campanha
		var vigIni, vigFim sql.NullTime
		var maxUsos sql.NullInt64
		if err := rows.Scan(
			&c.ID, &c.IDRede, &c.Nome, &c.Titulo, &c.Descricao, &c.ImagemURL,
			&c.IDPosto, &c.Status, &vigIni, &vigFim,
			&c.ValidaNoApp, &c.ValidaNoPostoFisico,
			&c.ModalidadeDesconto, &c.BaseDesconto,
			&c.ValorDesconto, &c.ValorMinimoCompra,
			&maxUsos,
			&c.CriadoPor, &c.CriadoEm, &c.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		if vigIni.Valid {
			t := vigIni.Time
			c.VigenciaInicio = &t
		}
		if vigFim.Valid {
			t := vigFim.Time
			c.VigenciaFim = &t
		}
		if maxUsos.Valid {
			v := int(maxUsos.Int64)
			c.MaxUsosPorCliente = &v
		}
		preencherTituloExibicaoEscopo(&c)
		lista = append(lista, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

func (r *campanhaPostgres) Criar(sessaoCriador string, c *modelos.Campanha) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	criadoPor, err := r.resolverCriadoPor(sessaoCriador)
	if err != nil {
		return err
	}

	var posto any
	if strings.TrimSpace(c.IDPosto) == "" {
		posto = nil
	} else {
		posto = strings.TrimSpace(c.IDPosto)
	}

	var vigIni, vigFim any
	if c.VigenciaInicio != nil {
		vigIni = *c.VigenciaInicio
	}
	if c.VigenciaFim != nil {
		vigFim = *c.VigenciaFim
	}

	var maxUsos any
	if c.MaxUsosPorCliente != nil {
		maxUsos = *c.MaxUsosPorCliente
	}

	const query = `
INSERT INTO campanhas (
  rede_id, nome, descricao, status, criado_por,
  imagem_url, titulo, posto_id, vigencia_inicio, vigencia_fim,
  valida_no_app, valida_no_posto_fisico,
  modalidade_desconto, base_desconto, valor_desconto, valor_minimo_compra, max_usos_por_cliente
)
VALUES (
  $1::uuid, $2, NULLIF($3, ''), $4::status_campanha, $5::uuid,
  NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17
)
RETURNING id::text, criado_em, atualizado_em`

	err = r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(c.IDRede),
		strings.TrimSpace(c.Nome),
		strings.TrimSpace(c.Descricao),
		string(c.Status),
		criadoPor,
		strings.TrimSpace(c.ImagemURL),
		strings.TrimSpace(c.Titulo),
		posto,
		vigIni,
		vigFim,
		c.ValidaNoApp,
		c.ValidaNoPostoFisico,
		strings.TrimSpace(c.ModalidadeDesconto),
		strings.TrimSpace(c.BaseDesconto),
		c.ValorDesconto,
		c.ValorMinimoCompra,
		maxUsos,
	).Scan(&c.ID, &c.CriadoEm, &c.AtualizadoEm)
	if err != nil {
		return err
	}
	c.CriadoPor = criadoPor
	preencherTituloExibicaoEscopo(c)
	return nil
}

func (r *campanhaPostgres) Atualizar(c *modelos.Campanha) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var posto any
	if strings.TrimSpace(c.IDPosto) == "" {
		posto = nil
	} else {
		posto = strings.TrimSpace(c.IDPosto)
	}

	var vigIni, vigFim any
	if c.VigenciaInicio != nil {
		vigIni = *c.VigenciaInicio
	}
	if c.VigenciaFim != nil {
		vigFim = *c.VigenciaFim
	}

	var maxUsos any
	if c.MaxUsosPorCliente != nil {
		maxUsos = *c.MaxUsosPorCliente
	}

	const query = `
UPDATE campanhas SET
  nome = $1,
  descricao = NULLIF($2, ''),
  status = $3::status_campanha,
  imagem_url = NULLIF($4, ''),
  titulo = NULLIF($5, ''),
  posto_id = $6,
  vigencia_inicio = $7,
  vigencia_fim = $8,
  valida_no_app = $9,
  valida_no_posto_fisico = $10,
  modalidade_desconto = $11,
  base_desconto = $12,
  valor_desconto = $13,
  valor_minimo_compra = $14,
  max_usos_por_cliente = $15,
  atualizado_em = NOW()
WHERE id = $16::uuid AND rede_id = $17::uuid`

	res, err := r.db.ExecContext(
		ctx,
		query,
		strings.TrimSpace(c.Nome),
		strings.TrimSpace(c.Descricao),
		string(c.Status),
		strings.TrimSpace(c.ImagemURL),
		strings.TrimSpace(c.Titulo),
		posto,
		vigIni,
		vigFim,
		c.ValidaNoApp,
		c.ValidaNoPostoFisico,
		strings.TrimSpace(c.ModalidadeDesconto),
		strings.TrimSpace(c.BaseDesconto),
		c.ValorDesconto,
		c.ValorMinimoCompra,
		maxUsos,
		strings.TrimSpace(c.ID),
		strings.TrimSpace(c.IDRede),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCampanhaNaoEncontrada
	}
	return nil
}
