package handlers

import (
	"errors"
	"net/http"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

func (h *Handlers) PerfilLogado(w http.ResponseWriter, r *http.Request) {
	usuario := middlewares.Usuario(r.Context())
	if usuario == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}

	out := map[string]any{
		"id_usuario":    usuario.IDUsuario,
		"nome_completo": usuario.NomeCompleto,
		"id_rede":       usuario.IDRede,
		"papel":         usuario.Papel,
		"request_id":    middlewares.ObterRequestID(r.Context()),
	}
	email, cpf, err := h.usuarioRedeService.EmailECPFPorUsuarioRede(usuario.IDUsuario, usuario.IDRede)
	if err == nil {
		out["email"] = email
		out["cpf"] = cpf
	} else {
		out["email"] = ""
		out["cpf"] = ""
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

// ExcluirContaClienteApp DELETE /v1/eu/conta — encerra conta do cliente (app); anonimiza dados.
func (h *Handlers) ExcluirContaClienteApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	if u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "exclusao disponivel apenas para contas de cliente")
		return
	}
	err := h.usuarioRedeService.ExcluirContaClienteApp(u.IDUsuario, u.IDRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrContaClienteExclusaoNaoAplicada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel concluir a exclusao")
		}
		return
	}
	h.autenticador.RevogarToken(middlewares.BearerToken(r))
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "conta encerrada"})
}
