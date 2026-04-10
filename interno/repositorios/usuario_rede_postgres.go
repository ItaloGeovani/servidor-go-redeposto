package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"github.com/jackc/pgx/v5/pgconn"
)

type usuarioRedePostgres struct {
	db *sql.DB
}

func NovoUsuarioRedePostgres(db *sql.DB) *usuarioRedePostgres {
	return &usuarioRedePostgres{db: db}
}

var papeisRedePermitidos = map[string]struct{}{
	"gestor_rede":   {},
	"gerente_posto": {},
	"frentista":     {},
	"cliente":       {},
}

// ErrEmailUsuarioEquipeDuplicado quando ja existe usuario com o mesmo email na rede.
var ErrEmailUsuarioEquipeDuplicado = errors.New("email ja cadastrado nesta rede")

// ErrPostoNaoPertenceARede quando id_posto nao e da rede informada.
var ErrPostoNaoPertenceARede = errors.New("posto nao pertence a esta rede")

// ErrUsuarioEquipeNaoEncontrado quando o usuario nao existe na rede ou nao e equipe de posto.
var ErrUsuarioEquipeNaoEncontrado = errors.New("usuario da equipe nao encontrado nesta rede")

// ErrUsuarioPainelLoginNaoEncontrado quando nao ha gerente, frentista ou cliente com o email informado.
var ErrUsuarioPainelLoginNaoEncontrado = errors.New("usuario nao encontrado para login no painel")

