package repositorios

import (
	"context"
	"database/sql"
	"time"
)

// EstatisticasPlataformaRepositorio agregados globais para relatorios do administrador geral.
type EstatisticasPlataformaRepositorio interface {
	ResumoPlataforma() (map[string]any, error)
}

type estatisticasPlataformaPostgres struct {
	db *sql.DB
}

func NovoEstatisticasPlataformaPostgres(db *sql.DB) EstatisticasPlataformaRepositorio {
	return &estatisticasPlataformaPostgres{db: db}
}

func (r *estatisticasPlataformaPostgres) ResumoPlataforma() (map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const q = `
SELECT
  (SELECT COUNT(*)::int FROM redes) AS total_redes,
  (SELECT COUNT(*)::int FROM redes WHERE COALESCE(ativa, true)) AS redes_ativas,
  (SELECT COUNT(*)::int FROM redes WHERE NOT COALESCE(ativa, true)) AS redes_inativas,
  (SELECT COALESCE(SUM(valor_mensalidade), 0)::float8 FROM redes WHERE COALESCE(ativa, true)) AS receita_mensal_prevista,
  (SELECT COALESCE(SUM(valor_implantacao), 0)::float8 FROM redes) AS receita_implantacao_prevista,
  (SELECT COUNT(*)::int FROM usuarios WHERE papel = 'gestor_rede') AS total_gestores,
  (SELECT COUNT(*)::int FROM usuarios WHERE papel = 'gestor_rede' AND ativo = true) AS gestores_ativos,
  (SELECT COUNT(*)::int FROM postos) AS total_postos,
  (SELECT COUNT(*)::int FROM campanhas) AS total_campanhas,
  (SELECT COUNT(*)::int FROM campanhas WHERE status = 'ATIVA') AS campanhas_ativas,
  (SELECT COUNT(*)::int FROM campanhas WHERE status = 'RASCUNHO') AS campanhas_rascunho,
  (SELECT COUNT(*)::int FROM campanhas WHERE status = 'PAUSADA') AS campanhas_pausadas,
  (SELECT COUNT(*)::int FROM campanhas WHERE status = 'ARQUIVADA') AS campanhas_arquivadas,
  (SELECT COUNT(*)::int FROM premios) AS total_premios,
  (SELECT COUNT(*)::int FROM premios WHERE ativo = true) AS premios_ativos,
  (SELECT COUNT(*)::int FROM usuarios WHERE papel = 'cliente') AS usuarios_clientes,
  (SELECT COUNT(*)::int FROM usuarios WHERE papel IN ('gerente_posto', 'frentista')) AS usuarios_equipe_posto,
  (SELECT COUNT(*)::int FROM usuarios) AS usuarios_total,
  (SELECT COUNT(*)::int FROM logs_auditoria) AS total_logs_auditoria`

	var (
		totalRedes, redesAtivas, redesInativas int
		receitaMensal, receitaImpl             float64
		totalGestores, gestoresAtivos          int
		totalPostos, totalCampanhas            int
		cAtivas, cRasc, cPaus, cArq          int
		totalPremios, premiosAtivos            int
		usuCli, usuEquipe, usuTotal            int
		totalLogs                              int
	)

	err := r.db.QueryRowContext(ctx, q).Scan(
		&totalRedes, &redesAtivas, &redesInativas,
		&receitaMensal, &receitaImpl,
		&totalGestores, &gestoresAtivos,
		&totalPostos, &totalCampanhas,
		&cAtivas, &cRasc, &cPaus, &cArq,
		&totalPremios, &premiosAtivos,
		&usuCli, &usuEquipe, &usuTotal,
		&totalLogs,
	)
	if err != nil {
		return nil, err
	}

	gestoresInativos := totalGestores - gestoresAtivos
	if gestoresInativos < 0 {
		gestoresInativos = 0
	}

	return map[string]any{
		"total_redes":                   totalRedes,
		"redes_ativas":                  redesAtivas,
		"redes_inativas":                redesInativas,
		"receita_mensal_prevista":       receitaMensal,
		"receita_implantacao_prevista":  receitaImpl,
		"total_gestores":                totalGestores,
		"gestores_ativos":               gestoresAtivos,
		"gestores_inativos":             gestoresInativos,
		"total_postos":                  totalPostos,
		"total_campanhas":               totalCampanhas,
		"campanhas_ativas":              cAtivas,
		"campanhas_rascunho":            cRasc,
		"campanhas_pausadas":            cPaus,
		"campanhas_arquivadas":          cArq,
		"total_premios":                 totalPremios,
		"premios_ativos":                premiosAtivos,
		"usuarios_clientes":             usuCli,
		"usuarios_equipe_postos":        usuEquipe,
		"usuarios_total":                usuTotal,
		"total_logs_auditoria":          totalLogs,
	}, nil
}
