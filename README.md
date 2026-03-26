# Servidor Go

Estrutura inicial do backend focada em organizacao, baixo acoplamento e manutencao simples.

## Estrutura
- `cmd/api`: ponto de entrada da API HTTP.
- `cmd/aplicar-migracoes`: aplicador de migracoes no banco.
- `cmd/teste-conexao`: teste rapido de conexao PostgreSQL.
- `interno/config`: leitura de configuracoes.
- `interno/http/handlers`: handlers por contexto.
- `interno/http/middlewares`: middlewares compartilhados (auth, logger, recovery, request id).
- `interno/http/rotas`: registro de rotas `publicas`, `protegidas` e `privadas`.
- `interno/modelos`: structs de dominio.
- `interno/repositorios`: interfaces/implementacoes de acesso a dados.
- `interno/servicos`: regras de negocio e autenticacao.
- `utils`: utilitarios reusaveis centralizados.

## Padrao de Rotas
- Publicas: sem autenticacao.
- Protegidas: exigem token.
- Privadas: exigem token + papel `super_admin`.

## Rotas de Administrador Geral (desenvolvimento)
- `POST /v1/admin-geral/dev/criar`
- `POST /v1/admin-geral/dev/login`
- `PUT /v1/admin/administradores-gerais/dev/editar` (privada)

## Rotas de Gestor da Rede com Plano (desenvolvimento)
- `POST /v1/admin/gestores-rede/dev/criar` (privada)
- `PUT /v1/admin/gestores-rede/dev/editar` (privada)

No momento da criacao/edicao do gestor, o payload ja inclui:
- `valor_implantacao`
- `valor_mensalidade`
- `primeiro_vencimento` (`YYYY-MM-DD`)

## Executar API
Na raiz de `servidor-go`:

```bash
go run ./cmd/api
```

## Bootstrap de administrador
No `.env`, a aplicacao cria automaticamente um admin inicial em memoria:
- `ADMIN_EMAIL_PADRAO=admin@gaspass.local`
- `ADMIN_SENHA_PADRAO=123456`

Com esse login, o endpoint de login devolve um token de sessao para uso nas rotas protegidas e privadas.
