package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"gaspass-servidor/interno/config"
	"gaspass-servidor/interno/http/estatico"
	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/http/rotas"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Aplicacao struct {
	cfg      config.Config
	servidor *http.Server
	banco    *sql.DB
}

func Nova() (*Aplicacao, error) {
	cfg := config.Carregar()
	banco, err := abrirBanco()
	if err != nil {
		return nil, err
	}

	autenticador := servicos.NovoAutenticadorToken(cfg.TokenPadraoAPI)
	repoAdmin := repositorios.NovoAdministradorGeralPostgres(banco)
	repoGestor := repositorios.NovoGestorRedePostgres(banco)
	repoRede := repositorios.NovoRedePostgres(banco)
	repoUsuarioRede := repositorios.NovoUsuarioRedePostgres(banco)
	repoPosto := repositorios.NovoPostoPostgres(banco)
	repoCampanha := repositorios.NovoCampanhaPostgres(banco)
	repoPremio := repositorios.NovoPremioPostgres(banco)
	repoAuditoria := repositorios.NovoAuditoriaPostgres(banco)
	estatisticasPlataforma := repositorios.NovoEstatisticasPlataformaPostgres(banco)
	repoAppMobile := repositorios.NovoAppMobileConfigPostgres(banco)
	repoAppCards := repositorios.NovoAppCardsRedePostgres(banco)
	repoMercadoPagoGateway := repositorios.NovoMercadoPagoGatewayPostgres(banco)
	repoVoucherCompra := repositorios.NovoVoucherCompraPostgres(banco)
	svcAdmin, err := servicos.NovoServicoAdministradorGeral(repoAdmin, autenticador)
	if err != nil {
		banco.Close()
		return nil, err
	}
	svcRede := servicos.NovoServicoRede(repoRede)
	svcGestor, err := servicos.NovoServicoGestorRede(repoGestor, repoRede, autenticador)
	if err != nil {
		banco.Close()
		return nil, err
	}
	svcUsuarioRede, err := servicos.NovoServicoUsuarioRede(repoUsuarioRede, repoRede, autenticador)
	if err != nil {
		banco.Close()
		return nil, err
	}
	svcPosto := servicos.NovoServicoPosto(repoPosto, repoRede)
	svcCampanha := servicos.NovoServicoCampanha(repoCampanha, repoRede)
	svcVoucherCompra := servicos.NovoServicoVoucherCompra(repoVoucherCompra, svcCampanha, repoMercadoPagoGateway, repoRede, cfg)
	svcPremio := servicos.NovoServicoPremio(repoPremio, repoRede)
	if err := bootstrapAdminPadrao(cfg, svcAdmin); err != nil {
		banco.Close()
		return nil, err
	}

	h := handlers.Novos(autenticador, svcAdmin, svcGestor, svcRede, svcUsuarioRede, svcPosto, svcCampanha, svcPremio, repoAuditoria, estatisticasPlataforma, repoAppMobile, repoAppCards, repoMercadoPagoGateway, svcVoucherCompra, cfg)

	muxPrincipal := http.NewServeMux()
	mwGlobal := []middlewares.Middleware{
		middlewares.CORS(cfg.CORSOrigemPermitida),
		middlewares.RequestID(),
		middlewares.RecuperacaoPanico(),
		middlewares.Logger(),
	}

	rotas.RegistrarPublicas(muxPrincipal, h, mwGlobal...)
	rotas.RegistrarProtegidas(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarPrivadas(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarGestorRedePainel(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarGerentePostoPainel(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarFrentistaPainel(muxPrincipal, h, autenticador, mwGlobal...)

	raizPainel := estatico.EncontrarRaizPainel(cfg.PastaPainelWeb)
	if raizPainel != "" {
		log.Printf("painel web estatico: %s (GET /)", raizPainel)
	} else {
		log.Printf("painel web: nenhuma pasta com index.html (assets/, PAINEL_WEB_ASSETS ou build:deploy); raiz mostra aviso HTML")
	}
	handlerPrincipal := estatico.ComSPAFallback(muxPrincipal, raizPainel)

	servidor := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.PortaHTTP),
		Handler:           handlerPrincipal,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Aplicacao{
		cfg:      cfg,
		servidor: servidor,
		banco:    banco,
	}, nil
}

// testr
func bootstrapAdminPadrao(cfg config.Config, svc servicos.ServicoAdministradorGeral) error {
	if !cfg.AdminBootstrapAtivado {
		return nil
	}

	_, err := svc.Criar(cfg.AdminNomePadrao, cfg.AdminEmailPadrao, cfg.AdminSenhaPadrao)
	if err == nil {
		log.Printf("admin geral padrao criado: email=%s", cfg.AdminEmailPadrao)
		return nil
	}
	if errors.Is(err, repositorios.ErrEmailJaCadastrado) {
		return nil
	}
	return err
}

func (a *Aplicacao) Executar(ctx context.Context) error {
	errCh := make(chan error, 1)

	log.Printf("servidor rodando em http://localhost%s (ambiente=%s)", a.servidor.Addr, a.cfg.Ambiente)

	go func() {
		if err := a.servidor.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.servidor.Shutdown(ctxShutdown); err != nil {
			return err
		}
		return a.banco.Close()
	case err := <-errCh:
		return err
	}
}

func abrirBanco() (*sql.DB, error) {
	dsn, err := utils.MontarDSNPostgres()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
