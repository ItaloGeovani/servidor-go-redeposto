package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
)

func RegistrarPrivadas(muxPrincipal *http.ServeMux, h *handlers.Handlers, aut servicos.Autenticador, mws ...middlewares.Middleware) {
	muxPrivada := http.NewServeMux()
	muxPrivada.Handle("/v1/admin/diagnostico", http.HandlerFunc(h.DiagnosticoAdmin))
	muxPrivada.Handle("/v1/admin/dashboard/resumo", http.HandlerFunc(h.ResumoDashboardAdmin))
	muxPrivada.Handle("/v1/admin/administradores-gerais/dev/editar", http.HandlerFunc(h.EditarAdministradorGeralDev))
	muxPrivada.Handle("/v1/admin/gestores-rede/dev/listar", http.HandlerFunc(h.ListarGestoresRedeDev))
	muxPrivada.Handle("/v1/admin/gestores-rede/dev/criar", http.HandlerFunc(h.CriarGestorRedeComPlanoDev))
	muxPrivada.Handle("/v1/admin/gestores-rede/dev/editar", http.HandlerFunc(h.EditarGestorRedeComPlanoDev))
	muxPrivada.Handle("/v1/admin/redes/dev/listar", http.HandlerFunc(h.ListarRedesDev))
	muxPrivada.Handle("/v1/admin/redes/dev/criar", http.HandlerFunc(h.CriarRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/editar", http.HandlerFunc(h.EditarRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/ativar", http.HandlerFunc(h.AtivarRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/desativar", http.HandlerFunc(h.DesativarRedeDev))

	chain := append([]middlewares.Middleware{}, mws...)
	chain = append(chain, middlewares.ExigirAutenticacao(aut), middlewares.ExigirPapel(modelos.PapelSuperAdmin))
	muxPrincipal.Handle("/v1/admin/", middlewares.Encadear(muxPrivada, chain...))
}
