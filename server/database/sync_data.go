package database

import (
	"context"
	"fmt"
	client "github.com/influxdata/influxdb1-client/v2"
	"gitlab.citicom.kz/CloudServer/server/influx"
	"gitlab.citicom.kz/CloudServer/server/models"
	"time"
)

func (db *DB) SynchronizeControllers(ctx context.Context, influxDB *influx.Influx, controllers []*models.CloudControllerResult, oilFieldID int64) {
	for _, controller := range controllers {
		primaryKey := getPrimaryKey(oilFieldID, controller.ControllerId)

		if !db.controllerExists(ctx, primaryKey) {
			_, err := db.sql.Exec(`INSERT INTO controllers(
									controller_id, 
									name,
									oil_field_id,
									slave_id,
									address,
									port,
									model,
									is_enabled,
									created_ts,
									updated_ts) 
									VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				primaryKey,
				controller.Name,
				oilFieldID,
				controller.SlaveId,
				controller.Address,
				controller.Port,
				controller.Model,
				controller.IsEnabled,
				controller.CreatedTs,
				controller.UpdatedTs,
			)
			if err != nil {
				fmt.Println("CONTROLLER INSERT ERROR: ", err)
				continue
			}
		} else {
			_, err := db.sql.Exec(`UPDATE controllers SET 
									name=?, 
									oil_field_id=?, 
									slave_id=?,
									address=?,
									port=?,
									model=?,
									is_enabled=?,
									created_ts=?,
									updated_ts=?
									WHERE controller_id=?`,
				controller.Name,
				oilFieldID,
				controller.SlaveId,
				controller.Address,
				controller.Port,
				controller.Model,
				controller.IsEnabled,
				controller.CreatedTs,
				controller.UpdatedTs,
				primaryKey,
			)
			if err != nil {
				fmt.Println("CONTROLLER UPDATE ERROR: ", err)
				continue
			}
		}

		for _, sensor := range controller.Sensors {
			sensorPrimaryKey := getPrimaryKey(primaryKey, sensor.TagName)

			if !db.sensorExists(ctx, sensorPrimaryKey) {
				_, err := db.sql.Exec(`INSERT INTO sensors(
										sensor_id,
										tag_name,
										controller_id,
										transform,
										address,
										range_l,
										range_h,
										alarm_l,
										alarm_ll,
										alarm_h,
										alarm_hh,
										unit,
										is_enabled,
										created_ts,
										updated_ts) 
										VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					sensorPrimaryKey,
					sensor.TagName,
					primaryKey,
					sensor.Transform,
					sensor.Address,
					sensor.RangeL,
					sensor.RangeH,
					sensor.AlarmL,
					sensor.AlarmLL,
					sensor.AlarmH,
					sensor.AlarmHH,
					sensor.Unit,
					sensor.IsEnabled,
					sensor.CreatedTs,
					sensor.UpdatedTs,
				)
				if err != nil {
					fmt.Println("SENSOR INSERT ERROR: ", err)
					continue
				}
			} else {
				_, err := db.sql.Exec(`UPDATE sensors SET 
										tag_name=?, 
										controller_id=?, 
										transform=?,
										address=?,
										range_l=?,
										range_h=?,
										alarm_l=?,
										alarm_ll=?,
										alarm_h=?,
										alarm_hh=?,
										unit=?,
										is_enabled=?,
										created_ts=?,
										updated_ts=?
										WHERE sensor_id=?`,
					sensor.TagName,
					primaryKey,
					sensor.Transform,
					sensor.Address,
					sensor.RangeL,
					sensor.RangeH,
					sensor.AlarmL,
					sensor.AlarmLL,
					sensor.AlarmH,
					sensor.AlarmHH,
					sensor.Unit,
					sensor.IsEnabled,
					sensor.CreatedTs,
					sensor.UpdatedTs,
					sensorPrimaryKey,
				)
				if err != nil {
					fmt.Println("SENSOR UPDATE ERROR: ", err)
					continue
				}
			}
		}
	}
}

