package servicos

import (
	"context"
	"log"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

const (
	defaultPixReconciliaIntervalo = 15 * time.Minute
	defaultPixReconciliaGrace     = 15 * time.Minute
	defaultPixReconciliaLote      = 40
	defaultPixReconciliaPausa     = 200 * time.Millisecond
)

// statusPixIndicaEstornoTotal status unificado de ConsultarPixVoucher / MP que cancela voucher.
func statusPixIndicaEstornoTotal(st string) bool {
	switch strings.ToLower(strings.TrimSpace(st)) {
	case "refunded", "cancelled", "canceled", "charged_back":
		return true
	default:
		return false
	}
}

// ReconciliaVouchersPixAtivos reconsulta o provedor para vouchers ATIVO (PIX) e cancela se estornados.
func (s *ServicoVoucherCompra) ReconciliaVouchersPixAtivos(ctx context.Context) (consultados, estornados, erros int) {
	return s.reconciliaVouchersPixAtivos(ctx, false)
}

func (s *ServicoVoucherCompra) reconciliaVouchersPixAtivos(ctx context.Context, forcar bool) (consultados, estornados, erros int) {
	if s == nil || s.repo == nil {
		return 0, 0, 0
	}
	log.Print("======= WORKER PIX RECONCILIA INICIADO ===========")
	defer log.Print("======= WORKER PIX RECONCILIA FINALIZADO ===========")

	cfg := s.cfgPixReconcilia()
	if nExp, errExp := s.repo.ExpirarAguardandoPagamentoVencidos(500); errExp != nil {
		log.Printf("voucher_pix reconcilia: expirar aguardando: %v", errExp)
	} else if nExp > 0 {
		log.Printf("voucher_pix reconcilia: expirados AGUARDANDO_PAGAMENTO vencidos=%d", nExp)
	}

	grace := cfg.grace
	if forcar {
		grace = time.Second
		log.Print("voucher_pix reconcilia: ciclo forçado pos-start (grace=1s) — loga RESPONSE e.Rede")
	}
	lista, err := s.repo.ListarAtivosPixParaReconcilia(cfg.lote, grace)
	if err != nil {
		log.Printf("voucher_pix reconcilia: listar: %v", err)
		return 0, 0, 1
	}
	log.Printf("voucher_pix reconcilia: elegiveis=%d (lote=%d grace=%s)", len(lista), cfg.lote, grace)
	agora := time.Now()
	for i, vc := range lista {
		if vc == nil {
			continue
		}
		if ctx.Err() != nil {
			log.Printf(
				"voucher_pix reconcilia: interrompido consultados=%d estornados=%d erros=%d",
				consultados, estornados, erros,
			)
			return consultados, estornados, erros
		}
		if i > 0 {
			select {
			case <-ctx.Done():
				log.Printf(
					"voucher_pix reconcilia: interrompido consultados=%d estornados=%d erros=%d",
					consultados, estornados, erros,
				)
				return consultados, estornados, erros
			case <-time.After(defaultPixReconciliaPausa):
			}
		}
		consultados++
		okEstorno, errC := s.reconciliaUmVoucherPixAtivo(ctx, vc, agora)
		if errC != nil {
			erros++
			log.Printf("voucher_pix reconcilia: compra=%s rede=%s: %v", vc.ID, vc.RedeID, errC)
			msg := strings.TrimSpace(errC.Error())
			if len(msg) > 220 {
				msg = msg[:220] + "…"
			}
			s.registrarEventoVoucher(
				modelos.EventoVoucherReconciliaErro,
				vc,
				"Worker: falha ao consultar provedor — "+msg,
			)
			// Evita monopolizar o lote enquanto o provedor falha.
			if errM := s.repo.MarcarReconciliadoPix(vc.ID, vc.RedeID, agora); errM != nil {
				log.Printf("voucher_pix reconcilia: marcar apos erro compra=%s: %v", vc.ID, errM)
			}
			continue
		}
		if okEstorno {
			estornados++
		}
	}
	log.Printf(
		"voucher_pix reconcilia: ciclo consultados=%d estornados=%d erros=%d",
		consultados, estornados, erros,
	)
	return consultados, estornados, erros
}

// errConsultaPixInexistente TID/pagamento não existe no provedor (ex.: e.Rede returnCode 78).
// Não é estorno e não vale alerta operacional repetido — só marca e segue.
func errConsultaPixInexistente(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "transaction does not exist") {
		return true
	}
	if strings.Contains(s, `"returncode":"78"`) || strings.Contains(s, "returncode\": \"78\"") {
		return true
	}
	return strings.Contains(s, "consulta status 404") && strings.Contains(s, "78")
}

