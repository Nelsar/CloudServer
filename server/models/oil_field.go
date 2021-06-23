package models

import validation "github.com/go-ozzo/ozzo-validation"

type OilField struct {
	OilFieldId  int64   `json:"oilFieldId"`
	HttpAddress string  `json:"httpAddress"`
	CompanyID   int64   `json:"companyId"`
	CompanyName string  `json:"companyName"`
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	IsDeleted   bool    `json:"isDeleted"`
	CreatedTs   int64   `json:"createdTs"`
	UpdatedTs   int64   `json:"updatedTs"`
}

type OilFieldResult struct {
	OilFieldId  int64   `json:"oilFieldId"`
	HttpAddress string  `json:"httpAddress"`
	CompanyID   int64   `json:"companyId"`
	CompanyName string  `json:"companyName"`
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	IsDeleted   bool    `json:"isDeleted"`
	CreatedTs   int64   `json:"createdTs"`
	UpdatedTs   int64   `json:"updatedTs"`
	IsOnline    bool    `json:"isOnline"`
}

func (ofr *OilFieldResult) Validate() error {
	return validation.ValidateStruct(
		ofr,
		validation.Field(
			&ofr.HttpAddress,
			validation.Required,
		),
		validation.Field(
			&ofr.Name,
			validation.Required,
		),
		validation.Field(
			&ofr.Lat,
			validation.Required,
		),
		validation.Field(
			&ofr.Lon,
			validation.Required,
		),
	)
}
