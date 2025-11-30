package middleware

import (
	"net/http"
	"time"

	"github.com/wb-go/wbf/zlog"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		zlog.Logger.Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Dur("duration", time.Since(start)).
			Msg("Request handled")
	})
}
