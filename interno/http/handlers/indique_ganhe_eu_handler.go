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
	cod, err := h.indiqueGanhe.MeuCodigoEU(rede, uid)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao obter codigo")
		return
	}
	red, err := h.redeService.BuscarPorID(rede)
	if err != nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{
			"codigo_indicacao": cod,
		})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"codigo_indicacao":         cod,
		"moeda_virtual_nome":       strings.TrimSpace(red.MoedaVirtualNome),
		"moeda_virtual_cotacao":    red.MoedaVirtualCotacao,
		"app_modulo_indique_ganhe": red.AppModuloIndiqueGanhe,
	})
}
