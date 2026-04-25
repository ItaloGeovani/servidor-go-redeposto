package repositorios

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// NivelClienteLinha um nivel configurado (Bronze, Prata, ...).
type NivelClienteLinha struct {
	Codigo         string  `json:"codigo"`
	Nome           string  `json:"nome"`
	MultMoeda      float64 `json:"mult_moeda"`
	MultDesconto   float64 `json:"mult_desconto"`
	Ordem          int     `json:"ordem"`
}

// RedeNiveisClienteConfig linha de rede_niveis_cliente_config.
type RedeNiveisClienteConfig struct {
	Ativo                bool
	MultDescontoAtivo    bool
	Niveis               []NivelClienteLinha
	AtualizadoEm         time.Time
}

// NiveisClienteRepositorio persistencia de niveis por rede.
type NiveisClienteRepositorio interface {
	Buscar(redeID string) (*RedeNiveisClienteConfig, error)
	Salvar(redeID string, ativo, multDesc bool, niveis []NivelClienteLinha) error
}

type niveisClientePostgres struct {
	db *sql.DB
}

func NovoNiveisClientePostgres(db *sql.DB) NiveisClienteRepositorio {
	return &niveisClientePostgres{db: db}
}

func (r *niveisClientePostgres) Buscar(redeID string) (*RedeNiveisClienteConfig, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return nil, errors.New("rede vazia")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  ativo,
  mult_desconto_ativo,
  niveis::text,
  atualizado_em
FROM rede_niveis_cliente_config
WHERE rede_id = $1::uuid`
	var x RedeNiveisClienteConfig
	var raw string
	err := r.db.QueryRowContext(ctx, q, redeID).Scan(&x.Ativo, &x.MultDescontoAtivo, &raw, &x.AtualizadoEm)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), &x.Niveis); err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *niveisClientePostgres) Salvar(redeID string, ativo, multDesc bool, niveis []NivelClienteLinha) error {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return errors.New("rede vazia")
	}
	b, err := json.Marshal(niveis)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
INSERT INTO rede_niveis_cliente_config (rede_id, ativo, mult_desconto_ativo, niveis, atualizado_em)
VALUES ($1::uuid, $2, $3, $4::jsonb, NOW())
ON CONFLICT (rede_id) DO UPDATE SET
  ativo = EXCLUDED.ativo,
  mult_desconto_ativo = EXCLUDED.mult_desconto_ativo,
  niveis = EXCLUDED.niveis,
  atualizado_em = NOW()
`
	_, err = r.db.ExecContext(ctx, q, redeID, ativo, multDesc, b)
	return err
}
