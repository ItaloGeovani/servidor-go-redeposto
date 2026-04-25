package servicos

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"

	"github.com/google/uuid"
)

const (
	tipoRefGireCusto  = "gire_ganhe"
	tipoRefGirePremio = "gire_ganhe_premio"
	qtdSlotsNormais   = 10
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

func slotValor(minV, maxV float64, idx int, totalSlots int) float64 {
	if totalSlots < 2 {
		totalSlots = 2
	}
	if idx < 0 {
		idx = 0
	}
	if idx > totalSlots-1 {
		idx = totalSlots - 1
	}
	minI := int(math.Round(minV))
	maxI := int(math.Round(maxV))
	if maxI < minI {
		maxI = minI
	}
	if maxI <= minI {
		return float64(minI)
	}
	passo := float64(maxI-minI) / float64(totalSlots-1)
	v := float64(minI) + passo*float64(idx)
	return float64(int(math.Round(v)))
}

type setorRoleta struct {
	Label      string
	Percentual float64
	Especial   bool
	ValorMoeda float64
}

func roletaPersonalizadaAtiva(cfg *repositorios.RedeGireGanheConfig) bool {
	if cfg == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(cfg.RoletaModo), "personalizado") && len(cfg.PremiosRoletaPersonalizada) > 0
}

func montarSetoresRoleta(cfg *repositorios.RedeGireGanheConfig) []setorRoleta {
	if cfg == nil {
		return []setorRoleta{}
	}
	if roletaPersonalizadaAtiva(cfg) {
		out := make([]setorRoleta, 0, len(cfg.PremiosRoletaPersonalizada))
		for _, p := range cfg.PremiosRoletaPersonalizada {
			v := float64(int(math.Round(p.ValorMoedas)))
			if v < 1 || p.Percentual <= 0 {
				continue
			}
			out = append(out, setorRoleta{
				Label:      strconv.Itoa(int(v)),
				Percentual: p.Percentual,
				Especial:   false,
				ValorMoeda: v,
			})
		}
		if len(out) > 0 {
			return out
		}
	}
	out := make([]setorRoleta, 0, qtdSlotsNormais+len(cfg.PremiosEspeciais))
	totEsp := totalPercentPremiosEspeciais(cfg)
	restante := 100 - totEsp
	if restante < 0 {
		restante = 0
	}
	pctNormal := restante / float64(qtdSlotsNormais)
	for i := 0; i < qtdSlotsNormais; i++ {
		v := slotValor(cfg.PremioMinMoeda, cfg.PremioMaxMoeda, i, qtdSlotsNormais)
		out = append(out, setorRoleta{
			Label:      strconv.Itoa(int(math.Round(v))),
			Percentual: pctNormal,
			Especial:   false,
			ValorMoeda: v,
		})
	}
	for _, p := range cfg.PremiosEspeciais {
		v := float64(int(math.Round(p.ValorMoedas)))
		if v <= 0 || p.Percentual <= 0 {
			continue
		}
		out = append(out, setorRoleta{
			Label:      strconv.Itoa(int(v)),
			Percentual: p.Percentual,
			Especial:   true,
			ValorMoeda: v,
		})
	}
	return out
}

func escolherSetorPorPeso(rng *rand.Rand, setores []setorRoleta) int {
	if len(setores) == 0 {
		return 0
	}
	var soma float64
	for _, s := range setores {
		if s.Percentual > 0 {
			soma += s.Percentual
		}
	}
	if soma <= 0 {
		return rng.Intn(len(setores))
	}
	r := rng.Float64() * soma
	var acc float64
	for i, s := range setores {
		if s.Percentual <= 0 {
			continue
		}
		acc += s.Percentual
		if r < acc {
			return i
		}
	}
	return len(setores) - 1
}

