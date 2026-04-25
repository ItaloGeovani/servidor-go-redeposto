package servicos

import (
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

// IndiqueGanheUsuario leitura/escrita de codigo (subset do repo de usuarios).
type IndiqueGanheUsuario interface {
	DefinirCodigoIndicacao(idUsuario, idRede, codigo string) error
	ObterCodigoIndicacao(idUsuario, idRede string) (string, error)
	BuscarIdClientePorCodigoIndicacao(idRede, codigo string) (string, error)
}

// ServicoIndiqueGanhe regras de indicacao, bonus na carteira (moeda virtual).
type ServicoIndiqueGanhe struct {
	rede repositorios.RedeRepositorio
	ind  repositorios.IndiqueGanheRepositorio
	cart repositorios.CarteiraRepositorio
	usu  IndiqueGanheUsuario
}

func NovoServicoIndiqueGanhe(
	rede repositorios.RedeRepositorio,
	ind repositorios.IndiqueGanheRepositorio,
	cart repositorios.CarteiraRepositorio,
	usu IndiqueGanheUsuario,
) *ServicoIndiqueGanhe {
	return &ServicoIndiqueGanhe{rede: rede, ind: ind, cart: cart, usu: usu}
}

const charsetCodigo = "BCDFGHJKLMNPQRSTVWXYZ23456789"

// GerarCodigoIndicacao 8 caracteres.
func GerarCodigoIndicacao() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	s := make([]byte, 8)
	for i := 0; i < 8; i++ {
		s[i] = charsetCodigo[int(b[i])%len(charsetCodigo)]
	}
	return string(s)
}

// GaranteCodigoIndicacao gera e grava se ainda inexistente.
func (s *ServicoIndiqueGanhe) GaranteCodigoIndicacao(rede, usuarioID string) (string, error) {
	rede = strings.TrimSpace(rede)
	usuarioID = strings.TrimSpace(usuarioID)
	if rede == "" || usuarioID == "" {
		return "", ErrDadosInvalidos
	}
	exist, err := s.usu.ObterCodigoIndicacao(usuarioID, rede)
	if err != nil {
		return "", err
	}
	if exist != "" {
		return exist, nil
	}
	for i := 0; i < 32; i++ {
		c := GerarCodigoIndicacao()
		if err := s.usu.DefinirCodigoIndicacao(usuarioID, rede, c); err == nil {
			return c, nil
		}
	}
	return "", errors.New("falha ao gerar codigo de indicacao")
}

// AposNovoCadastro: codigo para o novo usuario, vinculo e premio (se regra e modulo).
func (s *ServicoIndiqueGanhe) AposNovoCadastro(rede, novoID, codIndicadorInformado string) {
	rede = strings.TrimSpace(rede)
	novoID = strings.TrimSpace(novoID)
	if rede == "" || novoID == "" {
		return
	}
	_, _ = s.GaranteCodigoIndicacao(rede, novoID)
	r, err := s.rede.BuscarPorID(rede)
	if err != nil || r == nil || !r.AppModuloIndiqueGanhe {
		return
	}
	cod := strings.TrimSpace(codIndicadorInformado)
	if cod == "" {
		return
	}
	codU := strings.ToUpper(cod)
	refID, err := s.usu.BuscarIdClientePorCodigoIndicacao(rede, codU)
	if err != nil || refID == "" || refID == novoID {
		return
	}
	exist, err := s.ind.BuscarIndicacaoPorIndicado(rede, novoID)
	if err != nil || exist != nil {
		return
	}
	indicID, err := s.ind.InsertIndicacao(rede, refID, novoID, codU)
	if err != nil {
		return
	}
	cfg, err := s.ind.BuscarConfig(rede)
	if err != nil || cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.Regra) != "CADASTRAR" {
		return
	}
	s.premiarCadastro(rede, r, indicID, refID, novoID, cfg)
}

func (s *ServicoIndiqueGanhe) premiarCadastro(
	rede string, r *modelos.Rede, indicID, refID, novoID string, cfg *repositorios.RedeIndiqueGanheConfig,
) {
	s.cred(rede, r, refID, cfg.MoedasPremioReferente, "ig_ref_cad", indicID)
	s.cred(rede, r, novoID, cfg.MoedasPremioIndicado, "ig_ind_cad", indicID)
	_ = s.ind.MarcarPremioCadastro(rede, indicID, true, true)
}

func (s *ServicoIndiqueGanhe) cred(rede string, r *modelos.Rede, usuarioID string, moedas float64, tipoRef, indicacaoID string) {
	if moedas <= 0 {
		return
	}
	cid, err := s.cart.ObterOuCriarCarteira(rede, usuarioID, strings.TrimSpace(r.MoedaVirtualNome), r.MoedaVirtualCotacao)
	if err != nil {
		return
	}
	_ = s.cart.CreditarBonus(rede, cid, moedas, tipoRef, indicacaoID)
}

