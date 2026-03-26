package utils

import "time"

func ParseDataISO(valor string) (time.Time, error) {
	return time.Parse("2006-01-02", valor)
}

func ProximosVencimentosMensais(primeiro time.Time, quantidade int) []string {
	if quantidade <= 0 {
		return nil
	}

	lista := make([]string, 0, quantidade)
	base := primeiro
	for i := 0; i < quantidade; i++ {
		lista = append(lista, base.Format("2006-01-02"))
		base = base.AddDate(0, 1, 0)
	}
	return lista
}
