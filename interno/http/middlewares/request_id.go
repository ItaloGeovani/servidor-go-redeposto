package middlewares

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

var contadorRequest uint64

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				seq := atomic.AddUint64(&contadorRequest, 1)
				id = "req-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(seq, 10)
			}

			w.Header().Set("X-Request-ID", id)
			ctx := ComRequestID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
