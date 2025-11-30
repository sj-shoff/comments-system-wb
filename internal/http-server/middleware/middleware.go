package middleware

import (
	"net/http"
	"time"

	"github.com/wb-go/wbf/zlog"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		zlog.Logger.Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Str("ip", r.RemoteAddr).
			Msg("Request started")

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		zlog.Logger.Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Dur("duration", duration).
			Msg("Request completed")
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				zlog.Logger.Error().
					Interface("error", err).
					Str("method", r.Method).
					Str("url", r.URL.String()).
					Msg("Panic recovered")

				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
