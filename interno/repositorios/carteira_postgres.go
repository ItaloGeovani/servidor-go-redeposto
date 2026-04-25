package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type CarteiraRepositorio interface {
	ObterOuCriarCarteira(redeID, usuarioID, nomeToken string, cotacao float64) (string, error)
	CreditarBonus(
		redeID, carteiraID string,
		valorToken float64,
		tipoRef, idRef string,
	) error
}

type carteiraPostgres struct {
	db *sql.DB
}

func NovoCarteiraPostgres(db *sql.DB) CarteiraRepositorio {
	return &carteiraPostgres{db: db}
}

// ObterOuCriarCarteira cria [carteiras] com token da moeda virtual da rede.
func (r *carteiraPostgres) ObterOuCriarCarteira(redeID, usuarioID, nomeToken string, cotacao float64) (string, error) {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return "", errors.New("ids invalidos")
	}
	nt := strings.TrimSpace(nomeToken)
	if nt == "" {
		nt = "Moeda"
	}
	if cotacao <= 0 {
		return "", errors.New("cotacao invalida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	const find = `SELECT id::text FROM carteiras WHERE rede_id = $1::uuid AND usuario_id = $2::uuid`
	var id string
	err := r.db.QueryRowContext(ctx, find, redeID, usuarioID).Scan(&id)
	if err == nil && id != "" {
		return id, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	const ins = `
INSERT INTO carteiras (rede_id, usuario_id, codigo_moeda, nome_token, cotacao_token)
VALUES ($1::uuid, $2::uuid, 'BRL', $3, $4)
ON CONFLICT (rede_id, usuario_id) DO UPDATE SET
  atualizado_em = NOW()
RETURNING id::text`
	err = r.db.QueryRowContext(ctx, ins, redeID, usuarioID, nt, cotacao).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// CreditarBonus grava BONUS; idempotente via UNIQUE (rede, tipo_referencia, ref, tipo).
func (r *carteiraPostgres) CreditarBonus(redeID, carteiraID string, valorToken float64, tipoRef, idRef string) error {
	if valorToken <= 0 {
		return nil
	}
	redeID = strings.TrimSpace(redeID)
	carteiraID = strings.TrimSpace(carteiraID)
	tipoRef = strings.TrimSpace(tipoRef)
	idRef = strings.TrimSpace(idRef)
	if redeID == "" || carteiraID == "" || tipoRef == "" || idRef == "" {
		return errors.New("dados invalidos para bonus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'BONUS'::tipo_transacao_carteira, 0, $3::numeric, 1, $4, $5::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING
`
	_, err := r.db.ExecContext(ctx, ins, redeID, carteiraID, valorToken, tipoRef, idRef)
	return err
}
