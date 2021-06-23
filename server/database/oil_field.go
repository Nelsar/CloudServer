package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
)

func (db *DB) GetOilFields(ctx context.Context, companyID int64, all bool) ([]*models.OilField, error) {
	l, _ := icontext.GetLogger(ctx)
	query := `SELECT 
		oilF.oil_field_id,
		oilF.http_address,
		oilF.company_id,
		c.name,
		oilF.name,
		oilF.lat,
		oilF.lon,
		oilF.is_deleted,
		oilF.created_ts,
		oilF.updated_ts
		FROM oil_field AS oilF
		JOIN company c
		ON oilF.company_id=c.company_id`

	var rows *sql.Rows
	var err error

	if !all {
		query = fmt.Sprintf(`%s WHERE oilF.company_id=?`, query)
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
	oilFields := make([]*models.OilField, 0, 10)
	for rows.Next() {
		oilField := &models.OilField{}
		err := rows.Scan(
			&oilField.OilFieldId,
			&oilField.HttpAddress,
			&oilField.CompanyID,
			&oilField.CompanyName,
			&oilField.Name,
			&oilField.Lat,
			&oilField.Lon,
			&oilField.IsDeleted,
			&oilField.CreatedTs,
			&oilField.UpdatedTs,
		)
		if err != nil {
			continue
		}
		oilFields = append(oilFields, oilField)
	}

	return oilFields, nil
}

func (db *DB) GetOilField(ctx context.Context, oilFieldID int64) (*models.OilField, error) {
	l, _ := icontext.GetLogger(ctx)
	oilField := &models.OilField{}
	if err := db.sql.QueryRow(`SELECT 
		oilF.oil_field_id,
		oilF.http_address,
		oilF.company_id,
		oilF.name,
		oilF.lat,
		oilF.lon,
		oilF.is_deleted,
		oilF.created_ts,
		oilF.updated_ts
		FROM oil_field AS oilF 
		WHERE oilF.oil_field_id=?`, oilFieldID).Scan(
		&oilField.OilFieldId,
		&oilField.HttpAddress,
		&oilField.CompanyID,
		&oilField.Name,
		&oilField.Lat,
		&oilField.Lon,
		&oilField.IsDeleted,
		&oilField.CreatedTs,
		&oilField.UpdatedTs,
	); err != nil {
		l.Errorf("SELECT oilField ERROR: %f", err.Error())
		return nil, err
	}

	return oilField, nil
}

func (db *DB) GetOilFieldCheckingUser(ctx context.Context, companyID int64, all bool) ([]*models.OilField, error) {
	l, _ := icontext.GetLogger(ctx)
	query := `SELECT 
	oilF.oil_field_id,
	oilF.http_address,
	oilF.company_id,
	oilF.name,
	oilF.lat,
	oilF.lon,
	oilF.is_deleted,
	oilF.created_ts,
	oilF.updated_ts
	FROM oil_field AS oilF`

	var rows *sql.Rows
	var err error

	if !all {
		query = fmt.Sprintf(`%s WHERE oilF.company_id=?`, query)
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
	oilFields := make([]*models.OilField, 0, 10)
	for rows.Next() {
		oilField := &models.OilField{}
		err := rows.Scan(
			&oilField.OilFieldId,
			&oilField.HttpAddress,
			&oilField.CompanyID,
			&oilField.Name,
			&oilField.Lat,
			&oilField.Lon,
			&oilField.IsDeleted,
			&oilField.CreatedTs,
			&oilField.UpdatedTs,
		)

		if err != nil {
			continue
		}
		oilFields = append(oilFields, oilField)
	}

	return oilFields, nil
}

func (db *DB) oilFieldExists(ctx context.Context, oilFieldID int64) bool {
	return db.RowExists(
		ctx,
		`SELECT oilF.oil_field_id FROM oil_field AS oilF WHERE oilF.oil_field_id=?`,
		oilFieldID,
	)
}

func (db *DB) DeleteOilField(ctx context.Context, oilFieldID int64) (*models.OilField, error) {
	_, err := db.sql.Exec(`UPDATE oil_field SET is_deleted=? WHERE oil_field_id=?`, true, oilFieldID)
	if err != nil {
		return nil, err
	}

	return db.GetOilField(ctx, oilFieldID)
}

func (db *DB) SaveOilField(ctx context.Context, model models.OilFieldResult) (*models.OilField, error) {
	if db.oilFieldExists(ctx, model.OilFieldId) {
		if _, err := db.sql.Exec(
			`UPDATE oil_field SET http_address=?, company_id=?, name=?, lat=?, lon=?, is_deleted=?, created_ts=?, updated_ts=?
					WHERE oil_field_id=?`,
			model.HttpAddress,
			model.CompanyID,
			model.Name,
			model.Lat,
			model.Lon,
			model.IsDeleted,
			time.Now().Unix(),
			time.Now().Unix(),
			model.OilFieldId,
		); err != nil {
			return nil, err
		}

		return db.GetOilField(ctx, model.OilFieldId)
	} else {
		result, err := db.sql.Exec(
			`INSERT INTO oil_field(http_address, company_id, name, lat, lon, is_deleted, created_ts, updated_ts)
											VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			model.HttpAddress,
			model.CompanyID,
			model.Name,
			model.Lat,
			model.Lon,
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

		return db.GetOilField(ctx, lastID)
	}
}

func (db *DB) GetController(ctx context.Context, controllerID string) (*models.ControllerResult, error) {
	controller := &models.ControllerResult{}
	if err := db.sql.QueryRow(`SELECT 
			c.controller_id,
			c.name,
			c.oil_field_id,
			c.model,
			c.is_enabled,
			c.created_ts,
			c.updated_ts
			FROM controllers AS c WHERE c.controller_id=?
	`, controllerID).Scan(
		&controller.ControllerID,
		&controller.Name,
		&controller.OilFieldId,
		&controller.Model,
		&controller.IsEnabled,
		&controller.CreatedTs,
		&controller.UpdatedTs,
	); err != nil {
		return nil, err
	}

	return controller, nil
}

func (db *DB) GetControllers(ctx context.Context, oilFieldID int64) ([]*models.ControllerResult, error) {
	l, _ := icontext.GetLogger(ctx)
	rows, err := db.sql.Query(`SELECT 
			c.controller_id,
			c.name,
			c.oil_field_id,
			c.model,
			c.is_enabled,
			c.created_ts,
			c.updated_ts
			FROM controllers AS c WHERE c.oil_field_id=?
	`, oilFieldID)
	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("GetControllers error")
		return nil, err
	}
	defer rows.Close()
	controllers := make([]*models.ControllerResult, 0, 10)
	for rows.Next() {
		controller := &models.ControllerResult{}
		err := rows.Scan(
			&controller.ControllerID,
			&controller.Name,
			&controller.OilFieldId,
			&controller.Model,
			&controller.IsEnabled,
			&controller.CreatedTs,
			&controller.UpdatedTs,
		)
		if err != nil {
			continue
		}
		sensors, err := db.GetSensors(ctx, controller.ControllerID)
		if err == nil {
			controller.Sensors = sensors
		}
		controllers = append(controllers, controller)
	}

	return controllers, nil
}

func (db *DB) GetControllersArrayInputParam(ctx context.Context, oilFieldID []int64, sensorID string) ([]*models.ControllerResult, error) {
	l, _ := icontext.GetLogger(ctx)
	oilFieldIDs := []string{}

	for _, value := range oilFieldID {
		oilFieldIDs = append(oilFieldIDs, fmt.Sprintf("%d", value))
	}

	Ids := strings.Join(oilFieldIDs, ",")
	rows, err := db.sql.Query(`SELECT 
			c.controller_id,
			c.name,
			c.oil_field_id,
			c.model,
			c.is_enabled,
			c.created_ts,
			c.updated_ts
			FROM controllers AS c WHERE c.oil_field_id IN(?)`, Ids)
	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("GetControllers error")
		return nil, err
	}
	defer rows.Close()
	controllers := make([]*models.ControllerResult, 0, 10)
	for rows.Next() {
		controller := &models.ControllerResult{}
		err := rows.Scan(
			&controller.ControllerID,
			&controller.Name,
			&controller.OilFieldId,
			&controller.Model,
			&controller.IsEnabled,
			&controller.CreatedTs,
			&controller.UpdatedTs,
		)
		if err != nil {
			continue
		}
		var (
			sensors = []*models.SensorResult{}
		)

		if sensorID == "" {
			sensors, err = db.GetSensors(ctx, controller.ControllerID)
		}

		if sensorID != "" {
			sensors, err = db.GetSensor(ctx, controller.ControllerID, sensorID)
		}

		if err == nil {
			controller.Sensors = sensors
		}
		controllers = append(controllers, controller)
	}

	return controllers, nil
}

func (db *DB) GetSensors(ctx context.Context, controllerID string) ([]*models.SensorResult, error) {
	l, _ := icontext.GetLogger(ctx)
	rows, err := db.sql.Query(`SELECT 
		s.sensor_id,
		s.tag_name,
		s.controller_id,
		s.transform,
		s.range_l,
		s.range_h,
		a.value,
		a.time,
		s.alarm_l,
		s.alarm_ll,
		s.alarm_h,
		s.alarm_hh,
		s.unit,
		s.is_enabled,
		s.created_ts,
		s.updated_ts 
		FROM sensors AS s 
		LEFT JOIN alarms a ON s.sensor_id=a.sensor_id
		WHERE s.controller_id=?`, controllerID)
	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("GetSensors error")
		return nil, err
	}
	defer rows.Close()
	sensors := make([]*models.SensorResult, 0, 10)
	for rows.Next() {
		sensor := &models.SensorResult{}
		err := rows.Scan(
			&sensor.SensorId,
			&sensor.TagName,
			&sensor.ControllerId,
			&sensor.Transform,
			&sensor.RangeL,
			&sensor.RangeH,
			&sensor.AlarmValue,
			&sensor.AlarmTime,
			&sensor.AlarmL,
			&sensor.AlarmLL,
			&sensor.AlarmH,
			&sensor.AlarmHH,
			&sensor.Unit,
			&sensor.IsEnabled,
			&sensor.CreatedTs,
			&sensor.UpdatedTs,
		)
		if err != nil {
			fmt.Println("SENSOR", err)
			continue
		}
		sensors = append(sensors, sensor)
	}

	return sensors, nil
}
func (db *DB) GetSensor(ctx context.Context, controllerID string, sensorID string) ([]*models.SensorResult, error) {
	l, _ := icontext.GetLogger(ctx)
	rows, err := db.sql.Query(`SELECT 
		s.sensor_id,
		s.tag_name,
		s.controller_id,
		s.transform,
		s.range_l,
		s.range_h,
		a.value,
		a.time,
		s.alarm_l,
		s.alarm_ll,
		s.alarm_h,
		s.alarm_hh,
		s.unit,
		s.is_enabled,
		s.created_ts,
		s.updated_ts 
		FROM sensors AS s 
		LEFT JOIN alarms a ON s.sensor_id=a.sensor_id
		WHERE s.controller_id=? AND s.sensor_id`, controllerID, sensorID)
	if err != nil {
		l.WithFields(log.Fields{
			"Error": err,
		}).Error("GetSensors error")
		return nil, err
	}
	defer rows.Close()
	sensors := make([]*models.SensorResult, 0, 10)
	for rows.Next() {
		sensor := &models.SensorResult{}
		err := rows.Scan(
			&sensor.SensorId,
			&sensor.TagName,
			&sensor.ControllerId,
			&sensor.Transform,
			&sensor.RangeL,
			&sensor.RangeH,
			&sensor.AlarmValue,
			&sensor.AlarmTime,
			&sensor.AlarmL,
			&sensor.AlarmLL,
			&sensor.AlarmH,
			&sensor.AlarmHH,
			&sensor.Unit,
			&sensor.IsEnabled,
			&sensor.CreatedTs,
			&sensor.UpdatedTs,
		)
		if err != nil {
			fmt.Println("SENSOR", err)
			continue
		}
		sensors = append(sensors, sensor)
	}

	return sensors, nil
}
