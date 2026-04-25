package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// CheckinDiarioEU GET/POST /v1/eu/checkin-diario — estado e registo (cliente).
func (h *Handlers) CheckinDiarioEU(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getEuCheckinDiario(w, r)
	case http.MethodPost:
		h.postEuCheckinDiario(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getEuCheckinDiario(w http.ResponseWriter, r *http.Request) {
	if h.checkinDiario == nil {
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
	m, err := h.checkinDiario.EstadoCheckinEU(rede, uid, red, time.Now())
	if err != nil {
		if errors.Is(err, servicos.ErrCheckinModuloDesligado) {
			utils.ResponderErro(w, http.StatusForbidden, "modulo check-in diario desligado")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao obter check-in")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, m)
}

func (h *Handlers) postEuCheckinDiario(w http.ResponseWriter, r *http.Request) {
	if h.checkinDiario == nil {
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
	m, err := h.checkinDiario.RegistrarCheckinEU(rede, uid, red, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrCheckinModuloDesligado):
			utils.ResponderErro(w, http.StatusForbidden, "modulo check-in diario desligado")
		case errors.Is(err, servicos.ErrCheckinJaFeito):
			utils.ResponderErro(w, http.StatusConflict, "check-in ja feito neste ciclo")
		case errors.Is(err, servicos.ErrCheckinCicloNaoAberto):
			utils.ResponderErro(w, http.StatusBadRequest, "o ciclo de check-in ainda nao comecou")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao registrar check-in")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, m)
}
