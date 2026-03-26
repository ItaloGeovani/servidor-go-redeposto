# Repositorios

Este diretorio deve conter interfaces e implementacoes de acesso a dados.

Exemplo de organizacao:
- `usuario_repositorio.go`: interface
- `usuario_repositorio_postgres.go`: implementacao PostgreSQL

Mantendo essa separacao, os servicos ficam testaveis sem depender direto de SQL.
