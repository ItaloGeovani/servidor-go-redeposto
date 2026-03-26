package handlers

import (
	"net/http"

	"gaspass-servidor/utils"
)

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	// Endpoint inicial para desenvolvimento. A validacao real sera integrada
	// com usuarios no banco e emissao de JWT.
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "autenticacao de desenvolvimento habilitada",
		"tokens_teste": map[string]string{
			"super_admin": "dev-super-admin",
			"gestor_rede": "dev-gestor",
			"frentista":   "dev-frentista",
		},
		"admin_geral_dev": map[string]string{
			"criar":  "POST /v1/admin-geral/dev/criar",
			"login":  "POST /v1/admin-geral/dev/login",
			"editar": "PUT /v1/admin/administradores-gerais/dev/editar",
		},
		"gestor_rede_dev": map[string]string{
			"criar_com_plano":  "POST /v1/admin/gestores-rede/dev/criar",
			"editar_com_plano": "PUT /v1/admin/gestores-rede/dev/editar",
		},
		"redes_dev": map[string]string{
			"listar":    "GET /v1/admin/redes/dev/listar",
			"criar":     "POST /v1/admin/redes/dev/criar",
			"editar":    "PUT /v1/admin/redes/dev/editar",
			"ativar":    "PATCH /v1/admin/redes/dev/ativar",
			"desativar": "PATCH /v1/admin/redes/dev/desativar",
		},
		"uso": "envie Authorization: Bearer <token> nas rotas protegidas e privadas",
	})
}
