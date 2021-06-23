package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
)

func (db *DB) GetCompanies(ctx context.Context, companyID int64, all bool) ([]*models.Company, error) {
	l, _ := icontext.GetLogger(ctx)
	query := `SELECT 
	c.company_id,
	c.name,
	c.bin,
	c.iin,
	c.iic,
	c.bic,
	c.address,
	c.phone_numbers,
	c.is_deleted,
	c.created_ts,
	c.updated_ts
	FROM company AS c`

	var rows *sql.Rows
	var err error

	if !all {
		query = fmt.Sprintf(`%s WHERE c.company_id=?`, query)
		rows, err = db.sql.Query(query, companyID)
	} else {
		rows, err = db.sql.Query(query)
	}

	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("Get companies list error")
		return nil, err
	}

	defer rows.Close()
	companies := make([]*models.Company, 0, 10)
	for rows.Next() {
		company := &models.Company{}
		err := rows.Scan(
			&company.CompanyID,
			&company.Name,
			&company.Bin,
			&company.Iin,
			&company.Iic,
			&company.Bic,
			&company.Address,
			&company.PhoneNumbers,
			&company.IsDeleted,
			&company.CreatedTs,
			&company.UpdatedTs,
		)
		if err != nil {
			l.Errorf("%v", err)
			continue
		}
		companies = append(companies, company)
	}

	return companies, nil
}

func (db *DB) GetCompany(ctx context.Context, companyID int64) (*models.Company, error) {
	l, _ := icontext.GetLogger(ctx)
	var company = &models.Company{}
	if err := db.sql.QueryRow(`SELECT 
		c.company_id,
		c.name,
		c.bin,
		c.iin,
		c.iic,
		c.bic,
		c.address,
		c.phone_numbers,
		c.is_deleted,
		c.created_ts,
		c.updated_ts
		FROM company AS c 
		WHERE c.company_id=?`, companyID).Scan(
		&company.CompanyID,
		&company.Name,
		&company.Bin,
		&company.Iin,
		&company.Iic,
		&company.Bic,
		&company.Address,
		&company.PhoneNumbers,
		&company.IsDeleted,
		&company.CreatedTs,
		&company.UpdatedTs,
	); err != nil {
		l.Errorf("SELECT Company ERROR: %f", err.Error())
		return nil, err
	}

	return company, nil
}

func (db *DB) companyExists(ctx context.Context, companyID int64) bool {
	return db.RowExists(
		ctx,
		`SELECT c.company_id FROM company AS c WHERE c.company_id=?`,
		companyID,
	)
}

func (db *DB) DeleteCompany(ctx context.Context, companyID int64) (*models.Company, error) {
	_, err := db.sql.Exec(`UPDATE company SET is_deleted=? WHERE company_id=?`, true, companyID)
	if err != nil {
		return nil, err
	}

	return db.GetCompany(ctx, companyID)
}

func (db *DB) SaveCompany(ctx context.Context, model models.CompanyResult) (*models.Company, error) {
	if db.companyExists(ctx, model.CompanyID) {
		if _, err := db.sql.Exec(
			`UPDATE company SET name=?, bin=?, iin=?, iic=?, bic=?, address=?, phone_numbers=?, is_deleted=?, updated_ts=?
					WHERE company_id=?`,
			model.Name,
			model.Bin,
			model.Iin,
			model.Iic,
			model.Bic,
			model.Address,
			model.PhoneNumbers,
			model.IsDeleted,
			time.Now().Unix(),
			model.CompanyID,
		); err != nil {
			return nil, err
		}

		return db.GetCompany(ctx, model.CompanyID)
	} else {
		result, err := db.sql.Exec(
			`INSERT INTO company(name, bin, iin, iic, bic, address, phone_numbers, is_deleted, created_ts, updated_ts)
											VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			model.Name,
			model.Bin,
			model.Iin,
			model.Iic,
			model.Bic,
			model.Address,
			model.PhoneNumbers,
			model.IsDeleted,
			time.Now().Unix(),
			time.Now().Unix(),
		)
		if err != nil {
			return nil, err
		}

		lastID, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}

		return db.GetCompany(ctx, lastID)
	}
}
