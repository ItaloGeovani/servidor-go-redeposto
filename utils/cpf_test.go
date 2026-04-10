package utils

import "testing"

func TestValidarCPF(t *testing.T) {
	if !ValidarCPF("529.982.247-25") {
		t.Fatal("CPF valido deve passar")
	}
	if ValidarCPF("111.111.111-11") {
		t.Fatal("sequencia invalida")
	}
	if ValidarCPF("123") {
		t.Fatal("curto demais")
	}
}
