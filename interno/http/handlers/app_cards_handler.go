package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"
)

type reqSalvarAppCards struct {
	Cards []struct {
		Slot       int    `json:"slot"`
		Titulo     string `json:"titulo"`
		ImagemURL  string `json:"imagem_url"`
		LinkURL    string `json:"link_url"`
		Ativo      bool   `json:"ativo"`
	} `json:"cards"`
}

// AppCardsGestorRede GET lista / PUT salva cards do app para a rede da sessao (gestor ou gerente).
func (h *Handlers) AppCardsGestorRede(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listarAppCardsRede(w, r)
	case http.MethodPut, http.MethodPatch:
		h.salvarAppCardsRede(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) listarAppCardsRede(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	itens, err := h.appCardsRepo.ListarPorRedeID(idRede)
	if err != nil {
		log.Printf("listar app cards: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar cards")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"cards": h.enriquecerRespostaAppCards(itens),
	})
}

func (h *Handlers) salvarAppCardsRede(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqSalvarAppCards
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	vistos := make(map[int]bool)
	var lista []*modelos.AppCardRede
	for _, c := range req.Cards {
		if vistos[c.Slot] {
			utils.ResponderErro(w, http.StatusBadRequest, "slot duplicado no payload")
			return
		}
		vistos[c.Slot] = true
		if c.Slot < 0 || c.Slot > 3 {
			utils.ResponderErro(w, http.StatusBadRequest, "slot deve ser 0 (destaque) ou 1 a 3 (promocoes)")
			return
		}
		img := strings.TrimSpace(c.ImagemURL)
		if img != "" && !strings.HasPrefix(strings.ToLower(img), "http://") && !strings.HasPrefix(strings.ToLower(img), "https://") {
			utils.ResponderErro(w, http.StatusBadRequest, "imagem_url deve ser http ou https")
			return
		}
		link := strings.TrimSpace(c.LinkURL)
		if link != "" && !strings.HasPrefix(strings.ToLower(link), "http://") && !strings.HasPrefix(strings.ToLower(link), "https://") {
			utils.ResponderErro(w, http.StatusBadRequest, "link_url deve ser http ou https")
			return
		}
		lista = append(lista, &modelos.AppCardRede{
			Slot:      c.Slot,
			Titulo:    strings.TrimSpace(c.Titulo),
			ImagemURL: img,
			LinkURL:   link,
			Ativo:     c.Ativo,
		})
	}
	if err := h.appCardsRepo.SubstituirPorRede(idRede, lista); err != nil {
		if strings.Contains(err.Error(), "slot invalido") {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("salvar app cards: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar cards")
		return
	}
	itens, err := h.appCardsRepo.ListarPorRedeID(idRede)
	if err != nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "cards salvos"})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "cards salvos",
		"cards":    h.enriquecerRespostaAppCards(itens),
	})
}

func (h *Handlers) enriquecerRespostaAppCards(itens []*modelos.AppCardRede) map[string]any {
	var destaque *modelos.AppCardRede
	slots := make([]*modelos.AppCardRede, 3)
	for _, c := range itens {
		if c.Slot == 0 {
			destaque = c
		} else if c.Slot >= 1 && c.Slot <= 3 {
			slots[c.Slot-1] = c
		}
	}
	promos := make([]*modelos.AppCardRede, 0, 3)
	for _, p := range slots {
		if p != nil {
			promos = append(promos, p)
		}
	}
	return map[string]any{
		"lista":          itens,
		"destaque_rede":  destaque,
		"promocoes":      promos,
		"slots_ocupados": len(itens),
	}
}

// PublicListarAppCardsRede GET /v1/public/rede-cards?id_rede=uuid — para o app cliente (sem auth).
func (h *Handlers) PublicListarAppCardsRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	_, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrRedeNaoEncontrada) {
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao validar rede")
		return
	}
	itens, err := h.appCardsRepo.ListarPorRedeID(idRede)
	if err != nil {
		log.Printf("public listar app cards: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar cards")
		return
	}
	visiveis := filtrarCardsPublicos(itens)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede": idRede,
		"cards":   h.enriquecerRespostaAppCards(visiveis),
	})
}

func filtrarCardsPublicos(itens []*modelos.AppCardRede) []*modelos.AppCardRede {
	out := make([]*modelos.AppCardRede, 0, len(itens))
	for _, c := range itens {
		if c == nil || !c.Ativo {
			continue
		}
		if strings.TrimSpace(c.ImagemURL) == "" {
			continue
		}
		out = append(out, c)
	}
	return out
}
