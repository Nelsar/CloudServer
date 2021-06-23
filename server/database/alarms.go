package database

import (
	"context"
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
)

func (db *DB) GetAlarm(ctx context.Context, alarmID int64) (*models.AlarmResult, error) {
	alarm := &models.AlarmResult{}
	if err := db.sql.QueryRow(`SELECT 
 								a.alarm_id,
 								a.user_id,
 								a.oil_field_id,
 								a.controller_id,
 								a.sensor_id,
 								a.alarm_type,
 								a.alarm_value,
 								a.value,
 								a.viewed,
 								a.time
 								FROM alarms AS a WHERE a.alarm_id=?`,
		alarmID,
	).Scan(
		&alarm.AlarmID,
		&alarm.UserID,
		&alarm.OilFieldId,
		&alarm.ControllerID,
		&alarm.SensorID,
		&alarm.AlarmType,
		&alarm.AlarmValue,
		&alarm.Value,
		&alarm.IsViewed,
		&alarm.Time,
	); err != nil {
		return nil, err
	}

	return alarm, nil
}

func (db *DB) GetAlarms(ctx context.Context, companyID int64, all bool) ([]*models.AlarmResult, error) {
	l, _ := icontext.GetLogger(ctx)
	query := `SELECT 
	a.alarm_id,
	a.user_id,
	a.oil_field_id,
	oi.name,
	a.controller_id,
	c.name,
	a.sensor_id,
	a.alarm_type,
	a.alarm_value,
	a.value,
	a.viewed,
	a.time
	FROM alarms a 
	JOIN oil_field oi
	ON a.oil_field_id = oi.oil_field_id
	JOIN controllers c
	ON a.controller_id = c.controller_id`

	var rows *sql.Rows
	var err error

	if !all {
		query = fmt.Sprintf(`%s WHERE oi.company_id=?`, query)
		rows, err = db.sql.Query(query, companyID)
	} else {
		rows, err = db.sql.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	alarms := make([]*models.AlarmResult, 0, 10)
	for rows.Next() {
		alarm := &models.AlarmResult{}
		err := rows.Scan(
			&alarm.AlarmID,
			&alarm.UserID,
			&alarm.OilFieldId,
			&alarm.OilFieldName,
			&alarm.ControllerID,
			&alarm.ControllerName,
			&alarm.SensorID,
			&alarm.AlarmType,
			&alarm.AlarmValue,
			&alarm.Value,
			&alarm.IsViewed,
			&alarm.Time,
		)
		if err != nil {
			l.WithFields(log.Fields{
				"Error": err,
			}).Error("Scan alarm error")
			continue
		}
		alarms = append(alarms, alarm)
	}

	return alarms, nil
}

func (db *DB) MarkAlarmAsViewed(ctx context.Context, alarmID int64) (*models.AlarmResult, error) {
	_, err := db.sql.Exec(`UPDATE alarms SET viewed=? WHERE alarm_id=?`, true, alarmID)
	if err != nil {
		return nil, err
	}

	return db.GetAlarm(ctx, alarmID)
}

func (db *DB) SaveAlarms(ctx context.Context, alarms models.Alarms) []*models.AlarmResult {
	alarmResults := make([]*models.AlarmResult, 0, 10)
	for _, alarm := range alarms.Alarms {
		oilField, err := db.GetOilField(ctx, alarm.OilFieldID)
		if err != nil {
			continue
		}

		companyID := oilField.CompanyID
		if db.AlarmExists(ctx, alarm.SensorID, alarm.AlarmType, alarm.AlarmValue, alarm.Time) {
			continue
		}

		users, err := db.GetUsers(ctx, companyID, false)
		if err != nil {
			continue
		}

		for _, user := range users {
			result, err := db.sql.Exec(`INSERT INTO alarms(
								user_id, 
								oil_field_id, 
								controller_id, 
								sensor_id, 
								alarm_type, 
								alarm_value, 
								value, 
								viewed, 
								time) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				user.UserID,
				oilField.OilFieldId,
				alarm.ControllerID,
				alarm.SensorID,
				alarm.AlarmType,
				alarm.AlarmValue,
				alarm.Value,
				false,
				alarm.Time,
			)
			if err != nil {
				continue
			}
			alarmId, err := result.LastInsertId()
			if err != nil {
				continue
			}

			alarmResult, err := db.GetAlarm(ctx, alarmId)
			if err != nil {
				continue
			}

			alarmResults = append(alarmResults, alarmResult)
		}

	}

	return alarmResults
}

func (db *DB) AlarmExists(ctx context.Context, sensorID string, alarmType string, alarmValue float32, time int64) bool {
	timeBefore := time - 600
	timeAfter := time + 600

	return db.RowExists(
		ctx,
		`SELECT alarm_id FROM alarms WHERE sensor_id=? AND alarm_type=? AND alarm_value=? AND time BETWEEN ? AND ?`,
		sensorID,
		alarmType,
		alarmValue,
		timeBefore,
		timeAfter,
	)
}
