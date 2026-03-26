package handlers

import (
	"net/http"
	"runtime"
	"time"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/utils"
)

func (h *Handlers) DiagnosticoAdmin(w http.ResponseWriter, r *http.Request) {
	usuario := middlewares.Usuario(r.Context())
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":      "rota privada acessada com sucesso",
		"papel_usuario": usuario.Papel,
		"go_version":    runtime.Version(),
		"momento_utc":   time.Now().UTC().Format(time.RFC3339),
		"request_id":    middlewares.ObterRequestID(r.Context()),
	})
}
