BEGIN;

ALTER TABLE rede_whatsapp_notificacoes
  ADD COLUMN IF NOT EXISTS notify_voucher_estorno BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN rede_whatsapp_notificacoes.notify_voucher_estorno IS
  'Avisa no grupo WhatsApp quando PIX/pagamento do voucher e estornado/devolvido';

COMMIT;
