package utils

import (
	"strconv"
	"strings"
	"unicode"
)

// SomenteDigitosCPF remove tudo que não for dígito.
func SomenteDigitosCPF(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ValidarCPF verifica formato e dígitos verificadores (Brasil).
func ValidarCPF(s string) bool {
	cpf := SomenteDigitosCPF(s)
	if len(cpf) != 11 {
		return false
	}
	if cpf == strings.Repeat(string(cpf[0]), 11) {
		return false
	}

	sum := 0
	for i := 0; i < 9; i++ {
		n, _ := strconv.Atoi(string(cpf[i]))
		sum += n * (10 - i)
	}
	r := sum % 11
	d1 := 0
	if r >= 2 {
		d1 = 11 - r
	}
	n9, _ := strconv.Atoi(string(cpf[9]))
	if n9 != d1 {
		return false
	}

	sum = 0
	for i := 0; i < 10; i++ {
		n, _ := strconv.Atoi(string(cpf[i]))
		sum += n * (11 - i)
	}
	r = sum % 11
	d2 := 0
	if r >= 2 {
		d2 = 11 - r
	}
	n10, _ := strconv.Atoi(string(cpf[10]))
	return n10 == d2
}
