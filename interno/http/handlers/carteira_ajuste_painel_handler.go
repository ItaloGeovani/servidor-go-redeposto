package handlers

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"
)

const tipoRefAjusteGestorDebito = "AJUSTE_GESTOR_DEBITO"

type ajustarSaldoCarteiraReq struct {
	IDRede         string   `json:"id_rede"`
	IDUsuario      string   `json:"id_usuario"`
	NovoSaldoToken *float64 `json:"novo_saldo_token"`
}

// PostAjustarSaldoCarteiraClientePainel reduz saldo da moeda (nunca aumenta): gestor/admin.
// POST body: { id_usuario, novo_saldo_token, id_rede? }
func (h *Handlers) PostAjustarSaldoCarteiraClientePainel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.carteiraRepo == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "carteira indisponivel")
		return
	}

	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}

	var req ajustarSaldoCarteiraReq
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	idUsuario := strings.TrimSpace(req.IDUsuario)
	if idUsuario == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_usuario")
		return
	}
	if req.NovoSaldoToken == nil {
		utils.ResponderErro(w, http.StatusBadRequest, "informe novo_saldo_token")
		return
	}
	novo := *req.NovoSaldoToken
	if math.IsNaN(novo) || math.IsInf(novo, 0) || novo < 0 {
		utils.ResponderErro(w, http.StatusBadRequest, "novo_saldo_token deve ser >= 0")
		return
	}
	// Evita lixo de ponto flutuante / valores absurdos.
	if novo > 1e12 {
		utils.ResponderErro(w, http.StatusBadRequest, "novo_saldo_token invalido")
		return
	}

	idRede := ""
	switch u.Papel {
	case modelos.PapelGestorRede:
		var ok bool
		idRede, ok = h.idRedeDaSessao(w, r)
		if !ok {
			return
		}
	case modelos.PapelSuperAdmin:
		idRede = strings.TrimSpace(req.IDRede)
		if idRede == "" {
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
			return
		}
	default:
		utils.ResponderErro(w, http.StatusForbidden, "apenas gestor da rede ou administrador")
		return
	}

	saldoAtual, err := h.carteiraRepo.ObterSaldoToken(idRede, idUsuario)
	if err != nil {
		log.Printf("carteira ajuste: saldo rede=%s user=%s: %v", idRede, idUsuario, err)
		utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel ler o saldo")
		return
	}

	const eps = 1e-6
	if novo > saldoAtual+eps {
		utils.ResponderErro(w, http.StatusBadRequest, "nao e permitido aumentar o saldo; informe um valor menor ou igual ao atual")
		return
	}
	debito := saldoAtual - novo
	if debito < eps {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{
			"ok":                true,
			"alterado":          false,
			"saldo_anterior":    roundToken6(saldoAtual),
			"saldo_atual":       roundToken6(saldoAtual),
			"debitado":          0,
			"novo_saldo_token":  roundToken6(saldoAtual),
		})
		return
	}

	refID := uuid.NewString()
	if err := h.carteiraRepo.DebitarMoeda(idRede, idUsuario, debito, tipoRefAjusteGestorDebito, refID); err != nil {
		if errors.Is(err, repositorios.ErrSaldoInsuficiente) {
			utils.ResponderErro(w, http.StatusConflict, "saldo insuficiente (pode ter mudado); atualize a lista")
			return
		}
		log.Printf("carteira ajuste: debitar rede=%s user=%s: %v", idRede, idUsuario, err)
		utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel ajustar o saldo")
		return
	}

	saldoNovo, err := h.carteiraRepo.ObterSaldoToken(idRede, idUsuario)
	if err != nil {
		saldoNovo = novo
	}
	log.Printf(
		"carteira ajuste: gestor=%s rede=%s cliente=%s debito=%.6f saldo %.6f -> %.6f ref=%s",
		strings.TrimSpace(u.IDUsuario), idRede, idUsuario, debito, saldoAtual, saldoNovo, refID,
	)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"alterado":         true,
		"saldo_anterior":   roundToken6(saldoAtual),
		"saldo_atual":      roundToken6(saldoNovo),
		"debitado":         roundToken6(debito),
		"novo_saldo_token": roundToken6(saldoNovo),
		"referencia_id":    refID,
	})
}

func roundToken6(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}
