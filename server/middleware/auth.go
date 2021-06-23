package middleware

import (
	"context"
	"gitlab.citicom.kz/CloudServer/server/database"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/response"
	"net/http"
)

const (
	AuthTokenName = "AuthToken"
)

// AuthMiddleware - create an auth middleware
func AuthMiddleware(db *database.DB, nonAuthList []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l, _ := icontext.GetLogger(ctx)

			for _, v := range nonAuthList {
				if r.URL.Path == v {
					l.Infof("Non auth route")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			sessionToken := r.Header.Get(AuthTokenName)

			if sessionToken == "" {
				sessionToken = r.URL.Query().Get(AuthTokenName)

				if sessionToken == "" {
					l.Errorf("%s is missing", AuthTokenName)
					response.ErrorResponse(l, w, http.StatusUnauthorized, "Access denied", nil)
					return
				}
			}

			//TODO: by token get userId
			user, err := db.GetUserBySessionID(ctx, sessionToken)

			if err != nil {
				l.Errorf("Error in auth middleware: %s ", err.Error())
				response.ErrorResponse(l, w, http.StatusUnauthorized, "Access denied", nil)
				return
			}

			ctx = context.WithValue(ctx, icontext.UserContext, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}