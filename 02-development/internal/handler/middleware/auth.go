package middleware

import (
	"context"
	"net/http"
	"strings"

	"pottery-api/internal/service/auth"
)

type contextKey string

const ContextClientID contextKey = "client_id"

func Auth(jwt *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			claims, err := jwt.Validate(parts[1])
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextClientID, claims.ClientID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClientID(ctx context.Context) string {
	v, _ := ctx.Value(ContextClientID).(string)
	return v
}