func (db *DB) SynchronizeSensorData(ctx context.Context, influxDB *influx.Influx, sensorDatas []*models.SensorDataResult, oilFieldID int64) models.Alarms {
	//synchronizedIds := make([]int64, 0, 10)
	alarms := models.Alarms{
		Alarms: make([]*models.Alarm, 0, 10),
	}

	for _, sensorData := range sensorDatas {
		primaryKey := getPrimaryKey(oilFieldID, sensorData.ControllerId)
		sensorPrimaryKey := getPrimaryKey(primaryKey, sensorData.SensorTagName)
		//sensorDataPrimaryKey := getPrimaryKey(sensorPrimaryKey, sensorData.CreatedTs)
		//_, err := db.sql.Exec(`INSERT INTO sensor_data(
		//								sensor_data_id,
		//								sensor_id,
		//								raw_value,
		//								formatted_value,
		//								created_ts) VALUES(?, ?, ?, ?, ?)`,
		//	sensorDataPrimaryKey,
		//	sensorPrimaryKey,
		//	sensorData.RawValue,
		//	sensorData.FormattedValue,
		//	sensorData.CreatedTs,
		//)
		//if err != nil {
		//	//fmt.Println("SENSOR DATA INSERT ERROR: ", err)
		//	continue
		//}

		err := influxDB.SaveSensorData(sensorPrimaryKey, sensorData.FormattedValue, sensorData.CreatedTs)
		if err != nil {
			//fmt.Println("SENSOR DATA INSERT INFLUX ERROR: ", err)
			continue
		}

		//hasAlarm, alarmType, alarmValue := sensorData.HasAlarm(sensor)
		//if hasAlarm {
		//	alarm := &models.Alarm{
		//		OilFieldID:   oilFieldID,
		//		ControllerID: primaryKey,
		//		SensorID:     sensorPrimaryKey,
		//		AlarmType:    alarmType,
		//		AlarmValue:   alarmValue,
		//		Value:        sensorData.FormattedValue,
		//		Time:         sensorData.CreatedTs,
		//	}
		//	alarms.Add(alarm)
		//}
		//
		//synchronizedIds = append(synchronizedIds, sensorData.SensorDataId)
	}

	return alarms
}