// SanitizarPapeisFiltro remove valores invalidos; vazio significa "todos os papeis de rede" (exceto super_admin).
func SanitizarPapeisFiltro(papeis []string) []string {
	var out []string
	for _, p := range papeis {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := papeisRedePermitidos[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

func montarFiltroPapelIN(papeis []string, primeiroArg int) (condicao string, args []any) {
	if len(papeis) == 0 {
		return "", nil
	}
	ph := make([]string, len(papeis))
	args = make([]any, len(papeis))
	for i, p := range papeis {
		ph[i] = fmt.Sprintf("$%d", primeiroArg+i)
		args[i] = p
	}
	condicao = " AND u.papel::text IN (" + strings.Join(ph, ",") + ")"
	return condicao, args
}

// ListarPorRedeIDPaginado lista usuarios da rede (exceto super_admin), com filtro opcional de papeis, id_posto e paginacao.
func (r *usuarioRedePostgres) ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{strings.TrimSpace(idRede)}
	where := "WHERE u.rede_id = $1::uuid AND u.papel <> 'super_admin'"
	next := 2

	if posto := strings.TrimSpace(idPostoFiltro); posto != "" {
		where += fmt.Sprintf(" AND u.posto_id = $%d::uuid", next)
		args = append(args, posto)
		next++
	}

	filtroPapel, argsPapel := montarFiltroPapelIN(papeisFiltro, next)
	args = append(args, argsPapel...)
	whereSQL := "FROM usuarios u " + where + filtroPapel

	argsCount := append([]any{}, args...)
	queryCount := "SELECT COUNT(*) " + whereSQL
	var total int
	if err := r.db.QueryRowContext(ctx, queryCount, argsCount...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// args ja inclui argsPapel; nao duplicar (evita LIMIT/OFFSET com placeholders errados).
	argsLista := append([]any{}, args...)
	n := len(argsLista)
	lim := fmt.Sprintf("$%d", n+1)
	off := fmt.Sprintf("$%d", n+2)
	argsLista = append(argsLista, limite, offset)

	queryLista := `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  u.email,
  COALESCE(u.telefone, ''),
  u.ativo
` + whereSQL + `
ORDER BY u.papel::text, u.nome_completo
LIMIT ` + lim + ` OFFSET ` + off

	rows, err := r.db.QueryContext(ctx, queryLista, argsLista...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []*modelos.UsuarioVinculoRede
	for rows.Next() {
		var u modelos.UsuarioVinculoRede
		if err := rows.Scan(&u.ID, &u.IDRede, &u.IDPosto, &u.Papel, &u.Nome, &u.Email, &u.Telefone, &u.Ativo); err != nil {
			return nil, 0, err
		}
		lista = append(lista, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lista, total, nil
}

// PostoPertenceARede indica se o posto existe e pertence a rede.
func (r *usuarioRedePostgres) PostoPertenceARede(idPosto, idRede string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var um int
	err := r.db.QueryRowContext(ctx, `
SELECT 1 FROM postos WHERE id = $1::uuid AND rede_id = $2::uuid
LIMIT 1`, idPosto, idRede).Scan(&um)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CriarUsuarioEquipe insere gerente de posto ou frentista vinculado a rede e a um posto obrigatorio.
func (r *usuarioRedePostgres) CriarUsuarioEquipe(idRede, idPosto, papel, nome, email, senhaHash, telefone string) (*modelos.UsuarioVinculoRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
INSERT INTO usuarios (rede_id, posto_id, papel, nome_completo, email, senha_hash, ativo, telefone)
VALUES ($1::uuid, $2::uuid, $3::papel_usuario, $4, $5, $6, true, NULLIF($7, ''))
RETURNING
  id::text,
  rede_id::text,
  COALESCE(posto_id::text, ''),
  papel::text,
  nome_completo,
  email,
  COALESCE(telefone, ''),
  ativo`

	var u modelos.UsuarioVinculoRede
	err := r.db.QueryRowContext(
		ctx,
		query,
		idRede,
		strings.TrimSpace(idPosto),
		strings.TrimSpace(papel),
		strings.TrimSpace(nome),
		strings.TrimSpace(email),
		senhaHash,
		strings.TrimSpace(telefone),
	).Scan(&u.ID, &u.IDRede, &u.IDPosto, &u.Papel, &u.Nome, &u.Email, &u.Telefone, &u.Ativo)
	if err != nil {
		return nil, mapearErroUsuarioEquipePostgres(err)
	}
	return &u, nil
}

func mapearErroUsuarioEquipePostgres(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23505" {
		return ErrEmailUsuarioEquipeDuplicado
	}
	return err
}

// AtualizarUsuarioEquipe atualiza gerente de posto ou frentista; senhaHash vazio mantem a senha atual.
func (r *usuarioRedePostgres) AtualizarUsuarioEquipe(
	idRede, idUsuario string,
	nome, email, telefone string,
	ativo bool,
	papel, idPosto string,
	senhaHashOuVazio string,
) (*modelos.UsuarioVinculoRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	idRede = strings.TrimSpace(idRede)
	idUsuario = strings.TrimSpace(idUsuario)
	nome = strings.TrimSpace(nome)
	email = strings.TrimSpace(email)
	telefone = strings.TrimSpace(telefone)
	papel = strings.TrimSpace(papel)
	idPosto = strings.TrimSpace(idPosto)
	senhaHashOuVazio = strings.TrimSpace(senhaHashOuVazio)

	if idRede == "" || idUsuario == "" || nome == "" || email == "" || papel == "" || idPosto == "" {
		return nil, ErrDadosInvalidosUsuarioEquipe
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var papelAtual string
	err = tx.QueryRowContext(ctx, `
SELECT papel::text FROM usuarios
WHERE id = $1::uuid AND rede_id = $2::uuid
FOR UPDATE`, idUsuario, idRede).Scan(&papelAtual)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUsuarioEquipeNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if papelAtual != "gerente_posto" && papelAtual != "frentista" {
		return nil, ErrUsuarioEquipeNaoEncontrado
	}

	var duplicado int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM usuarios
WHERE rede_id = $1::uuid
  AND LOWER(TRIM(email)) = LOWER(TRIM($2))
  AND id <> $3::uuid`, idRede, email, idUsuario).Scan(&duplicado)
	if err != nil {
		return nil, err
	}
	if duplicado > 0 {
		return nil, ErrEmailUsuarioEquipeDuplicado
	}

	ok, err := r.postoPertenceARedeTx(ctx, tx, idPosto, idRede)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPostoNaoPertenceARede
	}

	const query = `
UPDATE usuarios SET
  nome_completo = $2,
  email = $3,
  telefone = NULLIF($4, ''),
  ativo = $5,
  papel = $6::papel_usuario,
  posto_id = $7::uuid,
  senha_hash = CASE WHEN $8 = '' THEN senha_hash ELSE $8 END,
  atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $9::uuid
RETURNING
  id::text,
  rede_id::text,
  COALESCE(posto_id::text, ''),
  papel::text,
  nome_completo,
  email,
  COALESCE(telefone, ''),
  ativo`

	var u modelos.UsuarioVinculoRede
	err = tx.QueryRowContext(
		ctx,
		query,
		idUsuario,
		nome,
		email,
		telefone,
		ativo,
		papel,
		idPosto,
		senhaHashOuVazio,
		idRede,
	).Scan(&u.ID, &u.IDRede, &u.IDPosto, &u.Papel, &u.Nome, &u.Email, &u.Telefone, &u.Ativo)
	if err != nil {
		return nil, mapearErroUsuarioEquipePostgres(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &u, nil
}

// ErrDadosInvalidosUsuarioEquipe parametros obrigatorios ausentes na atualizacao.
var ErrDadosInvalidosUsuarioEquipe = errors.New("dados invalidos para atualizar usuario da equipe")

// UsuarioPainelLogin linha de usuarios (gerente, frentista, cliente) para autenticacao.
type UsuarioPainelLogin struct {
	ID        string
	IDRede    string
	IDPosto   string
	Papel     string
	Nome      string
	SenhaHash string
	Ativo     bool
}

// BuscarPorEmailParaLoginPainel localiza um usuario pelo email (papeis de posto ou cliente).
// Se existir o mesmo email em varias redes, usa o registro mais recente ativo.
func (r *usuarioRedePostgres) BuscarPorEmailParaLoginPainel(email string) (*UsuarioPainelLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	email = strings.TrimSpace(email)
	if email == "" {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}

	const query = `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  u.senha_hash,
  u.ativo
FROM usuarios u
WHERE u.papel IN ('gerente_posto', 'frentista', 'cliente')
  AND LOWER(TRIM(u.email)) = LOWER(TRIM($1))
ORDER BY u.ativo DESC, u.criado_em DESC
LIMIT 1`

	var row UsuarioPainelLogin
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&row.ID,
		&row.IDRede,
		&row.IDPosto,
		&row.Papel,
		&row.Nome,
		&row.SenhaHash,
		&row.Ativo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *usuarioRedePostgres) postoPertenceARedeTx(ctx context.Context, tx *sql.Tx, idPosto, idRede string) (bool, error) {
	var um int
	err := tx.QueryRowContext(ctx, `
SELECT 1 FROM postos WHERE id = $1::uuid AND rede_id = $2::uuid
LIMIT 1`, idPosto, idRede).Scan(&um)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
