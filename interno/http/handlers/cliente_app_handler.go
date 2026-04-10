package handlers

import (
	"errors"
	"net/http"

	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqCadastroClienteApp struct {
	IDRede         string `json:"id_rede"`
	NomeCompleto   string `json:"nome_completo"`
	Email          string `json:"email"`
	Senha          string `json:"senha"`
	ConfirmarSenha string `json:"confirmar_senha"`
	Telefone       string `json:"telefone"`
}

// PublicCadastroClienteApp POST /v1/public/clientes/cadastro — cadastro de cliente no app (sem auth).
func (h *Handlers) PublicCadastroClienteApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCadastroClienteApp
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}

	token, sessao, err := h.usuarioRedeService.CadastrarClienteApp(servicos.CadastroClienteAppInput{
		IDRede:         req.IDRede,
		NomeCompleto:   req.NomeCompleto,
		Email:          req.Email,
		Senha:          req.Senha,
		ConfirmarSenha: req.ConfirmarSenha,
		Telefone:       req.Telefone,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrEmailUsuarioEquipeDuplicado):
			utils.ResponderErro(w, http.StatusConflict, "email ja cadastrado nesta rede")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel concluir o cadastro")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "cadastro realizado com sucesso",
		"token":    token,
		"sessao":   sessao,
	})
}
