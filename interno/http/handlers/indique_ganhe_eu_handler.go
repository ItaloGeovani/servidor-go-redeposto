package handlers

import (
	"net/http"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

// GetEuIndiqueGanhe GET /v1/eu/indique-ganhe — codigo e rotulo da moeda (cliente autenticado).
func (h *Handlers) GetEuIndiqueGanhe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.indiqueGanhe == nil {
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
	red, errRede := h.redeService.BuscarPorID(rede)
	if errRede != nil || red == nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	m, err := h.indiqueGanhe.TelaIndiqueGanheEU(rede, uid, red)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao obter indique e ganhe")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, m)
}
