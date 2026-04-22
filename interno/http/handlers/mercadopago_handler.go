package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// MercadoPagoWebhookPublico POST /v1/public/mercadopago/webhook/{rede_id}
// Cadastre no painel Mercado Pago: URL pública + id da rede no path.
func (h *Handlers) MercadoPagoWebhookPublico(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	const prefix = "/v1/public/mercadopago/webhook/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	idRede := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, prefix), "/"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "rede_id invalido")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "corpo invalido")
		return
	}
	_ = r.Body.Close()

	creds, err := h.mpGatewayRepo.BuscarPorRedeID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrMercadoPagoGatewayNaoConfigurado) {
			log.Printf("mercadopago webhook: rede %s sem credenciais (ignorado)", idRede)
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("mercadopago webhook: buscar creds rede=%s: %v", idRede, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if strings.TrimSpace(creds.WebhookSecret) == "" {
		log.Printf("mercadopago webhook: rede %s sem mp_webhook_secret", idRede)
		w.WriteHeader(http.StatusOK)
		return
	}

	var actionWrap struct {
		Action string `json:"action"`
	}
	_ = json.Unmarshal(body, &actionWrap)
	if actionWrap.Action != "" && actionWrap.Action != "payment.updated" {
		w.WriteHeader(http.StatusOK)
		return
	}

	dataID, err := servicos.ExtrairDataIDDoWebhookMercadoPago(body)
	if err != nil {
		log.Printf("mercadopago webhook: extrair data.id: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	xSig := r.Header.Get("x-signature")
	xReq := r.Header.Get("x-request-id")
	if !servicos.ValidarAssinaturaWebhookMercadoPago(creds.WebhookSecret, body, xSig, xReq, dataID) {
		log.Printf("mercadopago webhook: assinatura invalida rede=%s", idRede)
		utils.ResponderErro(w, http.StatusForbidden, "assinatura invalida")
		return
	}

	paymentID, err := strconv.Atoi(dataID)
	if err != nil {
		log.Printf("mercadopago webhook: data.id nao numerico: %q", dataID)
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	pay, err := servicos.ConsultarPagamentoMercadoPago(ctx, creds.AccessToken, paymentID)
	if err != nil {
		log.Printf("mercadopago webhook: consultar payment %d: %v", paymentID, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if strings.TrimSpace(pay.Status) != "approved" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if h.voucherCompraSvc != nil {
		h.voucherCompraSvc.ProcessarPagamentoAprovadoWebhook(idRede, pay)
	}
	servicos.LogPagamentoAprovadoWebhook(idRede, paymentID, pay.ExternalReference, pay.Status)
	w.WriteHeader(http.StatusOK)
}

// MercadoPagoGatewayGestor GET/PUT /v1/gestor-rede/dev/mercadopago-gateway
func (h *Handlers) MercadoPagoGatewayGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getMercadoPagoGatewayGestor(w, r)
	case http.MethodPut:
		h.putMercadoPagoGatewayGestor(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getMercadoPagoGatewayGestor(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	creds, err := h.mpGatewayRepo.BuscarPorRedeID(idRede)
	out := map[string]any{
		"webhook_url": h.urlWebhookMercadoPago(idRede),
	}
	if err != nil {
		out["mp_access_token_configurado"] = false
		out["mp_webhook_secret_configurado"] = false
		utils.ResponderJSON(w, http.StatusOK, out)
		return
	}
	configurado := strings.TrimSpace(creds.AccessToken) != ""
	secretOk := strings.TrimSpace(creds.WebhookSecret) != ""
	out["mp_access_token_configurado"] = configurado
	out["mp_webhook_secret_configurado"] = secretOk
	if configurado {
		out["mp_access_token_mascarado"] = mascararSegredoMercadoPago(creds.AccessToken)
	}
	if secretOk {
		out["mp_webhook_secret_mascarado"] = mascararSegredoMercadoPago(creds.WebhookSecret)
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

func (h *Handlers) putMercadoPagoGatewayGestor(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var body struct {
		AccessToken   string `json:"mp_access_token"`
		WebhookSecret string `json:"mp_webhook_secret"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	if strings.TrimSpace(body.AccessToken) == "" || strings.TrimSpace(body.WebhookSecret) == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "mp_access_token e mp_webhook_secret sao obrigatorios")
		return
	}
	if err := h.mpGatewayRepo.Upsert(idRede, body.AccessToken, body.WebhookSecret); err != nil {
		log.Printf("mercadopago upsert rede=%s: %v", idRede, err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar credenciais")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"webhook_url": h.urlWebhookMercadoPago(idRede),
	})
}

// PostClienteMercadoPagoPix POST /v1/eu/pagamentos/mercadopago/pix — cliente autenticado; valor validado no servidor.
func (h *Handlers) PostClienteMercadoPagoPix(w http.ResponseWriter, r *http.Request) {
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
		utils.ResponderErro(w, http.StatusForbidden, "disponivel apenas para cliente")
		return
	}

	var body struct {
		Valor             float64 `json:"valor"`
		Descricao         string  `json:"descricao"`
		PayerEmail        string  `json:"payer_email"`
		DocTipo           string  `json:"doc_tipo"`
		DocNumero         string  `json:"doc_numero"`
		ExternalReference string  `json:"external_reference"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	if body.Valor < 1.0 {
		utils.ResponderErro(w, http.StatusBadRequest, "valor minimo R$ 1,00")
		return
	}
	body.PayerEmail = strings.TrimSpace(body.PayerEmail)
	if body.PayerEmail == "" || !strings.Contains(body.PayerEmail, "@") {
		utils.ResponderErro(w, http.StatusBadRequest, "payer_email invalido")
		return
	}
	body.DocTipo = strings.TrimSpace(body.DocTipo)
	body.DocNumero = strings.TrimSpace(body.DocNumero)
	if body.DocTipo == "" || body.DocNumero == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "doc_tipo e doc_numero obrigatorios (ex: CPF e 11 digitos)")
		return
	}

	creds, err := h.mpGatewayRepo.BuscarPorRedeID(u.IDRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrMercadoPagoGatewayNaoConfigurado) {
			utils.ResponderErro(w, http.StatusBadRequest, "rede sem mercado pago configurado")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar gateway")
		return
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "rede sem mp_access_token")
		return
	}

	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
	if base == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "servidor sem PUBLIC_BASE_URL: necessario para notification_url do PIX")
		return
	}
	notif := base + "/v1/public/mercadopago/webhook/" + strings.TrimSpace(u.IDRede)

	desc := strings.TrimSpace(body.Descricao)
	if desc == "" {
		desc = "Pagamento AP GasPass"
	}
	ext := strings.TrimSpace(body.ExternalReference)
	if ext == "" {
		ext = "rede:" + u.IDRede + ";usuario:" + u.IDUsuario
	}

	ctx := r.Context()
	res, err := servicos.CriarCobrancaPixMercadoPago(ctx, creds.AccessToken, servicos.CriarCobrancaPixMercadoPagoInput{
		Valor:             body.Valor,
		Descricao:         desc,
		PayerEmail:        body.PayerEmail,
		DocTipo:           body.DocTipo,
		DocNumero:         body.DocNumero,
		ExternalReference: ext,
		NotificationURL:   notif,
	})
	if err != nil {
		log.Printf("mercadopago criar pix rede=%s: %v", u.IDRede, err)
		utils.ResponderErro(w, http.StatusBadGateway, "falha ao criar cobranca no mercado pago")
		return
	}

	qr := res.PointOfInteraction.TransactionData.QRCode
	qrB64 := res.PointOfInteraction.TransactionData.QRCodeBase64
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"payment_id":      res.ID,
		"status":          res.Status,
		"status_detail":   res.StatusDetail,
		"qr_code":         qr,
		"qr_code_base64":  qrB64,
		"amount":          body.Valor,
		"external_reference": res.ExternalReference,
	})
}

func (h *Handlers) urlWebhookMercadoPago(idRede string) string {
	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/v1/public/mercadopago/webhook/" + strings.TrimSpace(idRede)
}

func mascararSegredoMercadoPago(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 12 {
		return "****"
	}
	return s[:6] + "…" + s[len(s)-4:]
}
