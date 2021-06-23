package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
	"gitlab.citicom.kz/CloudServer/server/utils"
)

func (db *DB) GetUsers(ctx context.Context, companyID int64, all bool) ([]*models.User, error) {
	l, _ := icontext.GetLogger(ctx)

	query := `SELECT 
	u.user_id,
	u.company_id,
	c.name,
	u.email,
	u.first_name,
	u.last_name,
	u.role,
	u.is_deleted,
	u.created_ts,
	u.updated_ts
	FROM users AS u
	LEFT JOIN company c
	ON u.company_id=c.company_id
	`
	var rows *sql.Rows
	var err error

	if !all {
		query = fmt.Sprintf(`%s WHERE u.company_id=?`, query)
		rows, err = db.sql.Query(query, companyID)
	} else {
		rows, err = db.sql.Query(query)
	}

	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("Get controllers error")
		return nil, err
	}
	defer rows.Close()
	users := make([]*models.User, 0, 10)
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.UserID,
			&user.CompanyID,
			&user.CompanyName,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.IsDeleted,
			&user.CreatedTs,
			&user.UpdatedTs,
		)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func (db *DB) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	l, _ := icontext.GetLogger(ctx)
	user := &models.User{}
	if err := db.sql.QueryRow(`SELECT 
		u.user_id,
		u.company_id,
		u.email,
		u.first_name,
		u.last_name,
		u.role,
		u.is_deleted,
		u.created_ts,
		u.updated_ts
		FROM users AS u 
		WHERE u.user_id=?`, userID).Scan(
		&user.UserID,
		&user.CompanyID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsDeleted,
		&user.CreatedTs,
		&user.UpdatedTs,
	); err != nil {
		l.Errorf("SELECT Company ERROR: %f", err.Error())
		return nil, err
	}

	return user, nil
}

func (db *DB) userExists(ctx context.Context, userID int64) bool {
	return db.RowExists(
		ctx,
		`SELECT u.user_id FROM users AS u WHERE u.user_id=?`,
		userID,
	)
}

func (db *DB) DeleteUser(ctx context.Context, userID int64) (*models.User, error) {
	_, err := db.sql.Exec(`UPDATE users SET is_deleted=? WHERE user_id=?`, true, userID)
	if err != nil {
		return nil, err
	}

	return db.GetUser(ctx, userID)
}

func (db *DB) EmailExists(ctx context.Context, email string, userID int64) bool {
	if userID != 0 {
		return db.RowExists(
			ctx,
			`SELECT u.user_id FROM users AS u WHERE u.email=? AND u.user_id !=?`,
			email,
			userID,
		)
	}

	return db.RowExists(
		ctx,
		`SELECT u.user_id FROM users AS u WHERE u.email=?`,
		email,
	)
}

func (db *DB) CreateUser(ctx context.Context, model models.CreateUser) (*models.User, error) {
	encryptedPass := utils.GetMD5Hash(model.Password)
	result, err := db.sql.Exec(
		`INSERT INTO users(company_id, email, first_name, last_name, password, role, is_deleted, created_ts,updated_ts)
											VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		model.CompanyID,
		model.Email,
		model.FirstName,
		model.LastName,
		encryptedPass,
		model.Role,
		false,
		time.Now().Unix(),
		time.Now().Unix(),
	)
	if err != nil {
		return nil, err
	}
	userID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetUser(ctx, userID)
}

func (db *DB) UpdateUser(ctx context.Context, model models.UpdateUser) (*models.User, error) {
	_, err := db.sql.Exec(
		`UPDATE users SET company_id=?, email=?, first_name=?, last_name=?, role=?, is_deleted=?, updated_ts=? WHERE user_id=?`,
		model.CompanyID,
		model.Email,
		model.FirstName,
		model.LastName,
		model.Role,
		model.IsDeleted,
		time.Now().Unix(),
		model.UserID,
	)
	if err != nil {
		return nil, err
	}

	return db.GetUser(ctx, model.UserID)
}

func (db *DB) ChangeUserPassword(ctx context.Context, userID int64, password string) (*models.User, error) {
	encryptedPass := utils.GetMD5Hash(password)
	_, err := db.sql.Exec(
		`UPDATE users SET password=? WHERE user_id=?`,
		encryptedPass,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return db.GetUser(ctx, userID)
}
