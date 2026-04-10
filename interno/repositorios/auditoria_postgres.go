package repositorios

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

// ErrAuditoriaRedeIDInvalido quando rede_id esta vazio na consulta.
var ErrAuditoriaRedeIDInvalido = errors.New("rede_id invalido")

// AuditoriaRepositorio leitura de logs de auditoria por rede ou plataforma (admin).
type AuditoriaRepositorio interface {
	ListarPorRedeID(idRede string, limite, offset int) ([]*modelos.LogAuditoria, int, error)
	// ListarPlataforma lista todos os logs (ou filtra por id_rede quando informado).
	ListarPlataforma(idRedeFiltro string, limite, offset int) ([]*modelos.LogAuditoria, int, error)
}

type auditoriaPostgres struct {
	db *sql.DB
}

func NovoAuditoriaPostgres(db *sql.DB) AuditoriaRepositorio {
	return &auditoriaPostgres{db: db}
}

func (r *auditoriaPostgres) ListarPorRedeID(idRede string, limite, offset int) ([]*modelos.LogAuditoria, int, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, 0, ErrAuditoriaRedeIDInvalido
	}
	if limite < 1 {
		limite = 50
	}
	if limite > 200 {
		limite = 200
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var total int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM logs_auditoria
WHERE rede_id = $1::uuid`, idRede).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT
  id::text,
  rede_id::text,
  usuario_ator_id::text,
  tipo_evento,
  tipo_entidade,
  entidade_id::text,
  COALESCE(dados_anteriores::text, ''),
  COALESCE(dados_novos::text, ''),
  COALESCE(NULLIF(ip_origem::text, ''), ''),
  COALESCE(agente_usuario, ''),
  criado_em
FROM logs_auditoria
WHERE rede_id = $1::uuid
ORDER BY criado_em DESC
LIMIT $2 OFFSET $3`, idRede, limite, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []*modelos.LogAuditoria
	for rows.Next() {
		la, err := scanLinhaAuditoria(rows)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, la)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lista, total, nil
}

func scanLinhaAuditoria(rows *sql.Rows) (*modelos.LogAuditoria, error) {
	var (
		la          modelos.LogAuditoria
		redeID      sql.NullString
		usuarioAtor sql.NullString
		entidadeID  sql.NullString
		dadosAntStr string
		dadosNovStr string
		ipStr       string
		agenteStr   string
	)
	if err := rows.Scan(
		&la.ID,
		&redeID,
		&usuarioAtor,
		&la.TipoEvento,
		&la.TipoEntidade,
		&entidadeID,
		&dadosAntStr,
		&dadosNovStr,
		&ipStr,
		&agenteStr,
		&la.CriadoEm,
	); err != nil {
		return nil, err
	}
	if redeID.Valid && redeID.String != "" {
		s := redeID.String
		la.IDRede = &s
	}
	if usuarioAtor.Valid && usuarioAtor.String != "" {
		s := usuarioAtor.String
		la.IDUsuarioAtor = &s
	}
	if entidadeID.Valid && entidadeID.String != "" {
		s := entidadeID.String
		la.IDEntidade = &s
	}
	if dadosAntStr != "" && dadosAntStr != "null" {
		if json.Valid([]byte(dadosAntStr)) {
			la.DadosAnteriores = json.RawMessage(dadosAntStr)
		}
	}
	if dadosNovStr != "" && dadosNovStr != "null" {
		if json.Valid([]byte(dadosNovStr)) {
			la.DadosNovos = json.RawMessage(dadosNovStr)
		}
	}
	if ipStr != "" {
		la.IPOrigem = &ipStr
	}
	if agenteStr != "" {
		la.AgenteUsuario = &agenteStr
	}
	return &la, nil
}

func (r *auditoriaPostgres) ListarPlataforma(idRedeFiltro string, limite, offset int) ([]*modelos.LogAuditoria, int, error) {
	if limite < 1 {
		limite = 50
	}
	if limite > 200 {
		limite = 200
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	idRedeFiltro = strings.TrimSpace(idRedeFiltro)

	var (
		total int
		err   error
	)
	if idRedeFiltro == "" {
		err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM logs_auditoria`).Scan(&total)
	} else {
		err = r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM logs_auditoria WHERE rede_id = $1::uuid`, idRedeFiltro).Scan(&total)
	}
	if err != nil {
		return nil, 0, err
	}

	const colunas = `
SELECT
  id::text,
  rede_id::text,
  usuario_ator_id::text,
  tipo_evento,
  tipo_entidade,
  entidade_id::text,
  COALESCE(dados_anteriores::text, ''),
  COALESCE(dados_novos::text, ''),
  COALESCE(NULLIF(ip_origem::text, ''), ''),
  COALESCE(agente_usuario, ''),
  criado_em
FROM logs_auditoria`

	var rows *sql.Rows
	if idRedeFiltro == "" {
		rows, err = r.db.QueryContext(ctx, colunas+`
ORDER BY criado_em DESC
LIMIT $1 OFFSET $2`, limite, offset)
	} else {
		rows, err = r.db.QueryContext(ctx, colunas+`
WHERE rede_id = $1::uuid
ORDER BY criado_em DESC
LIMIT $2 OFFSET $3`, idRedeFiltro, limite, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []*modelos.LogAuditoria
	for rows.Next() {
		la, err := scanLinhaAuditoria(rows)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, la)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lista, total, nil
}
