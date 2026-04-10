package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
)

func RegistrarPublicas(mux *http.ServeMux, h *handlers.Handlers, mws ...middlewares.Middleware) {
	mux.Handle("/saude", middlewares.Encadear(http.HandlerFunc(h.Saude), mws...))
	mux.Handle("/v1/autenticacao/login", middlewares.Encadear(http.HandlerFunc(h.Login), mws...))
	mux.Handle("/v1/admin-geral/dev/criar", middlewares.Encadear(http.HandlerFunc(h.CriarAdministradorGeralDev), mws...))
	mux.Handle("/v1/admin-geral/dev/login", middlewares.Encadear(http.HandlerFunc(h.LoginAdministradorGeralDev), mws...))
	mux.Handle("/v1/gestor-rede/dev/login", middlewares.Encadear(http.HandlerFunc(h.LoginGestorRedeDev), mws...))
	mux.Handle("/v1/usuario-rede/dev/login", middlewares.Encadear(http.HandlerFunc(h.LoginUsuarioRedePainelDev), mws...))
	mux.Handle("/v1/app/versao", middlewares.Encadear(http.HandlerFunc(h.VerificarVersaoAppMobile), mws...))
}
