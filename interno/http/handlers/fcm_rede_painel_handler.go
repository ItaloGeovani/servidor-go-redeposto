package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/notificacoes"
	"gaspass-servidor/utils"
)

// PostFcmTesteRedePainel POST /v1/.../push/fcm/rede/teste — envia teste a todos os app clientes (tokens) da rede.
// Gestor da rede e gerente de posto: JWT com rede; titulo e corpo opcionais.
func (h *Handlers) PostFcmTesteRedePainel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "nao autenticado")
		return
	}
	if u.Papel != modelos.PapelGestorRede && u.Papel != modelos.PapelGerentePosto {
		utils.ResponderErro(w, http.StatusForbidden, "acesso negado")
		return
	}
	idRede := strings.TrimSpace(u.IDRede)
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "usuario sem rede vinculada")
		return
	}

	var body struct {
		Titulo string `json:"titulo"`
		Corpo  string `json:"corpo"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	titulo := strings.TrimSpace(body.Titulo)
	corpo := strings.TrimSpace(body.Corpo)

	cred := strings.TrimSpace(h.cfg.FcmCaminhoContaServico)
	if cred == "" {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "push nao configurado no servidor (defina FCM_SA)")
		return
	}
	tokens, err := h.usuarioRedeService.ListarTokensFCMClientesRede(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar tokens da rede")
		return
	}
	if len(tokens) == 0 {
		utils.ResponderErro(w, http.StatusBadRequest, "nenhum token FCM de clientes nesta rede. E preciso abrir o app (cliente) e permitir notificacoes.")
		return
	}
	xctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	ok, fal, err := notificacoes.EnviarTesteRede(xctx, cred, tokens, idRede, titulo, corpo)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"enviados":       ok,
		"falhas":         fal,
		"tokens_tentado": len(tokens),
		"id_rede":        idRede,
	})
}
