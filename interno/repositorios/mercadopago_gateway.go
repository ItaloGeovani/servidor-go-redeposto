package repositorios

import "errors"

var ErrMercadoPagoGatewayNaoConfigurado = errors.New("mercado pago nao configurado para esta rede")

// MercadoPagoGatewayCredenciais credenciais da aplicação Mercado Pago da rede (painel MP).
type MercadoPagoGatewayCredenciais struct {
	AccessToken   string
	WebhookSecret string
}

// MercadoPagoGatewayRepositorio persiste MP_ACCESS_TOKEN e MP_WEBHOOK_SECRET por rede.
type MercadoPagoGatewayRepositorio interface {
	BuscarPorRedeID(idRede string) (*MercadoPagoGatewayCredenciais, error)
	Upsert(idRede, accessToken, webhookSecret string) error
}
