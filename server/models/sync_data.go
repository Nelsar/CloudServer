package models

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"regexp"
)

type SensorData struct {
	SensorDataId   int64   `json:"sensorDataId"`
	SensorTagName  string  `json:"sensorTagName"`
	RawValue       []byte  `json:"rawValue"`
	FormattedValue float32 `json:"formattedValue"`
	CreatedTs      int64   `json:"createdTs"`
}

type SensorDataResult struct {
	SensorData
	ControllerId int64  `json:controllerId`
	TagName      string `json:sensorTagName`
}

func (sensorData *SensorData) HasAlarm(sensor *SensorResultCloud) (bool, string, float32) {
	if sensorData.FormattedValue <= sensor.AlarmLL {
		return true, ALARM_TYPE_LOW_LOW, sensor.AlarmLL
	}
	if sensorData.FormattedValue <= sensor.AlarmL {
		return true, ALARM_TYPE_LOW, sensor.AlarmL
	}
	if sensorData.FormattedValue >= sensor.AlarmHH {
		return true, ALARM_TYPE_HIGHT_HIGHT, sensor.AlarmHH
	}
	if sensorData.FormattedValue >= sensor.AlarmH {
		return true, ALARM_TYPE_HIGHT, sensor.AlarmH
	}
	return false, "", 0
}

type CloudSensorResult struct {
	TagName      string        `json:"tagName"`
	ControllerID int64         `json:"controllerId"`
	Transform    string        `json:"transform"`
	Address      int           `json:"address"`
	RangeL       float32       `json:"rangeL"`
	RangeH       float32       `json:"rangeH"`
	AlarmL       float32       `json:"alarmL"`
	AlarmLL      float32       `json:"alarmLL"`
	AlarmH       float32       `json:"alarmH"`
	AlarmHH      float32       `json:"alarmHH"`
	Unit         string        `json:"unit"`
	IsEnabled    bool          `json:"isEnabled"`
	CreatedTs    int64         `json:"createdTs"`
	UpdatedTs    int64         `json:"updatedTs"`
	IsOnline     bool          `json:"isOnline"`
	Data         []*SensorData `json:"data"`
}

type CloudControllerResult struct {
	ControllerId int64                `json:"controllerId"`
	Name         string               `json:"name"`
	SlaveId      int                  `json:"slaveId"`
	Address      string               `json:"address"`
	Port         string               `json:"port"`
	Model        string               `json:"model"`
	IsEnabled    bool                 `json:"isEnabled"`
	CreatedTs    int64                `json:"createdTs"`
	UpdatedTs    int64                `json:"updatedTs"`
	IsOnline     bool                 `json:"isOnline"`
	Sensors      []*CloudSensorResult `json:"sensors"`
}

type CloudControllersResult struct {
	ControllerResultCloud
	Sensors []*SensorResultCloud `json:"sensors"`
}

type SyncControllerDataRequest struct {
	ControllerID string `json:"controllerId"`
	SelectTime   string `json:"selectTime"`
	DiffTime     string `json:"diffTime"`
	GroupTime    string `json:"groupTime"`
}

type CloudDataAck struct {
	SensorTagName string `json:"sensorTagName"`
	MaxID         int64  `json:"maxId"`
}

type CloudGzipData struct {
	Controllers []*CloudControllersResult `json:"controllers"`
	Data        []*SensorData             `json:"data"`
}

func (scdr *SyncControllerDataRequest) Validate() error {
	return validation.ValidateStruct(
		scdr,
		validation.Field(
			&scdr.ControllerID,
			validation.Required,
		),
		validation.Field(
			&scdr.SelectTime,
			validation.Required,
			validation.Match(regexp.MustCompile("^[0-9]{0,}[dhms]$")).Error("allowed formats 1d, 1h, 1m, 1s"),
		),
		validation.Field(
			&scdr.GroupTime,
			validation.Required,
			validation.Match(regexp.MustCompile("^[0-9]{0,}[dhms]$")).Error("allowed formats 1d, 1h, 1m, 1s"),
		),
		validation.Field(
			&scdr.DiffTime,
			validation.Match(regexp.MustCompile("^[0-9]{0,}[dhms]$")).Error("allowed formats 1d, 1h, 1m, 1s"),
		),
	)
}