func (db *DB) SynchronizeData(
	ctx context.Context,
	influxDB *influx.Influx,
	controllers []*models.CloudControllersResult,
	data []*models.SensorData,
	oilFieldID int64,
) models.Alarms {
	sensors := make([]*models.SensorResultCloud, 0, 10)
	alarms := models.Alarms{
		Alarms: make([]*models.Alarm, 0, 10),
	}

	for _, controller := range controllers {
		primaryKey := getPrimaryKey(oilFieldID, controller.ControllerId)

		if !db.controllerExists(ctx, primaryKey) {
			_, err := db.sql.Exec(`INSERT INTO controllers(
									controller_id, 
									name,
									oil_field_id,
									slave_id,
									address,
									port,
									model,
									is_enabled,
									created_ts,
									updated_ts) 
									VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				primaryKey,
				controller.Name,
				oilFieldID,
				controller.SlaveId,
				controller.Address,
				controller.Port,
				controller.Model,
				controller.IsEnabled,
				controller.CreatedTs,
				controller.UpdatedTs,
			)
			if err != nil {
				fmt.Println("CONTROLLER INSERT ERROR: ", err)
				continue
			}
		} else {
			_, err := db.sql.Exec(`UPDATE controllers SET 
									name=?, 
									oil_field_id=?, 
									slave_id=?,
									address=?,
									port=?,
									model=?,
									is_enabled=?,
									created_ts=?,
									updated_ts=?
									WHERE controller_id=?`,
				controller.Name,
				oilFieldID,
				controller.SlaveId,
				controller.Address,
				controller.Port,
				controller.Model,
				controller.IsEnabled,
				controller.CreatedTs,
				controller.UpdatedTs,
				primaryKey,
			)
			if err != nil {
				fmt.Println("CONTROLLER UPDATE ERROR: ", err)
				continue
			}
		}

		for _, sensor := range controller.Sensors {
			sensors = append(sensors, sensor)
			sensorPrimaryKey := getPrimaryKey(primaryKey, sensor.TagName)

			if !db.sensorExists(ctx, sensorPrimaryKey) {
				_, err := db.sql.Exec(`INSERT INTO sensors(
										sensor_id,
										tag_name,
										controller_id,
										transform,
										address,
										range_l,
										range_h,
										alarm_l,
										alarm_ll,
										alarm_h,
										alarm_hh,
										unit,
										is_enabled,
										created_ts,
										updated_ts) 
										VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					sensorPrimaryKey,
					sensor.TagName,
					primaryKey,
					sensor.Transform,
					sensor.Address,
					sensor.RangeL,
					sensor.RangeH,
					sensor.AlarmL,
					sensor.AlarmLL,
					sensor.AlarmH,
					sensor.AlarmHH,
					sensor.Unit,
					sensor.IsEnabled,
					sensor.CreatedTs,
					sensor.UpdatedTs,
				)
				if err != nil {
					fmt.Println("SENSOR INSERT ERROR: ", err)
					continue
				}
			} else {
				_, err := db.sql.Exec(`UPDATE sensors SET 
										tag_name=?, 
										controller_id=?, 
										transform=?,
										address=?,
										range_l=?,
										range_h=?,
										alarm_l=?,
										alarm_ll=?,
										alarm_h=?,
										alarm_hh=?,
										unit=?,
										is_enabled=?,
										created_ts=?,
										updated_ts=?
										WHERE sensor_id=?`,
					sensor.TagName,
					primaryKey,
					sensor.Transform,
					sensor.Address,
					sensor.RangeL,
					sensor.RangeH,
					sensor.AlarmL,
					sensor.AlarmLL,
					sensor.AlarmH,
					sensor.AlarmHH,
					sensor.Unit,
					sensor.IsEnabled,
					sensor.CreatedTs,
					sensor.UpdatedTs,
					sensorPrimaryKey,
				)
				if err != nil {
					fmt.Println("SENSOR UPDATE ERROR: ", err)
					continue
				}
			}
		}
	}

	points, err := influxDB.NewBatchPoints()
	if err != nil {
		fmt.Println("INFLUX POINTS ERROR: ", err)
		return alarms
	}

	for _, sensorData := range data {
		sensorId := findSensor(sensors, sensorData.SensorTagName)
		if sensorId == -1 {
			continue
		}

		sensor := sensors[sensorId]
		primaryKey := getPrimaryKey(oilFieldID, sensor.ControllerID)
		sensorPrimaryKey := getPrimaryKey(primaryKey, sensor.TagName)

		tags := map[string]string{
			"tagName": sensorPrimaryKey,
		}
		value := map[string]interface{} {
			"value": sensorData.FormattedValue,
		}
		createdTime := time.Unix(sensorData.CreatedTs, 0)
		point, err := client.NewPoint(
			"cloudData",
			tags,
			value,
			createdTime,
		)
		if err != nil {
			fmt.Println("ERROR creating bp: ", err)
			continue
		}
		points.AddPoint(point)

		hasAlarm, alarmType, alarmValue := sensorData.HasAlarm(sensor)
		if hasAlarm {
			alarm := &models.Alarm{
				OilFieldID:   oilFieldID,
				ControllerID: primaryKey,
				SensorID:     sensorPrimaryKey,
				AlarmType:    alarmType,
				AlarmValue:   alarmValue,
				Value:        sensorData.FormattedValue,
				Time:         sensorData.CreatedTs,
			}
			alarms.Add(alarm)
		}
	}

	err = influxDB.Write(points)
	if err != nil {
		fmt.Println("SAVE INFLUX ERROR: ", err)
	} else {
		fmt.Println("INFLUX SAVED")
	}

	return alarms
}

func findSensor(a []*models.SensorResultCloud, tagName string) int {
	for i, n := range a {
		if tagName == n.TagName {
			return i
		}
	}
	return -1
}

func (db *DB) controllerExists(ctx context.Context, pk string) bool {
	return db.RowExists(
		ctx,
		`SELECT c.controller_id FROM controllers AS c WHERE c.controller_id=?`,
		pk,
	)
}

func (db *DB) sensorExists(ctx context.Context, pk string) bool {
	return db.RowExists(
		ctx,
		`SELECT s.sensor_id FROM sensors AS s WHERE s.sensor_id=?`,
		pk,
	)
}

func getPrimaryKey(first interface{}, second interface{}) string {
	return fmt.Sprintf("%v_%v", first, second)
}
