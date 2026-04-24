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

// PostRegistrarTokenFCM POST /v1/eu/push/fcm — regista token Firebase Cloud Messaging (app cliente logado).
func (h *Handlers) PostRegistrarTokenFCM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	if u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas contas de cliente do app")
		return
	}

	var body struct {
		Token      string `json:"token"`
		Plataforma string `json:"plataforma"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	body.Token = strings.TrimSpace(body.Token)
	body.Plataforma = strings.TrimSpace(body.Plataforma)
	if body.Token == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "token obrigatorio")
		return
	}

	if err := h.usuarioRedeService.RegistrarTokenFCM(u.IDUsuario, body.Token, body.Plataforma); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalido") {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel guardar o token")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PostFcmTeste POST /v1/eu/push/fcm/teste — envia uma notificacao de teste aos tokens FCM do utilizador (app cliente).
func (h *Handlers) PostFcmTeste(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	if u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas contas de cliente do app")
		return
	}
	cred := strings.TrimSpace(h.cfg.FcmCaminhoContaServico)
	if cred == "" {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "push nao configurado no servidor (FCM_SERVICE_ACCOUNT_PATH)")
		return
	}
	tokens, err := h.usuarioRedeService.ListarTokensFCM(u.IDUsuario)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel listar tokens")
		return
	}
	if len(tokens) == 0 {
		utils.ResponderErro(w, http.StatusBadRequest, "nenhum token registado. Abra o app e permita notificacoes, ou aguarde a sincronizacao.")
		return
	}
	xctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	ok, fal, err := notificacoes.EnviarTeste(xctx, cred, tokens)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"enviados":       ok,
		"falhas":         fal,
		"tokens_tentado": len(tokens),
	})
}
