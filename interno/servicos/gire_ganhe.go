package servicos

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"math/rand"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"

	"github.com/google/uuid"
)

const (
	tipoRefGireCusto  = "gire_ganhe"
	tipoRefGirePremio = "gire_ganhe_premio"
	qtdSlotsRoleta    = 12
)

var (
	ErrGireModuloDesligado = errors.New("modulo gire e ganhe desligado")
	ErrGireLimiteDiario    = errors.New("limite diario de giros atingido")
)

type ServicoGireGanhe struct {
	db     *sql.DB
	repo   repositorios.GireGanheRepositorio
	cart   repositorios.CarteiraRepositorio
	niveis *ServicoNiveisCliente
	usu    ServicoUsuarioRede
}

func NovoServicoGireGanhe(db *sql.DB, repo repositorios.GireGanheRepositorio, cart repositorios.CarteiraRepositorio, niveis *ServicoNiveisCliente, usu ServicoUsuarioRede) *ServicoGireGanhe {
	return &ServicoGireGanhe{db: db, repo: repo, cart: cart, niveis: niveis, usu: usu}
}

func (s *ServicoGireGanhe) fatorMultMoedaUsuario(rede, usuarioID string) float64 {
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

func slotValor(minV, maxV float64, idx int) float64 {
	if idx < 0 {
		idx = 0
	}
	if idx > qtdSlotsRoleta-1 {
		idx = qtdSlotsRoleta - 1
	}
	if maxV <= minV {
		return minV
	}
	passo := (maxV - minV) / float64(qtdSlotsRoleta-1)
	v := minV + passo*float64(idx)
	return math.Round(v*100) / 100
}

func cicloDiaLocal(now time.Time, loc *time.Location) time.Time {
	n := now.In(loc)
	y, m, d := n.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func proximoResetLocal(now time.Time, loc *time.Location) time.Time {
	n := now.In(loc)
	y, m, d := n.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, loc).UTC()
}

func (s *ServicoGireGanhe) BuscarConfigGestor(redeID string) (*repositorios.RedeGireGanheConfig, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("gire e ganhe indisponivel")
	}
	return s.repo.BuscarConfig(redeID)
}

func (s *ServicoGireGanhe) SalvarConfigGestor(redeID string, custo, minV, maxV float64, maxDia int, tz string, primeiroGratis bool) error {
	if s == nil || s.repo == nil {
		return errors.New("gire e ganhe indisponivel")
	}
	if custo <= 0 || minV < 0 || maxV < minV || maxDia < 1 || maxDia > 100 {
		return ErrDadosInvalidos
	}
	tz = strings.TrimSpace(tz)
	if tz == "" {
		tz = "America/Sao_Paulo"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ErrDadosInvalidos
	}
	return s.repo.SalvarConfig(redeID, &repositorios.RedeGireGanheConfig{
		CustoMoedas:             custo,
		PremioMinMoeda:          minV,
		PremioMaxMoeda:          maxV,
		GirosMaxDia:             maxDia,
		Timezone:                tz,
		PrimeiroGiroGratisAtivo: primeiroGratis,
	})
}

func (s *ServicoGireGanhe) ConfigPublicaParaRede(redeID string) map[string]any {
	cfg, err := s.repo.BuscarConfig(redeID)
	if err != nil || cfg == nil {
		return nil
	}
	return map[string]any{
		"gire_custo_moedas":               cfg.CustoMoedas,
		"gire_premio_min_moedas":          cfg.PremioMinMoeda,
		"gire_premio_max_moedas":          cfg.PremioMaxMoeda,
		"gire_giros_max_dia":              cfg.GirosMaxDia,
		"gire_timezone":                   cfg.Timezone,
		"gire_primeiro_giro_gratis_ativo": cfg.PrimeiroGiroGratisAtivo,
	}
}

