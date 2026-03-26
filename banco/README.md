# Migracoes do Banco

## Estrutura
- `migracoes/`: arquivos SQL versionados em ordem numerica.
- `cmd/aplicar-migracoes`: aplicador de migracoes com controle em `schema_migrations`.

## Como aplicar
Na raiz de `servidor-go`:

```bash
go run ./cmd/aplicar-migracoes
```

## Regras
- Sempre criar novos arquivos com prefixo numerico crescente (`006_*.sql`, `007_*.sql`).
- Nao alterar migracao ja aplicada em ambiente compartilhado.
- Se precisar corrigir, criar nova migracao incremental.

## Boas praticas
- Checklist de producao: `CHECKLIST_MIGRACAO_SEGURA.md`
- Modelo de arquivo inicial: `modelos/006_exemplo_migracao_segura.sql`
