package handlers

import "gaspass-servidor/interno/servicos"

type Handlers struct {
	autenticador  servicos.Autenticador
	adminService  servicos.ServicoAdministradorGeral
	gestorService servicos.ServicoGestorRede
	redeService   servicos.ServicoRede
}

func Novos(
	autenticador servicos.Autenticador,
	adminService servicos.ServicoAdministradorGeral,
	gestorService servicos.ServicoGestorRede,
	redeService servicos.ServicoRede,
) *Handlers {
	return &Handlers{
		autenticador:  autenticador,
		adminService:  adminService,
		gestorService: gestorService,
		redeService:   redeService,
	}
}
