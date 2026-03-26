# Checklist de Migracao Segura (Producao)

## Antes de criar a migracao
- [ ] Confirmar objetivo exato da mudanca (schema, dados, performance).
- [ ] Verificar impacto em API, aplicacao e relatorios.
- [ ] Definir estrategia aditiva (preferir `ADD` antes de `DROP`).
- [ ] Garantir backup recente e testado.

## Durante o desenvolvimento
- [ ] Criar novo arquivo numerado em `banco/migracoes` (`006_*.sql`, `007_*.sql`).
- [ ] Nao alterar migracoes ja aplicadas.
- [ ] Evitar comandos destrutivos diretos (`TRUNCATE`, `DROP COLUMN`, `DROP TABLE`) sem plano de transicao.
- [ ] Se houver mudanca de tipo/coluna critica, usar abordagem em fases:
  - adicionar nova coluna
  - preencher (backfill)
  - trocar leitura/escrita no app
  - remover legado em migracao posterior

## Validacao tecnica
- [ ] Rodar migracoes em ambiente local/homologacao com copia de dados reais (mascarados).
- [ ] Medir tempo de execucao e risco de lock.
- [ ] Validar indices e plano de consulta para endpoints criticos.
- [ ] Conferir compatibilidade de rollback.

## Janela de deploy
- [ ] Definir janela de manutencao (se necessario).
- [ ] Comunicar time e stakeholders.
- [ ] Aplicar migracoes com log e monitoramento ativo.
- [ ] Verificar rapidamente:
  - conectividade da aplicacao
  - erros de escrita/leitura
  - latencia dos endpoints principais

## Pos-migracao
- [ ] Confirmar estado esperado no banco (`schema_migrations` + tabelas alteradas).
- [ ] Validar dados criticos (contagens, totais, integridade referencial).
- [ ] Registrar resultado da execucao (data/hora, tempo, responsavel).
- [ ] Preparar plano de melhoria para proxima migracao.
