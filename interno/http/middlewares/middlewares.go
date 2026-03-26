package middlewares

import "net/http"

type Middleware func(http.Handler) http.Handler

func Encadear(handler http.Handler, mws ...Middleware) http.Handler {
	if len(mws) == 0 {
		return handler
	}

	encadeado := handler
	for i := len(mws) - 1; i >= 0; i-- {
		encadeado = mws[i](encadeado)
	}
	return encadeado
}
