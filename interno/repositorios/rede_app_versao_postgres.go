package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

// AppMobileRedeRepositorio versoes do app do cliente por rede (opcional; senao, usa a global).
type AppMobileRedeRepositorio interface {
	// Obter devolve a linha se existir; found=false se a rede ainda nao tiver registro.
	Obter(redeID string) (c *modelos.ConfiguracaoAppMobile, found bool, err error)
	Salvar(redeID string, c *modelos.ConfiguracaoAppMobile) error
}

type redeAppVersaoPostgres struct {
	db *sql.DB
}

func NovoAppMobileRedePostgres(db *sql.DB) AppMobileRedeRepositorio {
	return &redeAppVersaoPostgres{db: db}
}

func (r *redeAppVersaoPostgres) Obter(redeID string) (*modelos.ConfiguracaoAppMobile, bool, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return nil, false, errors.New("rede vazia")
	}
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
FROM rede_app_versao
WHERE rede_id = $1::uuid`

	var c modelos.ConfiguracaoAppMobile
	err := r.db.QueryRowContext(ctx, q, redeID).Scan(
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
			return nil, false, nil
		}
		return nil, false, err
	}
	return &c, true, nil
}

func (r *redeAppVersaoPostgres) Salvar(redeID string, c *modelos.ConfiguracaoAppMobile) error {
	if c == nil {
		return errors.New("configuracao vazia")
	}
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return errors.New("rede vazia")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
INSERT INTO rede_app_versao (
  rede_id, versao_ios, versao_android, url_loja_ios, url_loja_android,
  mensagem_atualizacao, atualizacao_obrigatoria, atualizado_em
) VALUES (
  $1::uuid, $2, $3, $4, $5, $6, $7, NOW()
)
ON CONFLICT (rede_id) DO UPDATE SET
  versao_ios = EXCLUDED.versao_ios,
  versao_android = EXCLUDED.versao_android,
  url_loja_ios = EXCLUDED.url_loja_ios,
  url_loja_android = EXCLUDED.url_loja_android,
  mensagem_atualizacao = EXCLUDED.mensagem_atualizacao,
  atualizacao_obrigatoria = EXCLUDED.atualizacao_obrigatoria,
  atualizado_em = NOW()`,
		redeID,
		strings.TrimSpace(c.VersaoIOS),
		strings.TrimSpace(c.VersaoAndroid),
		strings.TrimSpace(c.URLLojaIOS),
		strings.TrimSpace(c.URLLojaAndroid),
		strings.TrimSpace(c.MensagemAtualizacao),
		c.AtualizacaoObrigatoria,
	)
	return err
}
