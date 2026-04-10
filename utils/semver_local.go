package utils

import (
	"strings"

	"golang.org/x/mod/semver"
)

// VersaoSemverMenor retorna true se a < b no sentido semver (ex.: 1.0.0 < 1.0.1).
// Aceita versoes com ou sem prefixo "v". Se alguma for invalida, compara lexicograficamente apos normalizar.
func VersaoSemverMenor(a, b string) bool {
	cmp := CompararVersaoSemver(a, b)
	return cmp < 0
}

// CompararVersaoSemver retorna -1 se a < b, 0 se iguais, 1 se a > b.
func CompararVersaoSemver(a, b string) int {
	ca := canonicoSemver(a)
	cb := canonicoSemver(b)
	if semver.IsValid(ca) && semver.IsValid(cb) {
		return semver.Compare(ca, cb)
	}
	return strings.Compare(strings.TrimSpace(a), strings.TrimSpace(b))
}

func canonicoSemver(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "v0.0.0"
	}
	if !strings.HasPrefix(s, "v") {
		s = "v" + s
	}
	if semver.IsValid(s) {
		return s
	}
	return s
}
