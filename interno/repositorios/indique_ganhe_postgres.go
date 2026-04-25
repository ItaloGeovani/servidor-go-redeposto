package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// RedeIndiqueGanheConfig configuração persistida.
type RedeIndiqueGanheConfig struct {
	Regra                 string
	MoedasPremioReferente float64
	MoedasPremioIndicado  float64
}

// IndicacaoRegisto linha de indicacoes.
type IndicacaoRegisto struct {
	ID                  string
	RedeID              string
	ReferenteUsuarioID  string
	IndicadoUsuarioID   string
	PremiadoCadastroRef bool
	PremiadoCadastroInd bool
	PremiadoCompraRef   bool
	PremiadoCompraInd   bool
}

type IndiqueGanheRepositorio interface {
	BuscarConfig(redeID string) (*RedeIndiqueGanheConfig, error)
	SalvarConfig(redeID, regra string, ref, ind float64) error
	InsertIndicacao(rede, referente, indicado, codInformado string) (string, error)
	BuscarIndicacaoPorIndicado(rede, indicadoID string) (*IndicacaoRegisto, error)
	MarcarPremioCadastro(rede, indicID string, refOK, indOK bool) error
	MarcarPremioCompra(rede, indicID string, refOK, indOK bool) error
	ContarVouchersAprovadosUsuario(rede, usuarioID string) (int, error)
}

type indiqueGanhePostgres struct {
	db *sql.DB
}

func NovoIndiqueGanhePostgres(db *sql.DB) IndiqueGanheRepositorio {
	return &indiqueGanhePostgres{db: db}
}

func (r *indiqueGanhePostgres) BuscarConfig(redeID string) (*RedeIndiqueGanheConfig, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return nil, errors.New("rede vazia")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  regra,
  moedas_premio_referente::float8,
  moedas_premio_indicado::float8
FROM rede_indique_ganhe_config
WHERE rede_id = $1::uuid`
	var c RedeIndiqueGanheConfig
	err := r.db.QueryRowContext(ctx, q, redeID).Scan(&c.Regra, &c.MoedasPremioReferente, &c.MoedasPremioIndicado)
	if errors.Is(err, sql.ErrNoRows) {
		return &RedeIndiqueGanheConfig{
			Regra:                 "PRIMEIRA_COMPRA_VOUCHER",
			MoedasPremioReferente: 0,
			MoedasPremioIndicado:  0,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *indiqueGanhePostgres) SalvarConfig(redeID, regra string, ref, ind float64) error {
	redeID = strings.TrimSpace(redeID)
	regra = strings.TrimSpace(regra)
	if redeID == "" {
		return errors.New("rede vazia")
	}
	if regra != "CADASTRAR" && regra != "PRIMEIRA_COMPRA_VOUCHER" {
		return errors.New("regra invalida")
	}
	if ref < 0 || ind < 0 {
		return errors.New("valores nao negativos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	const q = `
INSERT INTO rede_indique_ganhe_config (rede_id, regra, moedas_premio_referente, moedas_premio_indicado, atualizado_em)
VALUES ($1::uuid, $2, $3, $4, NOW())
ON CONFLICT (rede_id) DO UPDATE SET
  regra = EXCLUDED.regra,
  moedas_premio_referente = EXCLUDED.moedas_premio_referente,
  moedas_premio_indicado = EXCLUDED.moedas_premio_indicado,
  atualizado_em = NOW()
`
	_, err := r.db.ExecContext(ctx, q, redeID, regra, ref, ind)
	return err
}

func (r *indiqueGanhePostgres) InsertIndicacao(rede, referente, indicado, codInformado string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	const q = `
INSERT INTO indicacoes (rede_id, referente_usuario_id, indicado_usuario_id, codigo_informado)
VALUES ($1::uuid, $2::uuid, $3::uuid, NULLIF(trim($4), ''))
RETURNING id::text
`
	var id string
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(rede), strings.TrimSpace(referente), strings.TrimSpace(indicado), codInformado).Scan(&id)
	return id, err
}

func (r *indiqueGanhePostgres) BuscarIndicacaoPorIndicado(rede, indicadoID string) (*IndicacaoRegisto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  id::text, rede_id::text, referente_usuario_id::text, indicado_usuario_id::text,
  premiado_cadastro_referente, premiado_cadastro_indicado, premiado_compra_referente, premiado_compra_indicado
FROM indicacoes
WHERE rede_id = $1::uuid AND indicado_usuario_id = $2::uuid
`
	var x IndicacaoRegisto
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(rede), strings.TrimSpace(indicadoID)).Scan(
		&x.ID, &x.RedeID, &x.ReferenteUsuarioID, &x.IndicadoUsuarioID,
		&x.PremiadoCadastroRef, &x.PremiadoCadastroInd, &x.PremiadoCompraRef, &x.PremiadoCompraInd,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *indiqueGanhePostgres) MarcarPremioCadastro(rede, indicID string, refOK, indOK bool) error {
	return r.marcar(rede, indicID, "cadastro", refOK, indOK)
}

func (r *indiqueGanhePostgres) MarcarPremioCompra(rede, indicID string, refOK, indOK bool) error {
	return r.marcar(rede, indicID, "compra", refOK, indOK)
}

func (r *indiqueGanhePostgres) marcar(rede, indicID, qual string, refOK, indOK bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if qual == "cadastro" {
		const q = `UPDATE indicacoes SET
			premiado_cadastro_referente = $3,
			premiado_cadastro_indicado = $4
		WHERE id = $1::uuid AND rede_id = $2::uuid`
		_, err := r.db.ExecContext(ctx, q, indicID, strings.TrimSpace(rede), refOK, indOK)
		return err
	}
	const q2 = `UPDATE indicacoes SET
		premiado_compra_referente = $3,
		premiado_compra_indicado = $4
	WHERE id = $1::uuid AND rede_id = $2::uuid`
	_, err := r.db.ExecContext(ctx, q2, indicID, strings.TrimSpace(rede), refOK, indOK)
	return err
}

// ContarVouchersAprovadosUsuario quantidade de compras com status ATIVO ou USADO.
func (r *indiqueGanhePostgres) ContarVouchersAprovadosUsuario(rede, usuarioID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT COUNT(*)::int
FROM voucher_compras
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid AND status IN ('ATIVO', 'USADO')`
	var n int
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(rede), strings.TrimSpace(usuarioID)).Scan(&n)
	return n, err
}
