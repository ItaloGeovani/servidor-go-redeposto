package handlers

import (
	"gaspass-servidor/interno/config"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
)

type Handlers struct {
	autenticador       servicos.Autenticador
	adminService       servicos.ServicoAdministradorGeral
	gestorService      servicos.ServicoGestorRede
	redeService        servicos.ServicoRede
	usuarioRedeService servicos.ServicoUsuarioRede
	postoService       servicos.ServicoPosto
	campanhaService    servicos.ServicoCampanha
	premioService      servicos.ServicoPremio
	auditoriaRepo      repositorios.AuditoriaRepositorio
	estatisticasRepo   repositorios.EstatisticasPlataformaRepositorio
	appMobileRepo      repositorios.AppMobileConfigRepositorio
	cfg                config.Config
}

func Novos(
	autenticador servicos.Autenticador,
	adminService servicos.ServicoAdministradorGeral,
	gestorService servicos.ServicoGestorRede,
	redeService servicos.ServicoRede,
	usuarioRedeService servicos.ServicoUsuarioRede,
	postoService servicos.ServicoPosto,
	campanhaService servicos.ServicoCampanha,
	premioService servicos.ServicoPremio,
	auditoriaRepo repositorios.AuditoriaRepositorio,
	estatisticasRepo repositorios.EstatisticasPlataformaRepositorio,
	appMobileRepo repositorios.AppMobileConfigRepositorio,
	cfg config.Config,
) *Handlers {
	return &Handlers{
		autenticador:       autenticador,
		adminService:       adminService,
		gestorService:      gestorService,
		redeService:        redeService,
		usuarioRedeService: usuarioRedeService,
		postoService:       postoService,
		campanhaService:    campanhaService,
		premioService:      premioService,
		auditoriaRepo:      auditoriaRepo,
		estatisticasRepo:   estatisticasRepo,
		appMobileRepo:      appMobileRepo,
		cfg:                cfg,
	}
}
