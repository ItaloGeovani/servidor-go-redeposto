package modelos

import "testing"

func TestFlagParaTipoVoucherEstorno(t *testing.T) {
	c := &RedeWhatsAppNotificacoes{
		Habilitado:           true,
		NotifyVoucherEstorno: true,
	}
	if !c.FlagParaTipo(EventoVoucherEstorno) {
		t.Fatal("esperado true com notify_voucher_estorno")
	}
	c.NotifyVoucherEstorno = false
	if c.FlagParaTipo(EventoVoucherEstorno) {
		t.Fatal("esperado false")
	}
	c.Habilitado = false
	c.NotifyVoucherEstorno = true
	if c.FlagParaTipo(EventoVoucherEstorno) {
		t.Fatal("desabilitado deve bloquear")
	}
}
