package config

import "gaspass-servidor/utils"

type Config struct {
	Ambiente              string
	PortaHTTP             int
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
		PortaHTTP:             utils.ObterEnvInt("APP_PORTA", 8080),
		TokenPadraoAPI:        utils.ObterEnv("API_TOKEN_PADRAO", "dev-super-admin"),
		AdminNomePadrao:       utils.ObterEnv("ADMIN_NOME_PADRAO", "Administrador Geral"),
		AdminEmailPadrao:      utils.ObterEnv("ADMIN_EMAIL_PADRAO", "admin@gaspass.local"),
		AdminSenhaPadrao:      utils.ObterEnv("ADMIN_SENHA_PADRAO", "123456"),
		AdminBootstrapAtivado: utils.ObterEnv("ADMIN_BOOTSTRAP_ATIVADO", "true") == "true",
		CORSOrigemPermitida:   utils.ObterEnv("CORS_ORIGEM_PERMITIDA", "http://localhost:5173"),
	}
}
