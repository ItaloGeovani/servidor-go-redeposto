package notificacoes

import (
	"strings"
	"testing"

	"gaspass-servidor/interno/modelos"
)

func TestRenderWhatsAppTemplateEstorno(t *testing.T) {
	txt := RenderWhatsAppTemplate(modelos.EventoVoucherEstorno, 0, WhatsAppTemplateDados{
		Cabecalho: "Posto Teste",
		Valor:     "10,00",
		Quem:      "Cliente",
		Meio:      "PIX",
		Status:    "CANCELADO",
		Codigo:    "ABCD1234",
		Extra:     "PIX ESTORNADO — voucher cancelado, nao honrar no posto",
	})
	if !strings.Contains(txt, "🔴") {
		t.Fatalf("faltou emoji vermelho de estorno: %s", txt)
	}
	if !strings.Contains(txt, "ESTORNADO") && !strings.Contains(txt, "DEVOLVIDO") {
		t.Fatalf("template sem estorno/devolvido: %s", txt)
	}
	if !strings.Contains(txt, "ABCD1234") {
		t.Fatalf("faltou codigo: %s", txt)
	}
	if !strings.Contains(txt, "nao honrar") {
		t.Fatalf("faltou extra: %s", txt)
	}
}

func TestRenderWhatsAppTemplateReconciliaErro(t *testing.T) {
	txt := RenderWhatsAppTemplate(modelos.EventoVoucherReconciliaErro, 0, WhatsAppTemplateDados{
		Cabecalho: "Posto Teste",
		Valor:     "25,00",
		Quem:      "Cliente",
		Meio:      "PIX",
		Status:    "ATIVO",
		Extra:     "Worker: falha ao consultar provedor — timeout",
	})
	if !strings.Contains(txt, "🟠") {
		t.Fatalf("faltou emoji laranja de erro: %s", txt)
	}
	if !strings.Contains(txt, "ERRO AO VERIFICAR PIX") {
		t.Fatalf("template sem titulo de erro: %s", txt)
	}
	if !strings.Contains(txt, "timeout") {
		t.Fatalf("faltou extra: %s", txt)
	}
}

func TestRenderWhatsAppEmojisPorTipo(t *testing.T) {
	cases := []struct {
		tipo  string
		emoji string
	}{
		{modelos.EventoVoucherGerado, "🟡"},
		{modelos.EventoVoucherPago, "🟢"},
		{modelos.EventoVoucherBaixa, "🔵"},
		{modelos.EventoVoucherEstorno, "🔴"},
		{modelos.EventoVoucherReconciliaErro, "🟠"},
	}
	for _, c := range cases {
		txt := RenderWhatsAppTemplate(c.tipo, 4, WhatsAppTemplateDados{
			Cabecalho: "Rede",
			Valor:     "1,00",
			Quem:      "X",
			Meio:      "PIX",
			Status:    "OK",
		})
		if !strings.Contains(txt, c.emoji) {
			t.Fatalf("tipo %s sem emoji %s: %s", c.tipo, c.emoji, txt)
		}
	}
}
