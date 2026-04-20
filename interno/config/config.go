package config

import (
	"os"
	"strconv"
	"strings"

	"gaspass-servidor/utils"
)

type Config struct {
	Ambiente              string
	PortaHTTP      int
	// PastaPainelWeb: absoluta ou relativa com index.html do painel (PAINEL_WEB_ASSETS). Vazio = auto.
	PastaPainelWeb string
	// PublicBaseURL: URL base pública (https://api.seudominio.com) para montar webhook Mercado Pago e notification_url do PIX.
	PublicBaseURL string
	TokenPadraoAPI        string
	AdminNomePadrao       string
	AdminEmailPadrao      string
	AdminSenhaPadrao      string
	AdminBootstrapAtivado bool
	CORSOrigemPermitida   string
}

func Carregar() Config {
	return Config{
		Ambiente:              utils.ObterEnv("APP_AMBIENTE", "desenvolvimento"),
		PortaHTTP:             portaHTTP(),
		PastaPainelWeb:        strings.TrimSpace(utils.ObterEnv("PAINEL_WEB_ASSETS", "")),
		TokenPadraoAPI:        utils.ObterEnv("API_TOKEN_PADRAO", "dev-super-admin"),
		AdminNomePadrao:       utils.ObterEnv("ADMIN_NOME_PADRAO", "Administrador Geral"),
		AdminEmailPadrao:      utils.ObterEnv("ADMIN_EMAIL_PADRAO", "admin@gaspass.local"),
		AdminSenhaPadrao:      utils.ObterEnv("ADMIN_SENHA_PADRAO", "123456"),
		AdminBootstrapAtivado: utils.ObterEnv("ADMIN_BOOTSTRAP_ATIVADO", "true") == "true",
		CORSOrigemPermitida:   utils.ObterEnv("CORS_ORIGEM_PERMITIDA", "http://localhost:5173"),
		PublicBaseURL:         strings.TrimRight(strings.TrimSpace(utils.ObterEnv("PUBLIC_BASE_URL", "")), "/"),
	}
}

// portaHTTP: Heroku/Elastic costumam definir PORT; senao APP_PORTA; padrao 8080.
func portaHTTP() int {
	p := strings.TrimSpace(os.Getenv("PORT"))
	if p != "" {
		n, err := strconv.Atoi(p)
		if err == nil && n > 0 {
			return n
		}
	}
	return utils.ObterEnvInt("APP_PORTA", 8080)
}
