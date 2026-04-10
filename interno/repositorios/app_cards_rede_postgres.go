package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

// AppCardsRedeRepositorio cards do app por rede (slots 0-3).
type AppCardsRedeRepositorio interface {
	ListarPorRedeID(idRede string) ([]*modelos.AppCardRede, error)
	SubstituirPorRede(idRede string, cards []*modelos.AppCardRede) error
}

type appCardsRedePostgres struct {
	db *sql.DB
}

func NovoAppCardsRedePostgres(db *sql.DB) AppCardsRedeRepositorio {
	return &appCardsRedePostgres{db: db}
}

func (r *appCardsRedePostgres) ListarPorRedeID(idRede string) ([]*modelos.AppCardRede, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, errors.New("rede_id invalido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
SELECT
  id::text,
  rede_id::text,
  slot,
  COALESCE(titulo, ''),
  COALESCE(imagem_url, ''),
  COALESCE(link_url, ''),
  ativo,
  criado_em,
  atualizado_em
FROM app_cards_rede
WHERE rede_id = $1::uuid
ORDER BY slot ASC`, idRede)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.AppCardRede
	for rows.Next() {
		var c modelos.AppCardRede
		if err := rows.Scan(
			&c.ID, &c.IDRede, &c.Slot, &c.Titulo, &c.ImagemURL, &c.LinkURL, &c.Ativo, &c.CriadoEm, &c.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		lista = append(lista, &c)
	}
	return lista, rows.Err()
}

func (r *appCardsRedePostgres) SubstituirPorRede(idRede string, cards []*modelos.AppCardRede) error {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return errors.New("rede_id invalido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM app_cards_rede WHERE rede_id = $1::uuid`, idRede); err != nil {
		return err
	}

	const ins = `
INSERT INTO app_cards_rede (rede_id, slot, titulo, imagem_url, link_url, ativo, atualizado_em)
VALUES ($1::uuid, $2, $3, $4, $5, $6, NOW())`

	for _, c := range cards {
		if c == nil {
			continue
		}
		slot := c.Slot
		if slot < 0 || slot > 3 {
			return errors.New("slot invalido (use 0 a 3)")
		}
		titulo := strings.TrimSpace(c.Titulo)
		img := strings.TrimSpace(c.ImagemURL)
		link := strings.TrimSpace(c.LinkURL)
		if _, err := tx.ExecContext(ctx, ins, idRede, slot, titulo, img, link, c.Ativo); err != nil {
			return err
		}
	}

	return tx.Commit()
}
