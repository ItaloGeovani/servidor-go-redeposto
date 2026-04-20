package servicos

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

var (
	ErrMercadoPagoAssinaturaInvalida = errors.New("assinatura do webhook invalida")
	ErrMercadoPagoCredenciais        = errors.New("credenciais mercado pago ausentes ou invalidas")
)

// ValidarAssinaturaWebhookMercadoPago valida o header x-signature (ts + v1 HMAC-SHA256 hex) conforme o painel MP.
func ValidarAssinaturaWebhookMercadoPago(secret string, body []byte, xSignature, xRequestID, dataID string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" || xSignature == "" {
		return false
	}
	ts, v1, ok := parseMercadoPagoSignatureHeader(xSignature)
	if !ok || ts == "" || v1 == "" {
		return false
	}
	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", dataID, strings.TrimSpace(xRequestID), ts)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(manifest))
	expected := mac.Sum(nil)
	got, err := hex.DecodeString(strings.ToLower(v1))
	if err != nil || len(got) != len(expected) {
		return false
	}
	return hmac.Equal(expected, got)
}

func parseMercadoPagoSignatureHeader(xSignature string) (ts, v1 string, ok bool) {
	// Formato típico: "ts=1700000000,v1=abcdef..." ou com espaços após vírgula
	for _, part := range strings.Split(xSignature, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "ts=") {
			ts = strings.TrimPrefix(part, "ts=")
		}
		if strings.HasPrefix(part, "v1=") {
			v1 = strings.TrimPrefix(part, "v1=")
		}
	}
	return ts, v1, ts != "" && v1 != ""
}

// ExtrairDataIDDoWebhookMercadoPago obtém data.id do JSON (payment id no MP).
func ExtrairDataIDDoWebhookMercadoPago(body []byte) (string, error) {
	var wrap struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return "", err
	}
	if len(wrap.Data) == 0 {
		return "", errors.New("campo data ausente")
	}
	var d struct {
		ID interface{} `json:"id"`
	}
	if err := json.Unmarshal(wrap.Data, &d); err != nil {
		return "", err
	}
	return normalizarIDMercadoPago(d.ID)
}

func normalizarIDMercadoPago(v interface{}) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", errors.New("data.id nulo")
	case float64:
		return strconv.FormatInt(int64(x), 10), nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return "", errors.New("data.id vazio")
		}
		return s, nil
	default:
		s := strings.TrimSpace(fmt.Sprint(x))
		if s == "" {
			return "", errors.New("data.id invalido")
		}
		return s, nil
	}
}

// ConsultarPagamentoMercadoPago GET /v1/payments/{id}.
func ConsultarPagamentoMercadoPago(ctx context.Context, accessToken string, paymentID int) (*payment.Response, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, ErrMercadoPagoCredenciais
	}
	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, err
	}
	client := payment.NewClient(cfg)
	return client.Get(ctx, paymentID)
}

// CriarCobrancaPixMercadoPago cria cobrança PIX (valor sempre validado no servidor).
type CriarCobrancaPixMercadoPagoInput struct {
	Valor               float64
	Descricao           string
	PayerEmail          string
	DocTipo             string
	DocNumero           string
	ExternalReference string
	NotificationURL   string
}

// CriarCobrancaPixMercadoPago retorna o payment criado (QR em PointOfInteraction).
func CriarCobrancaPixMercadoPago(ctx context.Context, accessToken string, in CriarCobrancaPixMercadoPagoInput) (*payment.Response, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, ErrMercadoPagoCredenciais
	}
	if in.Valor < 1.0 {
		return nil, errors.New("valor minimo para PIX: R$ 1,00")
	}
	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, err
	}
	client := payment.NewClient(cfg)

	req := payment.Request{
		TransactionAmount: in.Valor,
		Description:       strings.TrimSpace(in.Descricao),
		PaymentMethodID:   "pix",
		Payer: &payment.PayerRequest{
			Email: strings.TrimSpace(in.PayerEmail),
			Identification: &payment.IdentificationRequest{
				Type:   strings.TrimSpace(in.DocTipo),
				Number: strings.TrimSpace(in.DocNumero),
			},
		},
	}
	if strings.TrimSpace(in.ExternalReference) != "" {
		req.ExternalReference = strings.TrimSpace(in.ExternalReference)
	}
	if strings.TrimSpace(in.NotificationURL) != "" {
		req.NotificationURL = strings.TrimSpace(in.NotificationURL)
	}

	// SDK atual não expõe idempotency no Request; em produção pode-se envolver o HTTP client.
	res, err := client.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// LogPagamentoAprovadoWebhook placeholder até existir tabela de compras/vouchers.
func LogPagamentoAprovadoWebhook(idRede string, paymentID int, extRef, status string) {
	log.Printf("mercadopago webhook: pagamento aprovado rede=%s payment_id=%d external_reference=%q status=%s",
		idRede, paymentID, extRef, status)
}
