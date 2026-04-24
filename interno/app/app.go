package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// verificarCaminhoContaFCM confirma no arranque que o JSON existe e e ficheiro (log no terminal).
// Se FCM_SERVICE_ACCOUNT_PATH estiver vazio, nao e erro; push fica desativado.
// Tenta varios candidatos: caminho do .env, CWD, pasta do executavel (hospedagem com CWD diferente da app).
// Em sucesso, actualiza [cfg].FcmCaminhoContaServico para o caminho absoluto que abriu.
func verificarCaminhoContaFCM(cfg *config.Config) error {
	p := strings.TrimSpace(cfg.FcmCaminhoContaServico)
	if p == "" {
		log.Print("fcm: FCM_SERVICE_ACCOUNT_PATH nao definido; envio de push desativado")
		return nil
	}
	cands := candidatosCaminhoFcm(p)
	for _, c := range cands {
		info, err := os.Stat(c)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		abs, errA := filepath.Abs(c)
		if errA != nil {
			abs = c
		}
		cfg.FcmCaminhoContaServico = abs
		log.Printf("fcm: credenciais OK: %q (%d bytes)", abs, info.Size())
		return nil
	}
	b := filepath.Base(p)
	return fmt.Errorf("fcm: ficheiro da conta de servico nao encontrado. Tentou %d caminhos; coloque %q na pasta da API no servidor (ao lado do executavel) ou defina FCM_BASE_DIR. Ultimo FCM_SERVICE_ACCOUNT_PATH=%q", len(cands), b, p)
}

func candidatosCaminhoFcm(p string) []string {
	p = strings.TrimSpace(p)
	if p == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if a, e := filepath.Abs(s); e == nil {
			s = a
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(p)
	if !filepath.IsAbs(p) {
		if w, e := os.Getwd(); e == nil {
			add(filepath.Join(w, p))
		}
	}
	nome := filepath.Base(p)
	// 3) mesmo nome de ficheiro na pasta do executavel (typico: json na raiz do deploy, bin noutro sitio)
	if ex, e := os.Executable(); e == nil {
		ex = filepath.Clean(ex)
		dirEx := filepath.Dir(ex)
		if !filepath.IsAbs(p) {
			add(filepath.Join(dirEx, p))
		}
		add(filepath.Join(dirEx, nome))
	}
	// 4) so o nome do ficheiro no CWD
	if w, e := os.Getwd(); e == nil {
		add(filepath.Join(w, nome))
	}
	return out
}

func Nova() (*Aplicacao, error) {
	cfg := config.Carregar()
	if err := verificarCaminhoContaFCM(&cfg); err != nil {
		return nil, err
	}
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
	repoCombustivelRede := repositorios.NovoCombustivelRedePostgres(banco)
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
	svcCampanha := servicos.NovoServicoCampanha(repoCampanha, repoRede, repoCombustivelRede)
	svcVoucherCompra := servicos.NovoServicoVoucherCompra(repoVoucherCompra, svcCampanha, repoMercadoPagoGateway, repoRede, repoCombustivelRede, repoUsuarioRede, cfg)
	svcCombustivelRede := servicos.NovoServicoCombustivelRede(repoCombustivelRede, repoRede)
	svcPremio := servicos.NovoServicoPremio(repoPremio, repoRede)
	if err := bootstrapAdminPadrao(cfg, svcAdmin); err != nil {
		banco.Close()
		return nil, err
	}

	h := handlers.Novos(autenticador, svcAdmin, svcGestor, svcRede, svcUsuarioRede, svcPosto, svcCampanha, svcPremio, repoAuditoria, estatisticasPlataforma, repoAppMobile, repoAppCards, repoMercadoPagoGateway, svcVoucherCompra, svcCombustivelRede, cfg)

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