func (s *ServicoGireGanhe) EstadoEU(rede, usuarioID string, r *modelos.Rede, agora time.Time) (map[string]any, error) {
	if s == nil || r == nil || !r.AppModuloGireGanhe {
		return nil, ErrGireModuloDesligado
	}
	cfg, err := s.repo.BuscarConfig(rede)
	if err != nil {
		return nil, err
	}
	loc, lerr := time.LoadLocation(strings.TrimSpace(cfg.Timezone))
	if lerr != nil {
		loc = time.UTC
	}
	dia := cicloDiaLocal(agora, loc)
	hoje, err := s.repo.ContarGirosNoDia(rede, usuarioID, dia)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.ContarTotalGiros(rede, usuarioID)
	if err != nil {
		return nil, err
	}
	gratis := cfg.PrimeiroGiroGratisAtivo && total == 0
	mult := s.fatorMultMoedaUsuario(rede, usuarioID)
	return map[string]any{
		"app_modulo_gire_ganhe":           true,
		"moeda_virtual_nome":              strings.TrimSpace(r.MoedaVirtualNome),
		"custo_moedas":                    cfg.CustoMoedas,
		"premio_min_moedas":               cfg.PremioMinMoeda,
		"premio_max_moedas":               cfg.PremioMaxMoeda,
		"giros_max_dia":                   cfg.GirosMaxDia,
		"giros_feitos_hoje":               hoje,
		"giros_restantes_hoje":            max(0, cfg.GirosMaxDia-hoje),
		"primeiro_giro_gratis_disponivel": gratis,
		"primeiro_giro_gratis_ativo":      cfg.PrimeiroGiroGratisAtivo,
		"pode_girar":                      hoje < cfg.GirosMaxDia,
		"multiplicador_moeda_nivel":       mult,
		"proximo_reset":                   proximoResetLocal(agora, loc).Format(time.RFC3339),
		"slots":                           qtdSlotsRoleta,
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *ServicoGireGanhe) GirarEU(rede, usuarioID string, r *modelos.Rede, agora time.Time) (map[string]any, error) {
	if s == nil || r == nil || !r.AppModuloGireGanhe {
		return nil, ErrGireModuloDesligado
	}
	cfg, err := s.repo.BuscarConfig(rede)
	if err != nil {
		return nil, err
	}
	loc, lerr := time.LoadLocation(strings.TrimSpace(cfg.Timezone))
	if lerr != nil {
		loc = time.UTC
	}
	dia := cicloDiaLocal(agora, loc)
	hoje, err := s.repo.ContarGirosNoDia(rede, usuarioID, dia)
	if err != nil {
		return nil, err
	}
	if hoje >= cfg.GirosMaxDia {
		return nil, ErrGireLimiteDiario
	}
	total, err := s.repo.ContarTotalGiros(rede, usuarioID)
	if err != nil {
		return nil, err
	}
	gratis := cfg.PrimeiroGiroGratisAtivo && total == 0

	slotIdx := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(qtdSlotsRoleta)
	premioBase := slotValor(cfg.PremioMinMoeda, cfg.PremioMaxMoeda, slotIdx)
	mult := s.fatorMultMoedaUsuario(rede, usuarioID)
	premioCredito := math.Round((premioBase*mult)*100) / 100
	if premioCredito < 0 {
		premioCredito = 0
	}

	id := uuid.NewString()
	custoDebitado := 0.0
	if !gratis {
		custoDebitado = cfg.CustoMoedas
	}
	if _, err := s.cart.ObterOuCriarCarteira(rede, usuarioID, strings.TrimSpace(r.MoedaVirtualNome), r.MoedaVirtualCotacao); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if !gratis && cfg.CustoMoedas > 0 {
		if err := s.cart.DebitarMoedaTx(ctx, tx, rede, usuarioID, cfg.CustoMoedas, tipoRefGireCusto, id); err != nil {
			return nil, err
		}
	}
	if err := s.repo.InserirGiroTx(ctx, tx, &repositorios.GireGanheGiro{
		ID:                  id,
		CicloDia:            dia,
		NumeroNoDia:         hoje + 1,
		SlotIndex:           slotIdx,
		PremioBaseMoedas:    premioBase,
		PremioCreditoMoedas: premioCredito,
		MultiplicadorNivel:  mult,
		CustoDebitadoMoedas: custoDebitado,
		GiroGratis:          gratis,
	}, rede, usuarioID); err != nil {
		return nil, err
	}
	if premioCredito > 0 {
		cid, err := s.cart.ObterOuCriarCarteira(rede, usuarioID, strings.TrimSpace(r.MoedaVirtualNome), r.MoedaVirtualCotacao)
		if err != nil {
			return nil, err
		}
		if err := s.cart.CreditarBonusTx(ctx, tx, rede, cid, premioCredito, tipoRefGirePremio, id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	saldo, _ := s.cart.ObterSaldoToken(rede, usuarioID)
	return map[string]any{
		"mensagem":                   "giro realizado",
		"slot_index":                 slotIdx,
		"slots":                      qtdSlotsRoleta,
		"premio_base_moedas":         premioBase,
		"premio_creditado_moedas":    premioCredito,
		"multiplicador_moeda_nivel":  mult,
		"custo_debitado_moedas":      custoDebitado,
		"primeiro_giro_gratis_usado": gratis,
		"saldo_moeda":                saldo,
		"numero_no_dia":              hoje + 1,
		"moeda_virtual_nome":         strings.TrimSpace(r.MoedaVirtualNome),
		"premio_min_moedas":          cfg.PremioMinMoeda,
		"premio_max_moedas":          cfg.PremioMaxMoeda,
	}, nil
}
