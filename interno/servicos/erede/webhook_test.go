package erede

import "testing"

func TestParseWebhookPago(t *testing.T) {
	body := []byte(`{"companyNumber":"1","events":["PV.UPDATE_TRANSACTION_PIX"],"data":{"id":"TID123"}}`)
	r, err := ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if r.TID != "TID123" || r.Tipo != WebhookEventoPago {
		t.Fatalf("got %+v", r)
	}
}

func TestParseWebhookDevolucao(t *testing.T) {
	body := []byte(`{"events":["PV.REFUND_PIX"],"data":{"id":"TID999"}}`)
	r, err := ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if r.Tipo != WebhookEventoDevolucao || r.TID != "TID999" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseWebhookDevolucaoPrioridadeSobrePago(t *testing.T) {
	body := []byte(`{"events":["PV.UPDATE_TRANSACTION_PIX","PV.REFUND_PIX"],"data":{"id":"T1"}}`)
	r, err := ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if r.Tipo != WebhookEventoDevolucao {
		t.Fatalf("tipo=%s want devolucao", r.Tipo)
	}
}

func TestParseWebhookIgnorado(t *testing.T) {
	body := []byte(`{"events":["OUTRO"],"data":{"id":"T1"}}`)
	r, err := ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if r.Tipo != WebhookEventoIgnorado {
		t.Fatalf("tipo=%s", r.Tipo)
	}
}

func TestTransacaoAprovadaPixNaoQuandoCanceled(t *testing.T) {
	tx := &transactionResponse{
		Authorization: &authorizationBlock{ReturnCode: "00", Status: "Canceled"},
	}
	if TransacaoAprovadaPix(tx) {
		t.Fatal("Canceled com returnCode 00 nao deve ser aprovado")
	}
	if !TransacaoCanceladaPix(tx) {
		t.Fatal("esperado cancelada")
	}
}

func TestTransacaoAprovadaPixApproved(t *testing.T) {
	tx := &transactionResponse{
		Authorization: &authorizationBlock{ReturnCode: "00", Status: "Approved"},
	}
	if !TransacaoAprovadaPix(tx) {
		t.Fatal("Approved deveria ser aprovado")
	}
	if TransacaoCanceladaPix(tx) {
		t.Fatal("nao deveria ser cancelada")
	}
}
