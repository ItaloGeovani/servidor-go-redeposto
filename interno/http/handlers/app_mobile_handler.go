package handlers

import (
	"log"
	"net/http"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

func (h *Handlers) VerificarVersaoAppMobile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	q := r.URL.Query()
	plataforma := strings.ToLower(strings.TrimSpace(q.Get("plataforma")))
	instalada := strings.TrimSpace(q.Get("versao_instalada"))
	if instalada == "" {
		instalada = strings.TrimSpace(q.Get("versao"))
	}
	if plataforma != "ios" && plataforma != "android" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe plataforma=ios ou plataforma=android")
		return
	}
	if instalada == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe versao_instalada (ex.: 1.0.0)")
		return
	}

	cfg, err := h.appMobileRepo.Obter()
	if err != nil {
		log.Printf("verificar versao app mobile: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}

	var atual string
	var urlLoja string
	if plataforma == "ios" {
		atual = cfg.VersaoIOS
		urlLoja = cfg.URLLojaIOS
	} else {
		atual = cfg.VersaoAndroid
		urlLoja = cfg.URLLojaAndroid
	}
	if strings.TrimSpace(atual) == "" {
		atual = "0.0.0"
	}

	desatualizada := utils.VersaoSemverMenor(instalada, atual)
	atualizacaoDisponivel := desatualizada

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"plataforma":                 plataforma,
		"versao_instalada":           instalada,
		"versao_atual_servidor":      atual,
		"atualizacao_disponivel":     atualizacaoDisponivel,
		"instalada_desatualizada":    desatualizada,
		"url_loja":                   urlLoja,
		"mensagem":                   cfg.MensagemAtualizacao,
		"atualizacao_obrigatoria":    cfg.AtualizacaoObrigatoria,
		"deve_exibir_modal_atualizar": atualizacaoDisponivel,
	})
}

// AppMobileVersaoAdmin GET le / PUT salva configuracao dos apps (super admin).
func (h *Handlers) AppMobileVersaoAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.obterConfigAppMobileAdmin(w, r)
	case http.MethodPut, http.MethodPatch:
		h.salvarConfigAppMobileAdmin(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) obterConfigAppMobileAdmin(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.appMobileRepo.Obter()
	if err != nil {
		log.Printf("obter config app mobile admin: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"configuracao": cfg,
	})
}

type reqSalvarAppMobile struct {
	VersaoIOS              string `json:"versao_ios"`
	VersaoAndroid          string `json:"versao_android"`
	URLLojaIOS             string `json:"url_loja_ios"`
	URLLojaAndroid         string `json:"url_loja_android"`
	MensagemAtualizacao    string `json:"mensagem_atualizacao"`
	AtualizacaoObrigatoria *bool  `json:"atualizacao_obrigatoria"`
}

func (h *Handlers) salvarConfigAppMobileAdmin(w http.ResponseWriter, r *http.Request) {
	var req reqSalvarAppMobile
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	atual, err := h.appMobileRepo.Obter()
	if err != nil {
		log.Printf("salvar config app mobile obter: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao atual")
		return
	}
	out := &modelos.ConfiguracaoAppMobile{
		VersaoIOS:              strings.TrimSpace(req.VersaoIOS),
		VersaoAndroid:          strings.TrimSpace(req.VersaoAndroid),
		URLLojaIOS:             strings.TrimSpace(req.URLLojaIOS),
		URLLojaAndroid:         strings.TrimSpace(req.URLLojaAndroid),
		MensagemAtualizacao:    strings.TrimSpace(req.MensagemAtualizacao),
		AtualizacaoObrigatoria: atual.AtualizacaoObrigatoria,
	}
	if req.AtualizacaoObrigatoria != nil {
		out.AtualizacaoObrigatoria = *req.AtualizacaoObrigatoria
	}
	if out.VersaoIOS == "" {
		out.VersaoIOS = "0.0.0"
	}
	if out.VersaoAndroid == "" {
		out.VersaoAndroid = "0.0.0"
	}
	if err := h.appMobileRepo.Salvar(out); err != nil {
		log.Printf("salvar config app mobile: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar configuracao")
		return
	}
	salvo, err := h.appMobileRepo.Obter()
	if err != nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "configuracao salva"})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":       "configuracao salva",
		"configuracao": salvo,
	})
}
