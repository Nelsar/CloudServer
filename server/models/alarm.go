package models

const (
	ALARM_TYPE_LOW_LOW = "ALARM_TYPE_LOW_LOW"
	ALARM_TYPE_LOW     = "ALARM_TYPE_LOW"

	ALARM_TYPE_HIGHT       = "ALARM_TYPE_HIGHT"
	ALARM_TYPE_HIGHT_HIGHT = "ALARM_TYPE_HIGHT_HIGHT"
)

type AlarmResult struct {
	AlarmID      int64   `json:"alarmId"`
	UserID       int64   `json:"userId"`
	OilFieldId   int64   `json:"oilFieldId"`
	ControllerID string  `json:"controllerId"`
	OilFieldName string  `json:"oilFieldName"`
	ControllerName string  `json:"controllerName"`
	SensorID     string  `json:"sensorId"`
	AlarmType    string  `json:"alarmType"`
	AlarmValue   float32 `json:"alarmValue"`
	Value        float32 `json:"value"`
	Time         int64   `json:"time"`
	IsViewed     bool    `json:"isViewed"`
}

type Alarm struct {
	OilFieldID   int64   `json:"oil_field_id"`
	ControllerID string  `json:"controller_id"`
	SensorID     string  `json:"sensor_id"`
	AlarmType    string  `json:"alarm_type"`
	AlarmValue   float32 `json:"alarm_value"`
	Value        float32 `json:"value"`
	Time         int64   `json:"time"`
}

type Alarms struct {
	Alarms []*Alarm
}

func (alarms *Alarms) Size() int {
	return len(alarms.Alarms)
}

func (alarms *Alarms) Add(alarm *Alarm) {
	needAdd := true
	for _, childAlarm := range alarms.Alarms {
		if childAlarm.OilFieldID == alarm.OilFieldID &&
			childAlarm.ControllerID == alarm.ControllerID &&
			childAlarm.SensorID == alarm.SensorID &&
			childAlarm.AlarmType == alarm.AlarmType {
			needAdd = false
			break
		}
	}
	if needAdd {
		alarms.Alarms = append(alarms.Alarms, alarm)
	}
}
