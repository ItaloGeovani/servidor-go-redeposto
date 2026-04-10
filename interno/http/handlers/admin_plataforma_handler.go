package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"gaspass-servidor/utils"
)

func (h *Handlers) ResumoRelatoriosPlataformaAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	resumo, err := h.estatisticasRepo.ResumoPlataforma()
	if err != nil {
		log.Printf("resumo relatorios plataforma admin: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar relatorios da plataforma")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"resumo": resumo,
	})
}

func (h *Handlers) ListarAuditoriaPlataformaAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	q := r.URL.Query()
	limite := 50
	if v := strings.TrimSpace(q.Get("limite")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro limite invalido (1 a 200)")
			return
		}
		limite = n
	}
	offset := 0
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro offset invalido")
			return
		}
		offset = n
	}
	idRede := strings.TrimSpace(q.Get("id_rede"))
	itens, total, err := h.auditoriaRepo.ListarPlataforma(idRede, limite, offset)
	if err != nil {
		log.Printf("listar auditoria plataforma admin: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar auditoria")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens":  itens,
		"total":  total,
		"limite": limite,
		"offset": offset,
	})
}

func (h *Handlers) ConfiguracaoSistemaAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"configuracao": map[string]any{
			"ambiente":                h.cfg.Ambiente,
			"porta_http":              h.cfg.PortaHTTP,
			"cors_origem_permitida":   h.cfg.CORSOrigemPermitida,
			"admin_bootstrap_ativado": h.cfg.AdminBootstrapAtivado,
		},
	})
}
