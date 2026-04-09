package handlers

import "gaspass-servidor/interno/servicos"

type Handlers struct {
	autenticador        servicos.Autenticador
	adminService        servicos.ServicoAdministradorGeral
	gestorService       servicos.ServicoGestorRede
	redeService         servicos.ServicoRede
	usuarioRedeService  servicos.ServicoUsuarioRede
	postoService        servicos.ServicoPosto
	campanhaService     servicos.ServicoCampanha
}

func Novos(
	autenticador servicos.Autenticador,
	adminService servicos.ServicoAdministradorGeral,
	gestorService servicos.ServicoGestorRede,
	redeService servicos.ServicoRede,
	usuarioRedeService servicos.ServicoUsuarioRede,
	postoService servicos.ServicoPosto,
	campanhaService servicos.ServicoCampanha,
) *Handlers {
	return &Handlers{
		autenticador:       autenticador,
		adminService:       adminService,
		gestorService:      gestorService,
		redeService:        redeService,
		usuarioRedeService: usuarioRedeService,
		postoService:       postoService,
		campanhaService:    campanhaService,
	}
}
