package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type voucherCompraPostgres struct {
	db *sql.DB
}

func NovoVoucherCompraPostgres(db *sql.DB) VoucherCompraRepositorio {
	return &voucherCompraPostgres{db: db}
}

func (r *voucherCompraPostgres) CriarPendenteComPix(x *VoucherCompraRegistro) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	camp := nullUUIDString(x.CampanhaID)
	var mpID any
	if x.MpPaymentID != nil {
		mpID = *x.MpPaymentID
	}
	ref := ""
	if x.ReferenciaPagamento != nil {
		ref = *x.ReferenciaPagamento
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO voucher_compras (
  id, rede_id, usuario_id, campanha_id, valor_solicitado, desconto_aplicado, valor_final, litros, status,
  mp_payment_id, referencia_pagamento, expira_pagamento_em, criado_em, atualizado_em
) VALUES (
  $1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9::status_voucher_compra,
  $10, $11, $12, NOW(), NOW()
)
RETURNING id::text, criado_em, atualizado_em
`, x.ID, x.RedeID, x.UsuarioID, camp, x.ValorSolicitado, x.DescontoAplicado, x.ValorFinal, nullFloat64Ptr(x.Litros), x.Status,
		mpID, nullStringPtr(ref), x.ExpiraPagamento,
	).Scan(&x.ID, &x.CriadoEm, &x.AtualizadoEm)
}

func nullStringPtr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullFloat64Ptr(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func nullUUIDString(p *string) any {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil
	}
	return strings.TrimSpace(*p)
}

type scannerVcr interface {
	Scan(dest ...any) error
}

func scanVcr(s scannerVcr, x *VoucherCompraRegistro) error {
	var camp, ref, cod sql.NullString
	var mpID sql.NullInt64
	var litros sql.NullFloat64
	var exPag, exRes sql.NullTime
	err := s.Scan(
		&x.ID, &x.RedeID, &x.UsuarioID, &camp, &x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&mpID, &ref, &cod, &exPag, &exRes, &x.CriadoEm, &x.AtualizadoEm,
	)
	if err != nil {
		return err
	}
	if litros.Valid {
		v := litros.Float64
		x.Litros = &v
	}
	if camp.Valid && strings.TrimSpace(camp.String) != "" {
		v := camp.String
		x.CampanhaID = &v
	}
	if mpID.Valid {
		v := mpID.Int64
		x.MpPaymentID = &v
	}
	if ref.Valid {
		v := ref.String
		x.ReferenciaPagamento = &v
	}
	if cod.Valid {
		v := cod.String
		x.CodigoResgate = &v
	}
	if exPag.Valid {
		t := exPag.Time
		x.ExpiraPagamento = &t
	}
	if exRes.Valid {
		t := exRes.Time
		x.ExpiraResgate = &t
	}
	return nil
}

func (r *voucherCompraPostgres) BuscarPorID(id, usuarioID, redeID string) (*VoucherCompraRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  id::text, rede_id::text, usuario_id::text, campanha_id::text,
  valor_solicitado, desconto_aplicado, valor_final, litros::float8, status::text,
  mp_payment_id, referencia_pagamento, codigo_resgate, expira_pagamento_em, expira_resgate_em, criado_em, atualizado_em
FROM voucher_compras
WHERE id = $1::uuid AND usuario_id = $2::uuid AND rede_id = $3::uuid`
	var x VoucherCompraRegistro
	err := scanVcr(r.db.QueryRowContext(ctx, q, id, usuarioID, redeID), &x)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVoucherCompraNaoEncontrado
		}
		return nil, err
	}
	return &x, nil
}

func (r *voucherCompraPostgres) ListarDoUsuario(redeID, usuarioID string, limite int) ([]*VoucherCompraRegistro, error) {
	if limite < 1 || limite > 200 {
		limite = 50
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, `
SELECT
  id::text, rede_id::text, usuario_id::text, campanha_id::text,
  valor_solicitado, desconto_aplicado, valor_final, litros::float8, status::text,
  mp_payment_id, referencia_pagamento, codigo_resgate, expira_pagamento_em, expira_resgate_em, criado_em, atualizado_em
FROM voucher_compras
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid
ORDER BY criado_em DESC
LIMIT $3`, redeID, usuarioID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*VoucherCompraRegistro
	for rows.Next() {
		var x VoucherCompraRegistro
		if err := scanVcr(rows, &x); err != nil {
			return nil, err
		}
		out = append(out, &x)
	}
	return out, rows.Err()
}

func (r *voucherCompraPostgres) ContarUsosCampanhaUsuario(campanhaID, usuarioID, redeID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM voucher_compras
WHERE campanha_id = $1::uuid AND usuario_id = $2::uuid AND rede_id = $3::uuid
  AND status IN ('ATIVO', 'USADO')
`, campanhaID, usuarioID, redeID).Scan(&n)
	return n, err
}

func (r *voucherCompraPostgres) ListarUsosAprovadosPorCampanha(redeID, usuarioID string) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, `
SELECT campanha_id::text, COUNT(*)::int
FROM voucher_compras
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid
  AND campanha_id IS NOT NULL
  AND status IN ('ATIVO', 'USADO')
GROUP BY campanha_id
`, redeID, usuarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		if strings.TrimSpace(id) != "" {
			out[id] = n
		}
	}
	return out, rows.Err()
}

func (r *voucherCompraPostgres) BuscarPorIDRede(id, redeID string) (*VoucherCompraRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  id::text, rede_id::text, usuario_id::text, campanha_id::text,
  valor_solicitado, desconto_aplicado, valor_final, litros::float8, status::text,
  mp_payment_id, referencia_pagamento, codigo_resgate, expira_pagamento_em, expira_resgate_em, criado_em, atualizado_em
FROM voucher_compras
WHERE id = $1::uuid AND rede_id = $2::uuid`
	var x VoucherCompraRegistro
	err := scanVcr(r.db.QueryRowContext(ctx, q, id, redeID), &x)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVoucherCompraNaoEncontrado
		}
		return nil, err
	}
	return &x, nil
}

func (r *voucherCompraPostgres) AtivarPagamentoAprovado(id, redeID, codigo string, expiraResgate time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := r.db.ExecContext(ctx, `
UPDATE voucher_compras SET
  status = 'ATIVO',
  codigo_resgate = $3,
  expira_resgate_em = $4,
  atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $2::uuid
  AND status = 'AGUARDANDO_PAGAMENTO'
`, id, redeID, strings.TrimSpace(codigo), expiraResgate)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("nenhuma linha ativada; status ou id invalido")
	}
	return nil
}
