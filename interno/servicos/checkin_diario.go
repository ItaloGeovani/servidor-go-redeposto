package servicos

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const tipoRefCheckinDiario = "checkin_diario"

var (
	ErrCheckinModuloDesligado = errors.New("modulo checkin diario desligado")
	ErrCheckinJaFeito         = errors.New("checkin ja realizado neste ciclo")
	ErrCheckinConfigInvalida  = errors.New("configuracao de checkin invalida")
	ErrCheckinCicloNaoAberto  = errors.New("o ciclo de check-in ainda nao comecou")
)

// ServicoCheckinDiario janela diaria pos hora local, bonus na carteira e sequencia.
type ServicoCheckinDiario struct {
	db     *sql.DB
	chk    repositorios.CheckinDiarioRepositorio
	cart   repositorios.CarteiraRepositorio
	niveis *ServicoNiveisCliente
	usu    ServicoUsuarioRede
}

func NovoServicoCheckinDiario(
	db *sql.DB,
	chk repositorios.CheckinDiarioRepositorio,
	cart repositorios.CarteiraRepositorio,
	niveis *ServicoNiveisCliente,
	usu ServicoUsuarioRede,
) *ServicoCheckinDiario {
	return &ServicoCheckinDiario{db: db, chk: chk, cart: cart, niveis: niveis, usu: usu}
}

func parseHoraAbertura(s string) (h, m int, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 12, 0, nil
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return 0, 0, ErrCheckinConfigInvalida
	}
	h, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || h < 0 || h > 23 {
		return 0, 0, ErrCheckinConfigInvalida
	}
	m, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || m < 0 || m > 59 {
		return 0, 0, ErrCheckinConfigInvalida
	}
	return h, m, nil
}

func computeCiclo(now time.Time, loc *time.Location, horaH, horaM int) (cicloStart time.Time, cicloKey time.Time) {
	n := now.In(loc)
	y, mo, d := n.Date()
	midnight := time.Date(y, mo, d, 0, 0, 0, 0, loc)
	openToday := midnight.Add(time.Duration(horaH)*time.Hour + time.Duration(horaM)*time.Minute)
	var start time.Time
	if n.Before(openToday) {
		pm := midnight.AddDate(0, 0, -1)
		py, pmo, pd := pm.Date()
		prevMid := time.Date(py, pmo, pd, 0, 0, 0, 0, loc)
		start = prevMid.Add(time.Duration(horaH)*time.Hour + time.Duration(horaM)*time.Minute)
	} else {
		start = openToday
	}
	sy, sm, sd := start.In(loc).Date()
	cicloKey = time.Date(sy, sm, sd, 0, 0, 0, 0, time.UTC)
	return start.UTC(), cicloKey
}

func proximoCicloInicio(cicloStartUTC time.Time, loc *time.Location) time.Time {
	return cicloStartUTC.In(loc).AddDate(0, 0, 1)
}

