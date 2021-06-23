package models

type SensorResult struct {
	SensorId     string  `json:"sensorId"`
	TagName      string  `json:"tagName"`
	ControllerId string  `json:"controllerId"`
	Transform    string  `json:"transform"`
	RangeL       float32 `json:"rangeL"`
	RangeH       float32 `json:"rangeH"`
	AlarmValue   float32 `json:"alarmValue"`
	AlarmTime    int64   `json:"time"`
	AlarmL       float64 `json:"alarmL"`
	AlarmLL      float64 `json:"alarmLL"`
	AlarmH       float64 `json:"alarmH"`
	AlarmHH      float64 `json:"alarmHH"`
	Unit         string  `json:"unit"`
	IsEnabled    bool    `json:"isEnabled"`
	CreatedTs    int64   `json:"createdTs"`
	UpdatedTs    int64   `json:"updatedTs"`
}

type ControllerResult struct {
	ControllerID string          `json:"controllerId"`
	Name         string          `json:"name"`
	OilFieldId   int64           `json:"oilFieldId"`
	Model        string          `json:"model"`
	IsEnabled    int64           `json:"isEnabled"`
	CreatedTs    int64           `json:"createdTs"`
	UpdatedTs    int64           `json:"updatedTs"`
	Sensors      []*SensorResult `json:"sensors"`
}

type ControllerResultCloud struct {
	ControllerId int64  `json:"controllerId"`
	Name         string `json:"name"`
	SlaveId      int    `json:"slaveId"`
	Address      string `json:"address"`
	Port         string `json:"port"`
	Model        string `json:"model"`
	Type         string `json:"type"`
	IsEnabled    bool   `json:"isEnabled"`
	CreatedTs    int64  `json:"createdTs"`
	UpdatedTs    int64  `json:"updatedTs"`
	IsOnline     bool   `json:"isOnline"`
}

type SensorResultCloud struct {
	TagName      string  `json:"tagName"`
	ControllerID int64   `json:"controllerId"`
	Transform    string  `json:"transform"`
	Address      int     `json:"address"`
	Quantity     int     `json:"quantity"`
	ReadTemplate string  `json:"read_template"`
	ResultFormat string  `json:"result_format"`
	RangeL       float32 `json:"rangeL"`
	RangeH       float32 `json:"rangeH"`
	AlarmL       float32 `json:"alarmL"`
	AlarmLL      float32 `json:"alarmLL"`
	AlarmH       float32 `json:"alarmH"`
	AlarmHH      float32 `json:"alarmHH"`
	Unit         string  `json:"unit"`
	IsEnabled    bool    `json:"isEnabled"`
	SkipSave     bool    `json:"skip_save"`
	CreatedTs    int64   `json:"createdTs"`
	UpdatedTs    int64   `json:"updatedTs"`
	IsOnline     bool    `json:"isOnline"`
}
