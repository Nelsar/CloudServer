package database

import (
	"context"
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/influx"
	"gitlab.citicom.kz/CloudServer/server/models"
)

func (db *DB) GetMnemoscheme(ctx context.Context, mnemoID int64) (*models.MnemoResult, error) {
	mnemo := &models.MnemoResult{}
	var fileUrl sql.NullString
	var info sql.NullString
	if err := db.sql.QueryRow(`SELECT 
			m.mnemo_id, 
			m.name,
			m.company_id,
			m.file_url,
			m.info
			FROM mnemo AS m WHERE m.mnemo_id=?`, mnemoID).Scan(
		&mnemo.MnemoId,
		&mnemo.Name,
		&mnemo.CompanyId,
		&fileUrl,
		&info,
	); err != nil {
		return nil, err
	}
	mnemo.FileUrl = fileUrl.String
	mnemo.Info = info.String

	return mnemo, nil
}

func (db *DB) GetMnemoschemes(ctx context.Context, companyID int64, all bool) ([]*models.MnemoResult, error) {
	l, _ := icontext.GetLogger(ctx)
	query := `SELECT 
	m.mnemo_id, 
	m.name,
	m.company_id,
	m.file_url,
	m.info
	FROM mnemo AS m
	`
	var rows *sql.Rows
	var err error

	if !all {
		query = fmt.Sprintf(`%s WHERE m.company_id=?`, query)
		rows, err = db.sql.Query(query, companyID)
	} else {
		rows, err = db.sql.Query(query)
	}

	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("Get oilFields error")
		return nil, err
	}
	defer rows.Close()

	mnemos := make([]*models.MnemoResult, 0, 10)
	for rows.Next() {
		mnemo := &models.MnemoResult{}
		var fileUrl sql.NullString
		err := rows.Scan(
			&mnemo.MnemoId,
			&mnemo.Name,
			&mnemo.CompanyId,
			&fileUrl,
			&mnemo.Info,
		)
		if err != nil {
			l.WithFields(log.Fields{
				"ERROR": err,
			}).Error("Scan mnemo error")
			continue
		}

		mnemo.FileUrl = fileUrl.String
		mnemos = append(mnemos, mnemo)
	}

	return mnemos, nil
}

func (db *DB) SaveMnemoschemes(ctx context.Context, model models.MnemoResult) (*models.MnemoResult, error) {
	if model.MnemoId > 0 {
		if _, err := db.sql.Exec(
			`UPDATE mnemo SET name=?, company_id=?, file_url=?, info=?
					WHERE mnemo_id=?`,
			model.Name,
			model.CompanyId,
			model.FileUrl,
			model.Info,
			model.MnemoId,
		); err != nil {
			return nil, err
		}

		return db.GetMnemoscheme(ctx, model.MnemoId)
	} else {
		result, err := db.sql.Exec(
			`INSERT INTO mnemo(name, company_id, info, file_url)
											VALUES(?, ?, ?, ?)`,
			model.Name,
			model.CompanyId,
			model.Info,
			model.FileUrl,
		)
		if err != nil {
			return nil, err
		}

		lastID, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}

		return db.GetMnemoscheme(ctx, lastID)
	}
}

func (db *DB) GetLatestSensorValues(ctx context.Context, influxDB *influx.Influx, sensorIds []string) []*models.MnemoSensorDataResult {
	result := make([]*models.MnemoSensorDataResult, 0, 10)
	for _, sensorID := range sensorIds {
		model := &models.MnemoSensorDataResult{}

		if err := db.sql.QueryRow(`SELECT 
						s.sensor_id, 
						s.unit,
						s.range_h,
						s.range_l
						FROM sensors AS s 
						WHERE s.sensor_id=? LIMIT 0, 1`, sensorID).Scan(
			&model.SensorID,
			&model.Unit,
			&model.RangeH,
			&model.RangeL,
		); err != nil {
			continue
		}

		date, value := influxDB.GetLatestSensorValue(sensorID)
		model.FormattedValue = value
		model.CreatedTs = date

		result = append(result, model)
	}

	return result
}

func (db *DB) SaveMnemoschemeInfo(ctx context.Context, mnemoID int64, info string) (*models.MnemoResult, error) {
	if _, err := db.sql.Exec(
		`UPDATE mnemo SET info=? WHERE mnemo_id=?`,
		info,
		mnemoID,
	); err != nil {
		return nil, err
	}

	return db.GetMnemoscheme(ctx, mnemoID)
}
