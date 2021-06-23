package models

import (
	validation "github.com/go-ozzo/ozzo-validation"
	validation2 "gitlab.citicom.kz/CloudServer/server/utils/validation"
)

type MnemoResult struct {
	MnemoId   int64  `json:"mnemoId"`
	Name      string `json:"name"`
	CompanyId int64  `json:"companyId"`
	FileUrl   string `json:"fileUrl"`
	Info      string `json:"info"`
}

type MnemoSensorDataResult struct {
	SensorID       string  `json:"sensorId"`
	Unit           string  `json:"unit"`
	RangeH         string  `json:"rangeH"`
	RangeL         string  `json:"rangeL"`
	FormattedValue float64 `json:"formattedValue"`
	CreatedTs      int64   `json:"createdTs"`
}

func (mnemo *MnemoResult) Validate() error {
	return validation.ValidateStruct(
		mnemo,
		validation.Field(
			&mnemo.CompanyId,
			validation.Required,
			validation2.GreaterThanOrEqualCreate("company id greater than or equal 1", 1),
		),
		validation.Field(
			&mnemo.Name,
			validation.Required,
			validation.Length(5, 100).Error("Mnemo name length must be between 5 and 100"),
		),

	)
}