func mesmaDataUTC(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func diaAnteriorUTC(d time.Time) time.Time {
	return d.UTC().AddDate(0, 0, -1)
}

func (s *ServicoCheckinDiario) fatorMultMoedaUsuario(rede, usuarioID string) float64 {
	if s == nil || s.niveis == nil || s.usu == nil {
		return 1
	}
	cod, err := s.usu.ObterNivelCliente(usuarioID, rede)
	if err != nil {
		return s.niveis.FatorMultMoeda(rede, "bronze")
	}
	cod = strings.ToLower(strings.TrimSpace(cod))
	if cod == "" {
		cod = "bronze"
	}
	return s.niveis.FatorMultMoeda(rede, cod)
}

// BuscarConfigGestor config persistida ou padroes.
func (s *ServicoCheckinDiario) BuscarConfigGestor(redeID string) (*repositorios.RedeCheckinDiarioConfig, error) {
	if s == nil || s.chk == nil {
		return nil, errors.New("checkin indisponivel")
	}
	return s.chk.BuscarConfig(redeID)
}

// SalvarConfigGestor valida timezone e hora.
func (s *ServicoCheckinDiario) SalvarConfigGestor(redeID string, moedas float64, horaAbertura, tz string) error {
	if s == nil || s.chk == nil {
		return errors.New("checkin indisponivel")
	}
	if moedas <= 0 {
		return ErrDadosInvalidos
	}
	tz = strings.TrimSpace(tz)
	if tz == "" {
		tz = "America/Sao_Paulo"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ErrDadosInvalidos
	}
	if _, _, err := parseHoraAbertura(horaAbertura); err != nil {
		return ErrDadosInvalidos
	}
	c := &repositorios.RedeCheckinDiarioConfig{
		MoedasPorDia: moedas,
		HoraAbertura: strings.TrimSpace(horaAbertura),
		Timezone:     tz,
	}
	return s.chk.SalvarConfig(redeID, c)
}

// ConfigPublicaParaRede valores para /v1/public/rede-info (sem auth).
func (s *ServicoCheckinDiario) ConfigPublicaParaRede(redeID string) map[string]any {
	if s == nil || s.chk == nil {
		return nil
	}
	cfg, err := s.chk.BuscarConfig(redeID)
	if err != nil || cfg == nil {
		return nil
	}
	return map[string]any{
		"checkin_moedas_por_dia": cfg.MoedasPorDia,
		"checkin_hora_abertura": strings.TrimSpace(cfg.HoraAbertura),
		"checkin_timezone":      strings.TrimSpace(cfg.Timezone),
	}
}

// EstadoCheckinEU payload GET /v1/eu/checkin-diario.
func (s *ServicoCheckinDiario) EstadoCheckinEU(rede, usuarioID string, r *modelos.Rede, agora time.Time) (map[string]any, error) {
	if s == nil || r == nil || !r.AppModuloCheckinDiario {
		return nil, ErrCheckinModuloDesligado
	}
	cfg, err := s.chk.BuscarConfig(rede)
	if err != nil {
		return nil, err
	}
	loc, lerr := time.LoadLocation(strings.TrimSpace(cfg.Timezone))
	if lerr != nil {
		loc = time.UTC
	}
	hh, mm, herr := parseHoraAbertura(cfg.HoraAbertura)
	if herr != nil {
		return nil, herr
	}
	cicloStart, cicloKey := computeCiclo(agora, loc, hh, mm)
	mult := s.fatorMultMoedaUsuario(rede, usuarioID)
	valor := cfg.MoedasPorDia * mult

	reg, err := s.chk.BuscarRegistoCiclo(rede, usuarioID, cicloKey)
	if err != nil {
		return nil, err
	}
	jaFeito := reg != nil

	var streakEx int
	if jaFeito {
		streakEx = reg.Streak
	} else {
		ult, uerr := s.chk.UltimoRegisto(rede, usuarioID)
		if uerr != nil {
			return nil, uerr
		}
		if ult != nil && mesmaDataUTC(ult.CicloKey, diaAnteriorUTC(cicloKey)) {
			streakEx = ult.Streak
		}
	}

	pode := !jaFeito && !agora.UTC().Before(cicloStart.UTC())

	var prox time.Time
	if jaFeito {
		prox = proximoCicloInicio(cicloStart, loc)
	} else if agora.UTC().Before(cicloStart.UTC()) {
		prox = cicloStart
	} else {
		prox = proximoCicloInicio(cicloStart, loc)
	}

	out := map[string]any{
		"app_modulo_checkin_diario": true,
		"moeda_virtual_nome":        strings.TrimSpace(r.MoedaVirtualNome),
		"moedas_base":               cfg.MoedasPorDia,
		"mult_moeda_nivel":          mult,
		"moedas_creditadas":         valor,
		"hora_abertura":             cfg.HoraAbertura,
		"timezone":                  cfg.Timezone,
		"ciclo_inicio":              cicloStart.UTC().Format(time.RFC3339),
		"proximo_ciclo_inicio":      prox.UTC().Format(time.RFC3339),
		"ja_feito_este_ciclo":       jaFeito,
		"pode_checkin":              pode,
		"streak_dias":               streakEx,
	}
	return out, nil
}

// RegistrarCheckinEU POST — idempotente por ciclo.
func (s *ServicoCheckinDiario) RegistrarCheckinEU(rede, usuarioID string, r *modelos.Rede, agora time.Time) (map[string]any, error) {
	if s == nil || r == nil || !r.AppModuloCheckinDiario {
		return nil, ErrCheckinModuloDesligado
	}
	cfg, err := s.chk.BuscarConfig(rede)
	if err != nil {
		return nil, err
	}
	loc, lerr := time.LoadLocation(strings.TrimSpace(cfg.Timezone))
	if lerr != nil {
		loc = time.UTC
	}
	hh, mm, herr := parseHoraAbertura(cfg.HoraAbertura)
	if herr != nil {
		return nil, herr
	}
	cicloStart, cicloKey := computeCiclo(agora, loc, hh, mm)
	if agora.UTC().Before(cicloStart.UTC()) {
		return nil, ErrCheckinCicloNaoAberto
	}
	ex, err := s.chk.BuscarRegistoCiclo(rede, usuarioID, cicloKey)
	if err != nil {
		return nil, err
	}
	if ex != nil {
		return nil, ErrCheckinJaFeito
	}

	ult, err := s.chk.UltimoRegisto(rede, usuarioID)
	if err != nil {
		return nil, err
	}
	streak := 1
	if ult != nil && mesmaDataUTC(ult.CicloKey, diaAnteriorUTC(cicloKey)) {
		streak = ult.Streak + 1
	}

	mult := s.fatorMultMoedaUsuario(rede, usuarioID)
	valor := cfg.MoedasPorDia * mult
	if valor <= 0 {
		return nil, ErrCheckinConfigInvalida
	}

	id := uuid.NewString()
	carteiraID, err := s.cart.ObterOuCriarCarteira(rede, usuarioID, strings.TrimSpace(r.MoedaVirtualNome), r.MoedaVirtualCotacao)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.chk.InserirRegistoTx(ctx, tx, id, rede, usuarioID, cicloKey, valor, streak); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrCheckinJaFeito
		}
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return nil, ErrCheckinJaFeito
		}
		return nil, err
	}
	if err := s.cart.CreditarBonusTx(ctx, tx, rede, carteiraID, valor, tipoRefCheckinDiario, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	saldo, _ := s.cart.ObterSaldoToken(rede, usuarioID)
	return map[string]any{
		"mensagem":          "checkin registrado",
		"moedas_creditadas": valor,
		"streak_dias":       streak,
		"mult_moeda_nivel":  mult,
		"saldo_moeda":       saldo,
	}, nil
}
