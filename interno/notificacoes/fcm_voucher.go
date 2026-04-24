package notificacoes

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var (
	fcmClientMu  sync.Mutex
	fcmClient    *messaging.Client
	credPathUso  string
)

func fcmMensageria(ctx context.Context, cred string) (*messaging.Client, error) {
	if cred == "" {
		return nil, nil
	}
	fcmClientMu.Lock()
	defer fcmClientMu.Unlock()
	if fcmClient != nil && credPathUso == cred {
		return fcmClient, nil
	}
	b, err := os.ReadFile(cred)
	if err != nil {
		return nil, err
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON(b))
	if err != nil {
		return nil, err
	}
	c, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	fcmClient = c
	credPathUso = cred
	return c, nil
}

// EnviarVoucherAprovado push quando o pagamento do voucher no Mercado Pago é aprovado.
// [cred] é o caminho do JSON da conta de serviço Firebase (env FCM_SERVICE_ACCOUNT_PATH).
func EnviarVoucherAprovado(ctx context.Context, cred string, tokens []string, idCompra, codigo, valorReais string) {
	if cred == "" || len(tokens) == 0 {
		return
	}
	c, err := fcmMensageria(ctx, cred)
	if err != nil {
		log.Printf("fcm: abrir credenciais: %v", err)
		return
	}
	if c == nil {
		return
	}
	for i := 0; i < len(tokens); i += 500 {
		j := i + 500
		if j > len(tokens) {
			j = len(tokens)
		}
		batch := tokens[i:j]
		req := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title:    "Voucher aprovado",
				Body:     fmt.Sprintf("Seu pagamento de R$ %s foi confirmado. Abra o app para resgatar.", valorReais),
			},
			Data: map[string]string{
				"tipo":        "voucher_ativo",
				"id":          idCompra,
				"codigo":      codigo,
				"valor":       valorReais,
				"abrir_tela": "vouchers",
			},
		}
		br, err := c.SendEachForMulticast(ctx, req)
		if err != nil {
			log.Printf("fcm: SendEachForMulticast: %v", err)
			return
		}
		if br.FailureCount > 0 {
			log.Printf("fcm: lote: falhas=%d de %d (tokens invalidos ou desinstalacoes antigas)", br.FailureCount, len(batch))
		}
	}
}

// EnviarTeste notificacao simples (endpoint /v1/eu/push/fcm/teste) para validar FCM no dispositivo.
// Devolve o numero de envios com sucesso no lote; pode ser < len(tokens) se algum token for invalido.
func EnviarTeste(ctx context.Context, cred string, tokens []string) (int, int, error) {
	if cred == "" {
		return 0, 0, fmt.Errorf("credenciais fcm nao configuradas (FCM_SERVICE_ACCOUNT_PATH)")
	}
	if len(tokens) == 0 {
		return 0, 0, nil
	}
	c, err := fcmMensageria(ctx, cred)
	if err != nil {
		return 0, 0, err
	}
	if c == nil {
		return 0, 0, fmt.Errorf("cliente fcm nulo")
	}
	ok := 0
	fal := 0
	for i := 0; i < len(tokens); i += 500 {
		j := i + 500
		if j > len(tokens) {
			j = len(tokens)
		}
		batch := tokens[i:j]
		req := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: "Teste de notificacao",
				Body:  "Se recebeu isto, o push (FCM) esta a funcionar.",
			},
			Data: map[string]string{
				"tipo":         "fcm_teste",
				"abrir_tela":   "voucher",
			},
		}
		br, err := c.SendEachForMulticast(ctx, req)
		if err != nil {
			return ok, fal, err
		}
		ok += br.SuccessCount
		fal += br.FailureCount
	}
	return ok, fal, nil
}
