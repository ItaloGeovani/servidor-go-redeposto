package handlers

import (
	"net/http"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/utils"
)

func (h *Handlers) PerfilLogado(w http.ResponseWriter, r *http.Request) {
	usuario := middlewares.Usuario(r.Context())
	if usuario == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_usuario":    usuario.IDUsuario,
		"nome_completo": usuario.NomeCompleto,
		"id_rede":       usuario.IDRede,
		"papel":         usuario.Papel,
		"request_id":    middlewares.ObterRequestID(r.Context()),
	})
}
