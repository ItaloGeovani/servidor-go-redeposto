package middlewares

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

func ExigirAutenticacao(aut servicos.Autenticador) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			token := extrairBearer(authorization)
			if token == "" {
				log.Printf("auth_falhou rota=%s motivo=token_ausente authorization=%q", r.URL.Path, authorization)
				utils.ResponderErro(w, http.StatusUnauthorized, "token ausente")
				return
			}

			usuario, err := aut.ValidarToken(token)
			if err != nil {
				log.Printf("auth_falhou rota=%s token=%s erro=%v", r.URL.Path, mascararToken(token), err)
				utils.ResponderErro(w, http.StatusUnauthorized, fmt.Sprintf("token invalido: %v", err))
				return
			}

			next.ServeHTTP(w, r.WithContext(ComUsuario(r.Context(), usuario)))
		})
	}
}

func ExigirPapel(papel modelos.Papel) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			usuario := Usuario(r.Context())
			if usuario == nil {
				utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
				return
			}
			if usuario.Papel != papel {
				utils.ResponderErro(w, http.StatusForbidden, "acesso negado para este papel")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BearerToken extrai o token do cabecalho Authorization: Bearer <token>.
func BearerToken(r *http.Request) string {
	return extrairBearer(r.Header.Get("Authorization"))
}

func extrairBearer(cabecalho string) string {
	cabecalho = strings.TrimSpace(cabecalho)
	if cabecalho == "" {
		return ""
	}

	prefixo := "Bearer "
	if !strings.HasPrefix(cabecalho, prefixo) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(cabecalho, prefixo))
}

func mascararToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return "<vazio>"
	}
	if len(token) <= 12 {
		return token
	}
	return token[:6] + "..." + token[len(token)-4:]
}
