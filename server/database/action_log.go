package database

import (
	"context"
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
)

func (db *DB) GetActionLog(ctx context.Context, actionID int64) (*models.ActionLogResult, error) {
	l, _ := icontext.GetLogger(ctx)
	var action = &models.ActionLogResult{}
	if err := db.sql.QueryRow(`SELECT 
		ac.action_id,
		ac.action,
		ac.user_id,
		u.first_name,
		u.last_name,
		ac.company_id,
		c.name,
		ac.time
		FROM action_log AS ac
		JOIN users u
		ON ac.user_id = u.user_id
		JOIN company c
		ON ac.company_id = c.company_id
		WHERE ac.action_id=?`, actionID).Scan(
		&action.ActionID,
		&action.Action,
		&action.UserID,
		&action.CompanyID,
		&action.Time,
	); err != nil {
		l.Errorf("Action list request error: %v", err)
		return nil, err
	}

	return action, nil
}

func (db *DB) GetActionsLogs(ctx context.Context, companyID int64, all bool) ([]*models.ActionLogResult, error) {
	l, _ := icontext.GetLogger(ctx)
	query := `SELECT 
			ac.action_id,
			ac.action,
			ac.user_id,
			u.first_name,
			u.last_name,
			ac.company_id,
			c.name,
			ac.time
			FROM action_log AS ac
			JOIN users u
			ON ac.user_id = u.user_id
			JOIN company c
			ON ac.company_id = c.company_id`

	var rows *sql.Rows
	var err error

	if !all {
		query = fmt.Sprintf(`%s WHERE ac.company_id=?`, query)
		rows, err = db.sql.Query(query, companyID)
	} else {
		rows, err = db.sql.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	actions := make([]*models.ActionLogResult, 0, 10)
	for rows.Next() {
		action := &models.ActionLogResult{}
		err := rows.Scan(
			&action.ActionID,
			&action.Action,
			&action.UserID,
			&action.FirstName,
			&action.LastName,
			&action.CompanyID,
			&action.CompanyName,
			&action.Time,
		)
		if err != nil {
			l.WithFields(log.Fields{
				"Error": err,
			}).Error("Scan action_log error")
			continue
		}
		actions = append(actions, action)
	}

	return actions, err

}

func (db *DB) SaveActionLog(ctx context.Context, model models.ActionLogResult) (*models.ActionLogResult, error) {
	l, _ := icontext.GetLogger(ctx)
	l.Infof("SaveActionLog: %s", model.Action)
	result, err := db.sql.Exec(`INSERT INTO action_log(
		action_id,  
		action, 
		user_id, 
		company_id, 
		time) 
		VALUES(?,?,?,?,?)`,
		model.ActionID,
		model.Action,
		model.UserID,
		model.CompanyID,
		model.Time)
	if err != nil {
		l.Errorf("%v", err)
		return nil, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		l.Errorf("%v", err)
		return nil, err
	}

	return db.GetActionLog(ctx, lastID)

}
