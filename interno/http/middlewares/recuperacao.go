package middlewares

import (
	"log"
	"net/http"

	"gaspass-servidor/utils"
)

func RecuperacaoPanico() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic capturado request_id=%s erro=%v", ObterRequestID(r.Context()), rec)
					utils.ResponderErro(w, http.StatusInternalServerError, "erro interno")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
