package database

import (
	"context"
	"database/sql"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
	"time"
)

func (db *DB) GetCompanyData(ctx context.Context, companyID int64, dataType string) (*models.CompanyData, error) {
	l, _ := icontext.GetLogger(ctx)
	var desktopValue sql.NullString
	if err := db.sql.QueryRow(`SELECT 
		cd.value
		FROM company_data AS cd 
		WHERE cd.company_id=? AND type=?`, companyID, dataType).Scan(
		&desktopValue,
	); err != nil {
		l.Errorf("SELECT CompanyData ERROR: %f", err.Error())
		return nil, err
	}

	return &models.CompanyData{
		Value: desktopValue.String,
	}, nil
}

func (db *DB) SaveCompanyData(ctx context.Context, companyID int64, jsonValue string, dataType string) (*models.CompanyData, error) {
	_, err := db.sql.Exec(
		`REPLACE INTO company_data SET company_id=?, value=?, type=?, updated_ts=?`,
		companyID,
		jsonValue,
		dataType,
		time.Now().Unix(),
	)
	if err != nil {
		return nil, err
	}

	return db.GetCompanyData(ctx, companyID, dataType)
}