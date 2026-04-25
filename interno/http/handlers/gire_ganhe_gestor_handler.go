package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// GireGanheConfigGestor GET/PATCH /v1/gestor-rede/dev/redes/gire-ganhe
func (h *Handlers) GireGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getGireGanheConfigGestor(w, r)
	case http.MethodPatch:
		h.patchGireGanheConfigGestor(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getGireGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.gireGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	cfg, err := h.gireGanhe.BuscarConfigGestor(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"custo_moedas":               cfg.CustoMoedas,
		"premio_min_moedas":          cfg.PremioMinMoeda,
		"premio_max_moedas":          cfg.PremioMaxMoeda,
		"giros_max_dia":              cfg.GirosMaxDia,
		"timezone":                   strings.TrimSpace(cfg.Timezone),
		"primeiro_giro_gratis_ativo": cfg.PrimeiroGiroGratisAtivo,
	})
}

type reqGireGanheConfig struct {
	CustoMoedas             float64 `json:"custo_moedas"`
	PremioMinMoedas         float64 `json:"premio_min_moedas"`
	PremioMaxMoedas         float64 `json:"premio_max_moedas"`
	GirosMaxDia             int     `json:"giros_max_dia"`
	Timezone                string  `json:"timezone"`
	PrimeiroGiroGratisAtivo bool    `json:"primeiro_giro_gratis_ativo"`
}

func (h *Handlers) patchGireGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.gireGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqGireGanheConfig
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	tz := strings.TrimSpace(req.Timezone)
	if err := h.gireGanhe.SalvarConfigGestor(idRede, req.CustoMoedas, req.PremioMinMoedas, req.PremioMaxMoedas, req.GirosMaxDia, tz, req.PrimeiroGiroGratisAtivo); err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, "custo > 0, premio_max >= premio_min >= 0, giros_max_dia >= 1 e timezone valido")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		return
	}
	cfg, _ := h.gireGanhe.BuscarConfigGestor(idRede)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":                   "configuracao do gire e ganhe salva",
		"custo_moedas":               cfg.CustoMoedas,
		"premio_min_moedas":          cfg.PremioMinMoeda,
		"premio_max_moedas":          cfg.PremioMaxMoeda,
		"giros_max_dia":              cfg.GirosMaxDia,
		"timezone":                   cfg.Timezone,
		"primeiro_giro_gratis_ativo": cfg.PrimeiroGiroGratisAtivo,
	})
}
