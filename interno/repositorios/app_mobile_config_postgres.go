package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

// AppMobileConfigRepositorio configuracao global dos apps (uma linha).
type AppMobileConfigRepositorio interface {
	Obter() (*modelos.ConfiguracaoAppMobile, error)
	Salvar(c *modelos.ConfiguracaoAppMobile) error
}

type appMobileConfigPostgres struct {
	db *sql.DB
}

func NovoAppMobileConfigPostgres(db *sql.DB) AppMobileConfigRepositorio {
	return &appMobileConfigPostgres{db: db}
}

func (r *appMobileConfigPostgres) Obter() (*modelos.ConfiguracaoAppMobile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const q = `
SELECT
  COALESCE(NULLIF(TRIM(versao_ios), ''), '0.0.0'),
  COALESCE(NULLIF(TRIM(versao_android), ''), '0.0.0'),
  COALESCE(url_loja_ios, ''),
  COALESCE(url_loja_android, ''),
  COALESCE(mensagem_atualizacao, ''),
  atualizacao_obrigatoria,
  atualizado_em
FROM configuracao_app_mobile
WHERE id = 1`

	var c modelos.ConfiguracaoAppMobile
	err := r.db.QueryRowContext(ctx, q).Scan(
		&c.VersaoIOS,
		&c.VersaoAndroid,
		&c.URLLojaIOS,
		&c.URLLojaAndroid,
		&c.MensagemAtualizacao,
		&c.AtualizacaoObrigatoria,
		&c.AtualizadoEm,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &modelos.ConfiguracaoAppMobile{
				VersaoIOS:     "0.0.0",
				VersaoAndroid: "0.0.0",
			}, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *appMobileConfigPostgres) Salvar(c *modelos.ConfiguracaoAppMobile) error {
	if c == nil {
		return errors.New("configuracao vazia")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
INSERT INTO configuracao_app_mobile (
  id, versao_ios, versao_android, url_loja_ios, url_loja_android,
  mensagem_atualizacao, atualizacao_obrigatoria, atualizado_em
) VALUES (
  1, $1, $2, $3, $4, $5, $6, NOW()
)
ON CONFLICT (id) DO UPDATE SET
  versao_ios = EXCLUDED.versao_ios,
  versao_android = EXCLUDED.versao_android,
  url_loja_ios = EXCLUDED.url_loja_ios,
  url_loja_android = EXCLUDED.url_loja_android,
  mensagem_atualizacao = EXCLUDED.mensagem_atualizacao,
  atualizacao_obrigatoria = EXCLUDED.atualizacao_obrigatoria,
  atualizado_em = NOW()`,
		strings.TrimSpace(c.VersaoIOS),
		strings.TrimSpace(c.VersaoAndroid),
		strings.TrimSpace(c.URLLojaIOS),
		strings.TrimSpace(c.URLLojaAndroid),
		strings.TrimSpace(c.MensagemAtualizacao),
		c.AtualizacaoObrigatoria,
	)
	return err
}
