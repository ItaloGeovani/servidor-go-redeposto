package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// RedeCheckinDiarioConfig valores persistidos ou padrao em memoria.
type RedeCheckinDiarioConfig struct {
	MoedasPorDia float64
	HoraAbertura string // "HH:MM" 24h
	Timezone     string
}

// CheckinDiarioRegisto uma reivindicacao de ciclo.
type CheckinDiarioRegisto struct {
	ID           string
	CicloKey     time.Time // data (UTC meia-noite) do ancoramento do ciclo
	ValorMoedas  float64
	Streak       int
	CriadoEm     time.Time
	UsuarioID    string
}

// CheckinDiarioRepositorio persistencia do check-in diario.
type CheckinDiarioRepositorio interface {
	BuscarConfig(redeID string) (*RedeCheckinDiarioConfig, error)
	SalvarConfig(redeID string, c *RedeCheckinDiarioConfig) error
	BuscarRegistoCiclo(redeID, usuarioID string, cicloKey time.Time) (*CheckinDiarioRegisto, error)
	UltimoRegisto(redeID, usuarioID string) (*CheckinDiarioRegisto, error)
	InserirRegistoTx(ctx context.Context, tx *sql.Tx, id, redeID, usuarioID string, cicloKey time.Time, valor float64, streak int) error
}

type checkinDiarioPostgres struct {
	db *sql.DB
}

func NovoCheckinDiarioPostgres(db *sql.DB) CheckinDiarioRepositorio {
	return &checkinDiarioPostgres{db: db}
}

func padraoCheckinConfig() *RedeCheckinDiarioConfig {
	return &RedeCheckinDiarioConfig{
		MoedasPorDia: 10,
		HoraAbertura: "12:00",
		Timezone:     "America/Sao_Paulo",
	}
}

func (r *checkinDiarioPostgres) BuscarConfig(redeID string) (*RedeCheckinDiarioConfig, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return nil, errors.New("rede invalida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT moedas_por_dia::float8, trim(to_char(hora_abertura, 'HH24:MI')), trim(timezone)
FROM rede_checkin_diario_config WHERE rede_id = $1::uuid`
	var c RedeCheckinDiarioConfig
	err := r.db.QueryRowContext(ctx, q, redeID).Scan(&c.MoedasPorDia, &c.HoraAbertura, &c.Timezone)
	if errors.Is(err, sql.ErrNoRows) {
		p := padraoCheckinConfig()
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.Timezone) == "" {
		c.Timezone = padraoCheckinConfig().Timezone
	}
	if strings.TrimSpace(c.HoraAbertura) == "" {
		c.HoraAbertura = "12:00"
	}
	if c.MoedasPorDia <= 0 {
		c.MoedasPorDia = 10
	}
	return &c, nil
}

func (r *checkinDiarioPostgres) SalvarConfig(redeID string, c *RedeCheckinDiarioConfig) error {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" || c == nil {
		return errors.New("dados invalidos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const upsert = `
INSERT INTO rede_checkin_diario_config (rede_id, moedas_por_dia, hora_abertura, timezone)
VALUES ($1::uuid, $2::numeric, $3::time, $4)
ON CONFLICT (rede_id) DO UPDATE SET
  moedas_por_dia = EXCLUDED.moedas_por_dia,
  hora_abertura = EXCLUDED.hora_abertura,
  timezone = EXCLUDED.timezone,
  atualizado_em = NOW()`
	_, err := r.db.ExecContext(ctx, upsert, redeID, c.MoedasPorDia, c.HoraAbertura, strings.TrimSpace(c.Timezone))
	return err
}

func (r *checkinDiarioPostgres) BuscarRegistoCiclo(redeID, usuarioID string, cicloKey time.Time) (*CheckinDiarioRegisto, error) {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return nil, errors.New("ids invalidos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT id::text, ciclo_key, valor_moedas::float8, streak, criado_em, usuario_id::text
FROM checkin_diario_registos
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid AND ciclo_key = $3::date`
	var x CheckinDiarioRegisto
	err := r.db.QueryRowContext(ctx, q, redeID, usuarioID, cicloKey).Scan(
		&x.ID, &x.CicloKey, &x.ValorMoedas, &x.Streak, &x.CriadoEm, &x.UsuarioID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *checkinDiarioPostgres) UltimoRegisto(redeID, usuarioID string) (*CheckinDiarioRegisto, error) {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return nil, errors.New("ids invalidos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT id::text, ciclo_key, valor_moedas::float8, streak, criado_em, usuario_id::text
FROM checkin_diario_registos
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid
ORDER BY ciclo_key DESC
LIMIT 1`
	var x CheckinDiarioRegisto
	err := r.db.QueryRowContext(ctx, q, redeID, usuarioID).Scan(
		&x.ID, &x.CicloKey, &x.ValorMoedas, &x.Streak, &x.CriadoEm, &x.UsuarioID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *checkinDiarioPostgres) InserirRegistoTx(ctx context.Context, tx *sql.Tx, id, redeID, usuarioID string, cicloKey time.Time, valor float64, streak int) error {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	id = strings.TrimSpace(id)
	if redeID == "" || usuarioID == "" || id == "" || valor <= 0 || streak < 1 {
		return errors.New("dados invalidos checkin")
	}
	const ins = `
INSERT INTO checkin_diario_registos (id, rede_id, usuario_id, ciclo_key, valor_moedas, streak)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::date, $5::numeric, $6)`
	_, err := tx.ExecContext(ctx, ins, id, redeID, usuarioID, cicloKey, valor, streak)
	return err
}