func (s *ServicoVoucherCompra) reconciliaUmVoucherPixAtivo(
	ctx context.Context, vc *repositorios.VoucherCompraRegistro, agora time.Time,
) (estornou bool, err error) {
	idPosto := ""
	if vc.PostoCompraID != nil {
		idPosto = strings.TrimSpace(*vc.PostoCompraID)
	}
	gw, err := ResolverGatewayPagamentoConsulta(s.rede, s.mpGW, s.eredeGW, s.posto, s.cfg, vc.RedeID, idPosto)
	if err != nil {
		return false, err
	}
	provedor := strings.TrimSpace(vc.GatewayProvedor)
	if provedor == "" {
		provedor = gw.Provedor
	}
	tid := ""
	if vc.GatewayTID != nil {
		tid = strings.TrimSpace(*vc.GatewayTID)
	}
	pix, err := ConsultarPixVoucher(ctx, gw, provedor, tid, vc.MpPaymentID)
	if err != nil {
		if errConsultaPixInexistente(err) {
			log.Printf(
				"voucher_pix reconcilia: tid inexistente no provedor compra=%s tid=%s ambiente=%s — nao cancela: %v",
				vc.ID, tid, gw.ERedeAmbiente, err,
			)
			if errM := s.repo.MarcarReconciliadoPix(vc.ID, vc.RedeID, agora); errM != nil {
				log.Printf("voucher_pix reconcilia: marcar compra=%s: %v", vc.ID, errM)
			}
			return false, nil
		}
		return false, err
	}
	log.Printf(
		"voucher_pix reconcilia: compra=%s tid=%s provedor=%s ambiente=%s status_mapeado=%s status_gateway=%q",
		vc.ID, tid, provedor, gw.ERedeAmbiente, strings.TrimSpace(pix.Status), strings.TrimSpace(pix.GatewayStatusLabel),
	)
	st := strings.TrimSpace(pix.Status)
	if statusPixIndicaEstornoTotal(st) {
		s.ProcessarPagamentoEstornadoPorCompra(vc.RedeID, vc.ID, "reconcilia_"+strings.ToLower(st))
		return true, nil
	}
	// approved / pending / etc.: marca como checado (não cancela).
	if err := s.repo.MarcarReconciliadoPix(vc.ID, vc.RedeID, agora); err != nil {
		log.Printf("voucher_pix reconcilia: marcar compra=%s: %v", vc.ID, err)
	}
	return false, nil
}

type pixReconciliaCfg struct {
	desligado bool
	intervalo time.Duration
	grace     time.Duration
	lote      int
}

func (s *ServicoVoucherCompra) cfgPixReconcilia() pixReconciliaCfg {
	c := pixReconciliaCfg{
		intervalo: defaultPixReconciliaIntervalo,
		grace:     defaultPixReconciliaGrace,
		lote:      defaultPixReconciliaLote,
	}
	if s != nil {
		c.desligado = s.cfg.VoucherPixReconciliaDesligado
		if s.cfg.VoucherPixReconciliaIntervalo > 0 {
			c.intervalo = s.cfg.VoucherPixReconciliaIntervalo
		}
		if s.cfg.VoucherPixReconciliaGrace > 0 {
			c.grace = s.cfg.VoucherPixReconciliaGrace
		}
		if s.cfg.VoucherPixReconciliaLote > 0 {
			c.lote = s.cfg.VoucherPixReconciliaLote
		}
	}
	return c
}

// StartReconciliaPixWorker loop periódico até ctx cancelado.
func (s *ServicoVoucherCompra) StartReconciliaPixWorker(ctx context.Context) {
	if s == nil {
		return
	}
	cfg := s.cfgPixReconcilia()
	if cfg.desligado {
		log.Print("voucher_pix reconcilia: worker desligado (VOUCHER_PIX_RECONCILIA_DESLIGADO)")
		return
	}
	log.Printf(
		"voucher_pix reconcilia: worker ativo intervalo=%s grace=%s lote=%d",
		cfg.intervalo, cfg.grace, cfg.lote,
	)
	// Primeira passagem após 1 min (dar tempo ao HTTP subir / DB aquecer).
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		s.reconciliaVouchersPixAtivos(ctx, true)
	}
	ticker := time.NewTicker(cfg.intervalo)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Print("voucher_pix reconcilia: worker encerrado")
			return
		case <-ticker.C:
			s.ReconciliaVouchersPixAtivos(ctx)
		}
	}
}
