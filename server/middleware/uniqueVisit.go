package middleware

import (
	"context"
	"fmt"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/utils"
	"net/http"
)

func UniqueVisitMiddleware(routes []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, v := range routes {
				if r.URL.Path == v {
					guid := fmt.Sprintf("%s|%s", r.Header.Get("User-Agent"), r.RemoteAddr)
					uniqueIdentifier := utils.GetMD5Hash(guid)

					ctx := context.WithValue(r.Context(), icontext.UniqueVisitContextKey, uniqueIdentifier)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			ctx := r.Context()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