func setoresParaResposta(setores []setorRoleta) []map[string]any {
	out := make([]map[string]any, 0, len(setores))
	for _, s := range setores {
		out = append(out, map[string]any{
			"label":       s.Label,
			"percentual":  s.Percentual,
			"especial":    s.Especial,
			"valor_moeda": s.ValorMoeda,
		})
	}
	return out
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

func normalizarPremiosEspeciaisEntrada(maxMoedas float64, ativo bool, in []repositorios.GireGanhePremioEspecial) ([]repositorios.GireGanhePremioEspecial, error) {
	maxI := int(math.Round(maxMoedas))
	var out []repositorios.GireGanhePremioEspecial
	var soma float64
	for _, p := range in {
		v := p.ValorMoedas
		pc := p.Percentual
		if pc <= 0 || pc > 100 || v <= 0 {
			continue
		}
		vi := int(math.Round(v))
		if vi <= maxI {
			if ativo {
				return nil, ErrDadosInvalidos
			}
			continue
		}
		out = append(out, repositorios.GireGanhePremioEspecial{ValorMoedas: float64(vi), Percentual: pc})
		soma += pc
	}
	if soma > 100+1e-6 {
		return nil, ErrDadosInvalidos
	}
	if ativo && len(out) == 0 {
		return nil, ErrDadosInvalidos
	}
	return out, nil
}

func totalPercentPremiosEspeciais(cfg *repositorios.RedeGireGanheConfig) float64 {
	if cfg == nil {
		return 0
	}
	var t float64
	for _, p := range cfg.PremiosEspeciais {
		t += p.Percentual
	}
	return t
}

func premioEspecialMaxMoedas(cfg *repositorios.RedeGireGanheConfig) float64 {
	if cfg == nil || roletaPersonalizadaAtiva(cfg) || len(cfg.PremiosEspeciais) == 0 {
		return 0
	}
	var m float64
	for _, p := range cfg.PremiosEspeciais {
		if p.ValorMoedas > m {
			m = p.ValorMoedas
		}
	}
	return m
}

func valoresPremiosEspeciaisMoedas(cfg *repositorios.RedeGireGanheConfig) []float64 {
	if cfg == nil || roletaPersonalizadaAtiva(cfg) || len(cfg.PremiosEspeciais) == 0 {
		return []float64{}
	}
	out := make([]float64, 0, len(cfg.PremiosEspeciais))
	for _, p := range cfg.PremiosEspeciais {
		if p.ValorMoedas > 0 {
			out = append(out, p.ValorMoedas)
		}
	}
	return out
}

func normalizarRoletaPersonalizada(in []repositorios.GireGanhePremioEspecial) ([]repositorios.GireGanhePremioEspecial, error) {
	var out []repositorios.GireGanhePremioEspecial
	var soma float64
	for _, p := range in {
		pc := p.Percentual
		if pc <= 0 || pc > 100 {
			return nil, ErrDadosInvalidos
		}
		vi := int(math.Round(p.ValorMoedas))
		if vi < 1 {
			return nil, ErrDadosInvalidos
		}
		out = append(out, repositorios.GireGanhePremioEspecial{ValorMoedas: float64(vi), Percentual: pc})
		soma += pc
	}
	if len(out) < 1 {
		return nil, ErrDadosInvalidos
	}
	if math.Abs(soma-100) > 0.02 {
		return nil, ErrDadosInvalidos
	}
	return out, nil
}

func premiosRoletaPersonalizadaParaResposta(cfg *repositorios.RedeGireGanheConfig) []map[string]any {
	if cfg == nil || len(cfg.PremiosRoletaPersonalizada) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(cfg.PremiosRoletaPersonalizada))
	for _, p := range cfg.PremiosRoletaPersonalizada {
		out = append(out, map[string]any{"valor_moedas": p.ValorMoedas, "percentual": p.Percentual})
	}
	return out
}

func (s *ServicoGireGanhe) SalvarConfigGestor(redeID string, custo, minV, maxV float64, maxDia int, tz string, primeiroGratis bool, modo string, personal []repositorios.GireGanhePremioEspecial, premiosAtivo bool, premios []repositorios.GireGanhePremioEspecial) error {
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
	modoNorm := strings.ToLower(strings.TrimSpace(modo))
	if modoNorm != "personalizado" {
		modoNorm = "padrao"
	}
	var persNorm []repositorios.GireGanhePremioEspecial
	var espNorm []repositorios.GireGanhePremioEspecial
	var espAtivo bool
	if modoNorm == "personalizado" {
		var err error
		persNorm, err = normalizarRoletaPersonalizada(personal)
		if err != nil {
			return err
		}
		espAtivo = false
		espNorm = nil
	} else {
		modoNorm = "padrao"
		persNorm = nil
		var err error
		espNorm, err = normalizarPremiosEspeciaisEntrada(maxV, premiosAtivo, premios)
		if err != nil {
			return err
		}
		espAtivo = premiosAtivo
	}
	return s.repo.SalvarConfig(redeID, &repositorios.RedeGireGanheConfig{
		CustoMoedas:                custo,
		PremioMinMoeda:             minV,
		PremioMaxMoeda:             maxV,
		GirosMaxDia:                maxDia,
		Timezone:                   tz,
		PrimeiroGiroGratisAtivo:    primeiroGratis,
		RoletaModo:                 modoNorm,
		PremiosRoletaPersonalizada: persNorm,
		PremiosEspeciaisAtivo:      espAtivo,
		PremiosEspeciais:           espNorm,
	})
}

