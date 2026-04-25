package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type RedeGireGanheConfig struct {
	CustoMoedas             float64
	PremioMinMoeda          float64
	PremioMaxMoeda          float64
	GirosMaxDia             int
	Timezone                string
	PrimeiroGiroGratisAtivo bool
}

type GireGanheGiro struct {
	ID                  string
	CicloDia            time.Time
	NumeroNoDia         int
	SlotIndex           int
	PremioBaseMoedas    float64
	PremioCreditoMoedas float64
	MultiplicadorNivel  float64
	CustoDebitadoMoedas float64
	GiroGratis          bool
	CriadoEm            time.Time
}

type GireGanheRepositorio interface {
	BuscarConfig(redeID string) (*RedeGireGanheConfig, error)
	SalvarConfig(redeID string, c *RedeGireGanheConfig) error
	ContarTotalGiros(redeID, usuarioID string) (int, error)
	ContarGirosNoDia(redeID, usuarioID string, cicloDia time.Time) (int, error)
	InserirGiroTx(ctx context.Context, tx *sql.Tx, g *GireGanheGiro, redeID, usuarioID string) error
}

type gireGanhePostgres struct{ db *sql.DB }

func NovoGireGanhePostgres(db *sql.DB) GireGanheRepositorio { return &gireGanhePostgres{db: db} }

func padraoGireConfig() *RedeGireGanheConfig {
	return &RedeGireGanheConfig{
		CustoMoedas: 10, PremioMinMoeda: 1, PremioMaxMoeda: 20, GirosMaxDia: 1,
		Timezone: "America/Sao_Paulo", PrimeiroGiroGratisAtivo: true,
	}
}

func (r *gireGanhePostgres) BuscarConfig(redeID string) (*RedeGireGanheConfig, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return nil, errors.New("rede invalida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT custo_moedas::float8, premio_min_moedas::float8, premio_max_moedas::float8, giros_max_dia, trim(timezone), COALESCE(primeiro_giro_gratis_ativo, true)
FROM rede_gire_ganhe_config
WHERE rede_id = $1::uuid`
	var c RedeGireGanheConfig
	err := r.db.QueryRowContext(ctx, q, redeID).Scan(&c.CustoMoedas, &c.PremioMinMoeda, &c.PremioMaxMoeda, &c.GirosMaxDia, &c.Timezone, &c.PrimeiroGiroGratisAtivo)
	if errors.Is(err, sql.ErrNoRows) {
		return padraoGireConfig(), nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.Timezone) == "" {
		c.Timezone = "America/Sao_Paulo"
	}
	if c.CustoMoedas <= 0 {
		c.CustoMoedas = 10
	}
	if c.PremioMinMoeda < 0 {
		c.PremioMinMoeda = 0
	}
	if c.PremioMaxMoeda < c.PremioMinMoeda {
		c.PremioMaxMoeda = c.PremioMinMoeda
	}
	if c.GirosMaxDia < 1 {
		c.GirosMaxDia = 1
	}
	return &c, nil
}

func (r *gireGanhePostgres) SalvarConfig(redeID string, c *RedeGireGanheConfig) error {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" || c == nil {
		return errors.New("dados invalidos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const up = `
INSERT INTO rede_gire_ganhe_config (rede_id, custo_moedas, premio_min_moedas, premio_max_moedas, giros_max_dia, timezone, primeiro_giro_gratis_ativo)
VALUES ($1::uuid, $2::numeric, $3::numeric, $4::numeric, $5, $6, $7)
ON CONFLICT (rede_id) DO UPDATE SET
  custo_moedas = EXCLUDED.custo_moedas,
  premio_min_moedas = EXCLUDED.premio_min_moedas,
  premio_max_moedas = EXCLUDED.premio_max_moedas,
  giros_max_dia = EXCLUDED.giros_max_dia,
  timezone = EXCLUDED.timezone,
  primeiro_giro_gratis_ativo = EXCLUDED.primeiro_giro_gratis_ativo,
  atualizado_em = NOW()`
	_, err := r.db.ExecContext(ctx, up, redeID, c.CustoMoedas, c.PremioMinMoeda, c.PremioMaxMoeda, c.GirosMaxDia, strings.TrimSpace(c.Timezone), c.PrimeiroGiroGratisAtivo)
	return err
}

func (r *gireGanhePostgres) ContarTotalGiros(redeID, usuarioID string) (int, error) {
	redeID, usuarioID = strings.TrimSpace(redeID), strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return 0, errors.New("ids invalidos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `SELECT COUNT(1)::int FROM gire_ganhe_giros WHERE rede_id = $1::uuid AND usuario_id = $2::uuid`
	var n int
	if err := r.db.QueryRowContext(ctx, q, redeID, usuarioID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *gireGanhePostgres) ContarGirosNoDia(redeID, usuarioID string, cicloDia time.Time) (int, error) {
	redeID, usuarioID = strings.TrimSpace(redeID), strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return 0, errors.New("ids invalidos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `SELECT COUNT(1)::int FROM gire_ganhe_giros WHERE rede_id = $1::uuid AND usuario_id = $2::uuid AND ciclo_dia = $3::date`
	var n int
	if err := r.db.QueryRowContext(ctx, q, redeID, usuarioID, cicloDia).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *gireGanhePostgres) InserirGiroTx(ctx context.Context, tx *sql.Tx, g *GireGanheGiro, redeID, usuarioID string) error {
	if g == nil {
		return errors.New("giro invalido")
	}
	redeID, usuarioID = strings.TrimSpace(redeID), strings.TrimSpace(usuarioID)
	g.ID = strings.TrimSpace(g.ID)
	if redeID == "" || usuarioID == "" || g.ID == "" || g.NumeroNoDia < 1 || g.SlotIndex < 0 || g.SlotIndex > 11 {
		return errors.New("dados invalidos para giro")
	}
	const ins = `
INSERT INTO gire_ganhe_giros (
  id, rede_id, usuario_id, ciclo_dia, numero_no_dia, slot_index,
  premio_base_moedas, premio_creditado_moedas, multiplicador_nivel, custo_debitado_moedas, giro_gratis
) VALUES (
  $1::uuid, $2::uuid, $3::uuid, $4::date, $5, $6,
  $7::numeric, $8::numeric, $9::numeric, $10::numeric, $11
)`
	_, err := tx.ExecContext(ctx, ins, g.ID, redeID, usuarioID, g.CicloDia, g.NumeroNoDia, g.SlotIndex,
		g.PremioBaseMoedas, g.PremioCreditoMoedas, g.MultiplicadorNivel, g.CustoDebitadoMoedas, g.GiroGratis)
	return err
}
