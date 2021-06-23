package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
	"gitlab.citicom.kz/CloudServer/server/utils"
	"time"
)

var NotFound = errors.New("Not found")
var MinuteBlocked = errors.New("MinuteBlocked")

func (db *DB) GetUserBySessionID(ctx context.Context, sessionId string) (*models.User, error) {
	l, _ := icontext.GetLogger(ctx)
	user := &models.User{}

	if err := db.sql.QueryRow(`SELECT 
				u.user_id, 
				u.company_id,
				u.email,
				u.role,
				u.created_ts
			FROM sessions AS s
			LEFT JOIN users AS u ON u.user_id = s.user_id 
			WHERE s.session_id= ?
			LIMIT 1`, sessionId).Scan(&user.UserID, &user.CompanyID, &user.Email, &user.Role, &user.CreatedTs); err == nil {
		return user, nil
	} else if err == sql.ErrNoRows {
		return nil, NotFound
	} else {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("GetUserBySessionID error")
		return nil, err
	}
}

func (db *DB) CheckAuth(ctx context.Context, email, password string, uid string) (*models.User, error) {
	if len(uid) == 0 {
		return nil, fmt.Errorf("uid not found")
	}

	l, _ := icontext.GetLogger(ctx)
	user := &models.User{}
	encryptedPass := utils.GetMD5Hash(password)

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	var lastAuthTs int64
	if err := tx.sql.QueryRow(`SELECT created_ts 
			FROM auth_attempt 
			WHERE hash=? 
			ORDER BY created_ts DESC LIMIT 1`, uid).Scan(&lastAuthTs); err == nil || err == sql.ErrNoRows {
	} else {
		_ = tx.sql.Rollback()
		return nil, err
	}

	attemptCount := 0
	timeToCheck := time.Now().Unix() - 60
	if err := tx.sql.QueryRow(`SELECT COUNT(hash) FROM auth_attempt WHERE hash=? AND created_ts>=?`, uid, timeToCheck).Scan(&attemptCount); err != nil {
		_ = tx.sql.Rollback()
		return nil, err
	}

	if attemptCount >= 3 {
		_ = tx.sql.Rollback()
		return nil, MinuteBlocked
	}

	_, err = tx.sql.Exec(`INSERT INTO auth_attempt(hash, created_ts) VALUES(?, ?)`, uid, time.Now().Unix())
	if err != nil {
		_ = tx.sql.Rollback()
		return nil, err
	}

	if err := tx.sql.QueryRow(`SELECT
				u.user_id, 
				u.company_id,
				u.email,
				u.role,
				u.created_ts
			  FROM users AS u 
			  WHERE email=? AND password=?
			  LIMIT 1`, email, encryptedPass).Scan(&user.UserID, &user.CompanyID, &user.Email, &user.Role, &user.CreatedTs); err == nil {
			  	_, _ = tx.sql.Exec(`DELETE FROM auth_attempt WHERE hash=?`, uid)
		return user, tx.sql.Commit()
	} else if err == sql.ErrNoRows {
		_ = tx.sql.Commit()
		return nil, NotFound
	} else {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("GetUserByEmail error")
		_ = tx.sql.Commit()
		return nil, err
	}
}

func (db *DB) CreateSession(ctx context.Context, userID int64) *string {
	l, _ := icontext.GetLogger(ctx)
	sessionID := xid.New().String()

	_, _ = db.sql.Exec(`DELETE FROM sessions WHERE user_id=?`, userID)
	_, err := db.sql.Exec(`INSERT INTO sessions 
		(session_id, user_id, created_ts) 
	VALUES (?, ?, ?)`,
		sessionID,
		userID,
		time.Now().Unix(),
	)
	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("CreateSession error")
		return nil
	}
	return &sessionID
}
