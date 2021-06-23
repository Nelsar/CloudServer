package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/database"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
	"gitlab.citicom.kz/CloudServer/server/response"
)

func LogMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			guid := xid.New()
			requestID := guid.String()
			requestLogger := log.WithFields(log.Fields{"request_id": requestID})
			requestLogger.WithFields(log.Fields{
				"Path":   r.URL.Path,
				"Method": r.Method,
			}).Info("Incoming request")
			ctx := context.WithValue(r.Context(), icontext.LoggerContextKey, requestLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ActionLogMiddleware(db *database.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, _ := icontext.GetUser(ctx)
			l, _ := icontext.GetLogger(ctx)
			action := models.ActionLogResult{
				Action:    r.URL.Path,
				UserID:    user.UserID,
				CompanyID: user.CompanyID,
				Time:      time.Now().Unix(),
			}

			_, err := db.SaveActionLog(ctx, action)
			if err != nil {
				l.Errorf("Error in ActionLog middleware: %s ", err.Error())
				response.ErrorResponse(l, rw, http.StatusBadRequest, "Bad Request", nil)
				return
			}
			ctx = context.WithValue(ctx, icontext.LoggerContextKey, user)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