func (s *ServicoGireGanhe) ConfigPublicaParaRede(redeID string) map[string]any {
	cfg, err := s.repo.BuscarConfig(redeID)
	if err != nil || cfg == nil {
		return nil
	}
	totEsp := totalPercentPremiosEspeciais(cfg)
	ativo := !roletaPersonalizadaAtiva(cfg) && cfg.PremiosEspeciaisAtivo && len(cfg.PremiosEspeciais) > 0 && totEsp > 0
	return map[string]any{
		"gire_custo_moedas":                     cfg.CustoMoedas,
		"gire_premio_min_moedas":                cfg.PremioMinMoeda,
		"gire_premio_max_moedas":                cfg.PremioMaxMoeda,
		"gire_giros_max_dia":                    cfg.GirosMaxDia,
		"gire_timezone":                         cfg.Timezone,
		"gire_primeiro_giro_gratis_ativo":       cfg.PrimeiroGiroGratisAtivo,
		"gire_premios_especiais_ativo":          ativo,
		"gire_premio_especial_max_moedas":       premioEspecialMaxMoedas(cfg),
		"gire_premio_especial_chance_total_pct": totEsp,
		"gire_roleta_modo":                      strings.TrimSpace(cfg.RoletaModo),
		"gire_premios_roleta_personalizada":     premiosRoletaPersonalizadaParaResposta(cfg),
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
	totEsp := 0.0
	if !roletaPersonalizadaAtiva(cfg) {
		totEsp = totalPercentPremiosEspeciais(cfg)
	}
	espAtivo := !roletaPersonalizadaAtiva(cfg) && cfg.PremiosEspeciaisAtivo && len(cfg.PremiosEspeciais) > 0 && totEsp > 0
	setores := montarSetoresRoleta(cfg)
	return map[string]any{
		"app_modulo_gire_ganhe":            true,
		"moeda_virtual_nome":               strings.TrimSpace(r.MoedaVirtualNome),
		"custo_moedas":                     cfg.CustoMoedas,
		"premio_min_moedas":                cfg.PremioMinMoeda,
		"premio_max_moedas":                cfg.PremioMaxMoeda,
		"giros_max_dia":                    cfg.GirosMaxDia,
		"giros_feitos_hoje":                hoje,
		"giros_restantes_hoje":             max(0, cfg.GirosMaxDia-hoje),
		"primeiro_giro_gratis_disponivel":  gratis,
		"primeiro_giro_gratis_ativo":       cfg.PrimeiroGiroGratisAtivo,
		"pode_girar":                       hoje < cfg.GirosMaxDia,
		"multiplicador_moeda_nivel":        mult,
		"proximo_reset":                    proximoResetLocal(agora, loc).Format(time.RFC3339),
		"slots":                            len(setores),
		"setores":                          setoresParaResposta(setores),
		"premios_especiais_ativo":          espAtivo,
		"premio_especial_max_moedas":       premioEspecialMaxMoedas(cfg),
		"premio_especial_chance_total_pct": totEsp,
		"premios_especiais_moedas":         valoresPremiosEspeciaisMoedas(cfg),
		"roleta_modo":                      strings.TrimSpace(cfg.RoletaModo),
		"premios_roleta_personalizada":     premiosRoletaPersonalizadaParaResposta(cfg),
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

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	setores := montarSetoresRoleta(cfg)
	slotIdx := escolherSetorPorPeso(rng, setores)
	if slotIdx < 0 || slotIdx >= len(setores) {
		slotIdx = 0
	}
	setor := setores[slotIdx]
	esp := setor.Especial
	premioBase := setor.ValorMoeda
	mult := s.fatorMultMoedaUsuario(rede, usuarioID)
	premioCredito := float64(int(math.Round(premioBase * mult)))
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
		PremioEspecial:      esp,
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
		"slots":                      len(setores),
		"setores":                    setoresParaResposta(setores),
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
		"premio_especial":            esp,
		"roleta_modo":                strings.TrimSpace(cfg.RoletaModo),
	}, nil
}
