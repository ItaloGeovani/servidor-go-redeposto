package middlewares

import (
	"log"
	"net/http"
	"time"
)

func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inicio := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("metodo=%s rota=%s request_id=%s duracao=%s", r.Method, r.URL.Path, ObterRequestID(r.Context()), time.Since(inicio))
		})
	}
}
