-- MODELO DE MIGRACAO SEGURA
-- Copie este arquivo para `banco/migracoes/006_nome_da_mudanca.sql`
-- e ajuste para o caso real.

BEGIN;

-- 1) Mudanca aditiva (segura)
-- Exemplo: adiciona coluna sem quebrar consumo atual.
-- ALTER TABLE usuarios
--   ADD COLUMN telefone TEXT;

-- 2) Backfill opcional (controlado)
-- Exemplo: preenche dado novo com base em dados existentes.
-- UPDATE usuarios
-- SET telefone = ''
-- WHERE telefone IS NULL;

-- 3) Constraint/indice depois do backfill
-- Exemplo: cria indice para performance em leitura.
-- CREATE INDEX IF NOT EXISTS idx_usuarios_telefone
--   ON usuarios(telefone);

COMMIT;

-- Notas:
-- - Evite DROP de coluna/tabela na mesma migracao da adicao.
-- - Para mudancas destrutivas, faca em fase posterior.
-- - Se falhar, a transacao inteira faz rollback automaticamente.