// AposVoucherAprovado primeira compra com PIX aprovada (regra PRIMEIRA_COMPRA).
func (s *ServicoIndiqueGanhe) AposVoucherAprovado(rede, usuarioID, _compraID string) {
	rede = strings.TrimSpace(rede)
	usuarioID = strings.TrimSpace(usuarioID)
	if rede == "" || usuarioID == "" {
		return
	}
	red, err := s.rede.BuscarPorID(rede)
	if err != nil || red == nil || !red.AppModuloIndiqueGanhe {
		return
	}
	cfg, err := s.ind.BuscarConfig(rede)
	if err != nil || cfg == nil || strings.TrimSpace(cfg.Regra) != "PRIMEIRA_COMPRA_VOUCHER" {
		return
	}
	ind, err := s.ind.BuscarIndicacaoPorIndicado(rede, usuarioID)
	if err != nil || ind == nil {
		return
	}
	if ind.PremiadoCompraRef && ind.PremiadoCompraInd {
		return
	}
	n, err := s.ind.ContarVouchersAprovadosUsuario(rede, usuarioID)
	if err != nil || n < 1 {
		return
	}
	s.cred(rede, red, ind.ReferenteUsuarioID, cfg.MoedasPremioReferente, "ig_ref_compra", ind.ID)
	s.cred(rede, red, ind.IndicadoUsuarioID, cfg.MoedasPremioIndicado, "ig_ind_compra", ind.ID)
	_ = s.ind.MarcarPremioCompra(rede, ind.ID, true, true)
}

// BuscarConfigIndique delegacao.
func (s *ServicoIndiqueGanhe) BuscarConfigIndique(rede string) (*repositorios.RedeIndiqueGanheConfig, error) {
	return s.ind.BuscarConfig(rede)
}

// SalvarConfigIndique regra: CADASTRAR | PRIMEIRA_COMPRA_VOUCHER; moedas >= 0.
func (s *ServicoIndiqueGanhe) SalvarConfigIndique(rede, regra string, mRef, mInd float64) error {
	r := strings.TrimSpace(regra)
	if r != "CADASTRAR" && r != "PRIMEIRA_COMPRA_VOUCHER" {
		return ErrDadosInvalidos
	}
	if mRef < 0 || mInd < 0 {
		return ErrDadosInvalidos
	}
	if err := s.ind.SalvarConfig(rede, r, mRef, mInd); err != nil {
		return err
	}
	return nil
}

// MeuCodigoEU garante e devolve o codigo do usuario (app autenticado).
func (s *ServicoIndiqueGanhe) MeuCodigoEU(rede, usuarioID string) (string, error) {
	return s.GaranteCodigoIndicacao(rede, usuarioID)
}

// mascararNomeIndicado ex.: "Maria Silva Costa" -> "Maria C."
func mascararNomeIndicado(nome string) string {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "Indicado"
	}
	parts := strings.Fields(nome)
	if len(parts) == 1 {
		return parts[0]
	}
	first := parts[0]
	last := []rune(parts[len(parts)-1])
	if len(last) == 0 {
		return first
	}
	return first + " " + string(last[0]) + "."
}

func premioReferenteAprovado(regra string, premCadRef, premCompraRef bool) bool {
	if strings.TrimSpace(regra) == "CADASTRAR" {
		return premCadRef
	}
	return premCompraRef
}

func textoPendenciaPeloRegra(regra string) string {
	if strings.TrimSpace(regra) == "CADASTRAR" {
		return "Ainda processando o bônus de cadastro. Se demorar, fale com o suporte da rede."
	}
	return "A pessoa indicada ainda não fez a primeira compra de voucher paga e aprovada no app. Assim que isso acontecer, seu bônus em moedas da rede é creditado automaticamente."
}

// TelaIndiqueGanheEU payload completo para a tela do app (código, regra, lista).
func (s *ServicoIndiqueGanhe) TelaIndiqueGanheEU(rede, usuarioID string, r *modelos.Rede) (map[string]any, error) {
	cod, err := s.GaranteCodigoIndicacao(rede, usuarioID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.ind.BuscarConfig(rede)
	if err != nil || cfg == nil {
		cfg = &repositorios.RedeIndiqueGanheConfig{Regra: "PRIMEIRA_COMPRA_VOUCHER"}
	}
	linhas, err := s.ind.ListarIndicacoesDoReferente(rede, usuarioID)
	if err != nil {
		return nil, err
	}
	regra := strings.TrimSpace(cfg.Regra)
	indicacoes := make([]map[string]any, 0, len(linhas))
	for _, li := range linhas {
		aprov := premioReferenteAprovado(regra, li.PremiadoCadastroRef, li.PremiadoCompraRef)
		item := map[string]any{
			"id":                        strings.TrimSpace(li.ID),
			"indicado_nome_mascarado":  mascararNomeIndicado(li.IndicadoNomeCompleto),
			"criado_em":                 li.CriadoEm.UTC().Format(time.RFC3339),
			"premio_referente_aprovado": aprov,
		}
		if !aprov {
			item["motivo_pendencia"] = textoPendenciaPeloRegra(regra)
		}
		indicacoes = append(indicacoes, item)
	}
	m := map[string]any{
		"codigo_indicacao":          cod,
		"regra_premio":              regra,
		"moedas_premio_referente":   cfg.MoedasPremioReferente,
		"moedas_premio_indicado":   cfg.MoedasPremioIndicado,
		"indicacoes":                indicacoes,
		"texto_regra_cadastrar":     "Quem se cadastrar com seu código ganha a recompensa assim que o cadastro for concluído (conforme pontos definidos pela rede).",
		"texto_regra_primeira_compra": "Quem se cadastrar com seu código só libera bônus para vocês após a primeira compra de voucher aprovada no app.",
	}
	if r != nil {
		m["rede_nome_fantasia"] = strings.TrimSpace(r.NomeFantasia)
		m["moeda_virtual_nome"] = strings.TrimSpace(r.MoedaVirtualNome)
		m["moeda_virtual_cotacao"] = r.MoedaVirtualCotacao
		m["app_modulo_indique_ganhe"] = r.AppModuloIndiqueGanhe
	}
	return m, nil
}
