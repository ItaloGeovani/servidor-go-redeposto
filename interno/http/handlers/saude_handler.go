package handlers

import (
	"net/http"
	"time"

	"gaspass-servidor/utils"
)

func (h *Handlers) Saude(w http.ResponseWriter, r *http.Request) {
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"servico":     "gaspass-servidor",
		"data_hora":   time.Now().UTC().Format(time.RFC3339),
		"mensagem":    "API operacional",
		"documentado": true,
	})
}
