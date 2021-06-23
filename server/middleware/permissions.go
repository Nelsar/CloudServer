package middleware

import (
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/response"
	"net/http"
)

func PermissionsMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l, _ := icontext.GetLogger(ctx)
			user, ok := icontext.GetUser(ctx)

			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			role := user.GetRole()

			if role.Can(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}


			response.ErrorResponse(l, w, http.StatusForbidden, "Access denied permission", nil)
		})
	}
}