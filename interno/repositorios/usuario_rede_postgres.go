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
