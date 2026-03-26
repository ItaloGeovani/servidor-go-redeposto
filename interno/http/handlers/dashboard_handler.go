package handlers

import (
	"net/http"

	"gaspass-servidor/utils"
)

func (h *Handlers) ResumoDashboardAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	redes, err := h.redeService.Listar()
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar resumo de redes")
		return
	}

	totalGestores, gestoresAtivos, err := h.gestorService.Contar()
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar resumo de gestores")
		return
	}

	totalRedes := len(redes)
	redesAtivas := 0
	redesInativas := 0
	receitaMensalPrevista := 0.0
	receitaImplantacaoPrevista := 0.0

	for _, rede := range redes {
		receitaImplantacaoPrevista += rede.ValorImplantacao
		if rede.Ativa {
			redesAtivas++
			receitaMensalPrevista += rede.ValorMensalidade
		} else {
			redesInativas++
		}
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"resumo": map[string]any{
			"total_redes":                  totalRedes,
			"redes_ativas":                 redesAtivas,
			"redes_inativas":               redesInativas,
			"total_gestores":               totalGestores,
			"gestores_ativos":              gestoresAtivos,
			"gestores_inativos":            totalGestores - gestoresAtivos,
			"receita_mensal_prevista":      receitaMensalPrevista,
			"receita_implantacao_prevista": receitaImplantacaoPrevista,
		},
	})
}
