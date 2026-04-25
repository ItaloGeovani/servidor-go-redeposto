package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// GireGanheEU GET/POST /v1/eu/gire-ganhe
func (h *Handlers) GireGanheEU(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getEuGireGanhe(w, r)
	case http.MethodPost:
		h.postEuGireGanhe(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getEuGireGanhe(w http.ResponseWriter, r *http.Request) {
	if h.gireGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes do app")
		return
	}
	rede := strings.TrimSpace(u.IDRede)
	uid := strings.TrimSpace(u.IDUsuario)
	if rede == "" || uid == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "sessao sem rede")
		return
	}
	red, err := h.redeService.BuscarPorID(rede)
	if err != nil || red == nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	m, err := h.gireGanhe.EstadoEU(rede, uid, red, time.Now())
	if err != nil {
		if errors.Is(err, servicos.ErrGireModuloDesligado) {
			utils.ResponderErro(w, http.StatusForbidden, "modulo gire e ganhe desligado")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao obter estado")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, m)
}

func (h *Handlers) postEuGireGanhe(w http.ResponseWriter, r *http.Request) {
	if h.gireGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes do app")
		return
	}
	rede := strings.TrimSpace(u.IDRede)
	uid := strings.TrimSpace(u.IDUsuario)
	if rede == "" || uid == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "sessao sem rede")
		return
	}
	red, err := h.redeService.BuscarPorID(rede)
	if err != nil || red == nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	m, err := h.gireGanhe.GirarEU(rede, uid, red, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrGireModuloDesligado):
			utils.ResponderErro(w, http.StatusForbidden, "modulo gire e ganhe desligado")
		case errors.Is(err, servicos.ErrGireLimiteDiario):
			utils.ResponderErro(w, http.StatusConflict, "limite diario de giros atingido")
		case errors.Is(err, repositorios.ErrSaldoInsuficiente):
			utils.ResponderErro(w, http.StatusBadRequest, "saldo insuficiente para girar")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao girar")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, m)
}
