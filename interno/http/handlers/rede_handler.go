package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqCriarRede struct {
	NomeFantasia     string  `json:"nome_fantasia"`
	RazaoSocial      string  `json:"razao_social"`
	CNPJ             string  `json:"cnpj"`
	EmailContato     string  `json:"email_contato"`
	Telefone         string  `json:"telefone"`
	ValorImplantacao float64 `json:"valor_implantacao"`
	ValorMensalidade float64 `json:"valor_mensalidade"`
	PrimeiroCobranca string  `json:"primeiro_cobranca"`
}

type reqEditarRede struct {
	ID               string  `json:"id"`
	NomeFantasia     string  `json:"nome_fantasia"`
	RazaoSocial      string  `json:"razao_social"`
	CNPJ             string  `json:"cnpj"`
	EmailContato     string  `json:"email_contato"`
	Telefone         string  `json:"telefone"`
	ValorImplantacao float64 `json:"valor_implantacao"`
	ValorMensalidade float64 `json:"valor_mensalidade"`
	PrimeiroCobranca string  `json:"primeiro_cobranca"`
}

type reqMudarStatusRede struct {
	ID string `json:"id"`
}

type reqEditarMoedaVirtualRede struct {
	ID                  string  `json:"id"`
	MoedaVirtualNome    string  `json:"moeda_virtual_nome"`
	MoedaVirtualCotacao float64 `json:"moeda_virtual_cotacao"`
}

func (h *Handlers) ListarRedesDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	redes, err := h.redeService.Listar()
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar redes")
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens": redes,
		"total": len(redes),
	})
}

func (h *Handlers) CriarRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCriarRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, fmt.Sprintf("payload invalido: %v", err))
		return
	}

	rede, err := h.redeService.Criar(servicos.CriarRedeInput{
		NomeFantasia:     req.NomeFantasia,
		RazaoSocial:      req.RazaoSocial,
		CNPJ:             req.CNPJ,
		EmailContato:     req.EmailContato,
		Telefone:         req.Telefone,
		ValorImplantacao: req.ValorImplantacao,
		ValorMensalidade: req.ValorMensalidade,
		PrimeiroCobranca: req.PrimeiroCobranca,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrRedeCNPJDuplicado), errors.Is(err, repositorios.ErrRedeNomeDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao criar rede")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "rede criada com sucesso",
		"rede":     rede,
	})
}

func (h *Handlers) EditarRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqEditarRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, fmt.Sprintf("payload invalido: %v", err))
		return
	}

	rede, err := h.redeService.Editar(servicos.EditarRedeInput{
		ID:               req.ID,
		NomeFantasia:     req.NomeFantasia,
		RazaoSocial:      req.RazaoSocial,
		CNPJ:             req.CNPJ,
		EmailContato:     req.EmailContato,
		Telefone:         req.Telefone,
		ValorImplantacao: req.ValorImplantacao,
		ValorMensalidade: req.ValorMensalidade,
		PrimeiroCobranca: req.PrimeiroCobranca,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrRedeCNPJDuplicado), errors.Is(err, repositorios.ErrRedeNomeDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao editar rede")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "rede atualizada com sucesso",
		"rede":     rede,
	})
}

func (h *Handlers) EditarMoedaVirtualRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqEditarMoedaVirtualRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, fmt.Sprintf("payload invalido: %v", err))
		return
	}

	rede, err := h.redeService.EditarMoedaVirtual(servicos.EditarMoedaVirtualRedeInput{
		ID:                  req.ID,
		MoedaVirtualNome:    req.MoedaVirtualNome,
		MoedaVirtualCotacao: req.MoedaVirtualCotacao,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id, nome da moeda e cotacao maior que zero")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar moeda virtual")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "moeda virtual atualizada com sucesso",
		"rede":     rede,
	})
}

func (h *Handlers) AtivarRedeDev(w http.ResponseWriter, r *http.Request) {
	h.mudarStatusRedeDev(w, r, true)
}

func (h *Handlers) DesativarRedeDev(w http.ResponseWriter, r *http.Request) {
	h.mudarStatusRedeDev(w, r, false)
}

func (h *Handlers) mudarStatusRedeDev(w http.ResponseWriter, r *http.Request, ativa bool) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqMudarStatusRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, fmt.Sprintf("payload invalido: %v", err))
		return
	}

	var rede *modelos.Rede
	var err error

	if ativa {
		rede, err = h.redeService.Ativar(req.ID)
	} else {
		rede, err = h.redeService.Desativar(req.ID)
	}
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar status da rede")
		}
		return
	}

	mensagem := "rede ativada com sucesso"
	if !ativa {
		mensagem = "rede desativada com sucesso"
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": mensagem,
		"rede":     rede,
	})
}
