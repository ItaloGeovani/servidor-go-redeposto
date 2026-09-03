BEGIN;

ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS reconciliado_em TIMESTAMPTZ NULL;

COMMENT ON COLUMN voucher_compras.reconciliado_em IS
  'Ultima vez que o worker reconsultou o status PIX no provedor (vouchers ATIVO)';

CREATE INDEX IF NOT EXISTS idx_voucher_compras_pix_ativo_reconcilia
  ON voucher_compras (reconciliado_em NULLS FIRST, atualizado_em ASC)
  WHERE status = 'ATIVO'
    AND COALESCE(NULLIF(TRIM(meio_pagamento), ''), 'PIX') = 'PIX';

COMMIT;
