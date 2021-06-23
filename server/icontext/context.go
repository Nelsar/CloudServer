package icontext

import (
	"context"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/models"
)

type key string

const (
	UserContext      = key("userContext")
	LoggerContextKey = key("loggerContextKey")
	UniqueVisitContextKey = key("uniqueVisitContextKey")
)

// GetUserID - return user id from context if it exists.
func GetUser(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(UserContext).(*models.User)
	return u, ok
}

// GetLogger - return logger instance from context if it exists.
func GetLogger(ctx context.Context) (*log.Entry, bool) {
	u, ok := ctx.Value(LoggerContextKey).(*log.Entry)
	return u, ok
}

// GetLogger - return logger instance from context if it exists.
func GetUniqueIdentifier(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(UniqueVisitContextKey).(string)
	return u, ok
}
