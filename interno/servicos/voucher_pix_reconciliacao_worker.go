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
	if s == nil || s.repo == nil {
		return 0, 0, 0
	}
	cfg := s.cfgPixReconcilia()
	lista, err := s.repo.ListarAtivosPixParaReconcilia(cfg.lote, cfg.grace)
	if err != nil {
		log.Printf("voucher_pix reconcilia: listar: %v", err)
		return 0, 0, 1
	}
	agora := time.Now()
	for i, vc := range lista {
		if vc == nil {
			continue
		}
		if ctx.Err() != nil {
			return consultados, estornados, erros
		}
		if i > 0 {
			select {
			case <-ctx.Done():
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
	if consultados > 0 || estornados > 0 || erros > 0 {
		log.Printf(
			"voucher_pix reconcilia: ciclo consultados=%d estornados=%d erros=%d",
			consultados, estornados, erros,
		)
	}
	return consultados, estornados, erros
}

func (s *ServicoVoucherCompra) reconciliaUmVoucherPixAtivo(
	ctx context.Context, vc *repositorios.VoucherCompraRegistro, agora time.Time,
) (estornou bool, err error) {
	idPosto := ""
	if vc.PostoCompraID != nil {
		idPosto = strings.TrimSpace(*vc.PostoCompraID)
	}
	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.posto, s.cfg, vc.RedeID, idPosto)
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
		return false, err
	}
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
		s.ReconciliaVouchersPixAtivos(ctx)
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
