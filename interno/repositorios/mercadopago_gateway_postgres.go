package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type mercadoPagoGatewayPostgres struct {
	db *sql.DB
}

func NovoMercadoPagoGatewayPostgres(db *sql.DB) MercadoPagoGatewayRepositorio {
	return &mercadoPagoGatewayPostgres{db: db}
}

func (r *mercadoPagoGatewayPostgres) BuscarPorRedeID(idRede string) (*MercadoPagoGatewayCredenciais, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrMercadoPagoGatewayNaoConfigurado
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const q = `
SELECT
  COALESCE(TRIM(mp_access_token), ''),
  COALESCE(TRIM(mp_webhook_secret), '')
FROM rede_mercado_pago
WHERE rede_id = $1`

	var c MercadoPagoGatewayCredenciais
	err := r.db.QueryRowContext(ctx, q, idRede).Scan(&c.AccessToken, &c.WebhookSecret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMercadoPagoGatewayNaoConfigurado
		}
		return nil, err
	}
	if c.AccessToken == "" && c.WebhookSecret == "" {
		return nil, ErrMercadoPagoGatewayNaoConfigurado
	}
	return &c, nil
}

func (r *mercadoPagoGatewayPostgres) Upsert(idRede, accessToken, webhookSecret string) error {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return errors.New("rede_id vazio")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	at := strings.TrimSpace(accessToken)
	ws := strings.TrimSpace(webhookSecret)

	const q = `
INSERT INTO rede_mercado_pago (rede_id, mp_access_token, mp_webhook_secret, atualizado_em)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NOW())
ON CONFLICT (rede_id) DO UPDATE SET
  mp_access_token = NULLIF(EXCLUDED.mp_access_token, ''),
  mp_webhook_secret = NULLIF(EXCLUDED.mp_webhook_secret, ''),
  atualizado_em = NOW()`

	_, err := r.db.ExecContext(ctx, q, idRede, at, ws)
	return err
}
