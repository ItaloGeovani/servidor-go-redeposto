package servicos

import (
	"errors"
	"testing"
)

func TestStatusPixIndicaEstornoTotal(t *testing.T) {
	casos := []struct {
		st   string
		want bool
	}{
		{"refunded", true},
		{"REFUNDED", true},
		{"cancelled", true},
		{"canceled", true},
		{"charged_back", true},
		{"approved", false},
		{"pending", false},
		{"", false},
		{"  refunded  ", true},
	}
	for _, c := range casos {
		if got := statusPixIndicaEstornoTotal(c.st); got != c.want {
			t.Fatalf("statusPixIndicaEstornoTotal(%q)=%v want %v", c.st, got, c.want)
		}
	}
}

func TestErrConsultaPixInexistente(t *testing.T) {
	casos := []struct {
		msg  string
		want bool
	}{
		{`e.rede consulta status 404: {"returnCode":"78","returnMessage":"Transaction does not exist."}`, true},
		{`e.rede consulta status 500: timeout`, false},
		{`este posto nao aceita pagamento pix no momento`, false},
	}
	for _, c := range casos {
		err := errors.New(c.msg)
		if got := errConsultaPixInexistente(err); got != c.want {
			t.Fatalf("errConsultaPixInexistente(%q)=%v want %v", c.msg, got, c.want)
		}
	}
	if errConsultaPixInexistente(nil) {
		t.Fatal("nil deve ser false")
	}
}
