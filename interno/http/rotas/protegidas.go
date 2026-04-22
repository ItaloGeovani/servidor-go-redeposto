package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/servicos"
)

func RegistrarProtegidas(muxPrincipal *http.ServeMux, h *handlers.Handlers, aut servicos.Autenticador, mws ...middlewares.Middleware) {
	muxProtegida := http.NewServeMux()
	muxProtegida.Handle("/v1/eu/perfil", http.HandlerFunc(h.PerfilLogado))
	muxProtegida.Handle("/v1/eu/conta", http.HandlerFunc(h.ExcluirContaClienteApp))
	muxProtegida.Handle("/v1/eu/pagamentos/mercadopago/pix", http.HandlerFunc(h.PostClienteMercadoPagoPix))
	muxProtegida.Handle("/v1/eu/vouchers/calcular", http.HandlerFunc(h.PostVoucherCompraCalcular))
	muxProtegida.Handle("/v1/eu/vouchers/pagar", http.HandlerFunc(h.PostVoucherCompraPagar))
	muxProtegida.Handle("/v1/eu/vouchers", http.HandlerFunc(h.GetVoucherCompras))
	muxProtegida.Handle("/v1/eu/vouchers/detalhe", http.HandlerFunc(h.GetVoucherCompraDetalhe))

	chain := append([]middlewares.Middleware{}, mws...)
	chain = append(chain, middlewares.ExigirAutenticacao(aut))
	muxPrincipal.Handle("/v1/eu/", middlewares.Encadear(muxProtegida, chain...))
}
