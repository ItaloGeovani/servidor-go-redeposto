package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

type RedeLinksSociaisRepositorio interface {
	ListarPorRedeID(redeID string) ([]modelos.RedeLinkSocial, error)
	Substituir(redeID string, links []modelos.RedeLinkSocial) error
}

type redeLinksSociaisPostgres struct{ db *sql.DB }

func NovoRedeLinksSociaisPostgres(db *sql.DB) RedeLinksSociaisRepositorio {
	return &redeLinksSociaisPostgres{db: db}
}

func (r *redeLinksSociaisPostgres) ListarPorRedeID(redeID string) ([]modelos.RedeLinkSocial, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return nil, errors.New("rede invalida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT lower(trim(plataforma)), trim(titulo_exibicao), trim(url)
FROM rede_links_sociais
WHERE rede_id = $1::uuid
ORDER BY ordem ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, redeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []modelos.RedeLinkSocial
	for rows.Next() {
		var p, tit, u string
		if err := rows.Scan(&p, &tit, &u); err != nil {
			return nil, err
		}
		if p == "" || u == "" {
			continue
		}
		out = append(out, modelos.RedeLinkSocial{
			Plataforma:      p,
			TituloExibicao: tit,
			URL:            u,
		})
	}
	return out, rows.Err()
}

func (r *redeLinksSociaisPostgres) Substituir(redeID string, links []modelos.RedeLinkSocial) error {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return errors.New("rede invalida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM rede_links_sociais WHERE rede_id = $1::uuid`, redeID); err != nil {
		return err
	}
	const ins = `INSERT INTO rede_links_sociais (rede_id, ordem, plataforma, titulo_exibicao, url) VALUES ($1::uuid, $2, $3, $4, $5)`
	for i, L := range links {
		p := strings.ToLower(strings.TrimSpace(L.Plataforma))
		tit := strings.TrimSpace(L.TituloExibicao)
		u := strings.TrimSpace(L.URL)
		if p == "" || u == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, ins, redeID, i, p, tit, u); err != nil {
			return err
		}
	}
	return tx.Commit()
}
