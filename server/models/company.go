package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"regexp"
)

type Company struct {
	CompanyID    int64  `json:"companyId"`
	Name         string `json:"name"`
	Bin          string `json:"bin"`
	Iin          string `json:"iin"`
	Iic          string `json:"iic"`
	Bic          string `json:"bic"`
	Address      string `json:"address"`
	PhoneNumbers string `json:"phoneNumbers"`
	IsDeleted    bool   `json:"isDeleted"`
	CreatedTs    int64  `json:"createdTs"`
	UpdatedTs    int64  `json:"updatedTs"`
}

type CompanyResult struct {
	Company
}

func (company *CompanyResult) Validate() error {
	return validation.ValidateStruct(
		company,
		validation.Field(
			&company.Name,
			validation.Required,
		),
		validation.Field(
			&company.Bin,
			validation.Required,
		),
		validation.Field(
			&company.Iin,
			validation.Required,
		),
		validation.Field(
			&company.Iic,
			validation.Required,
		),
		validation.Field(
			&company.Bic,
			validation.Required,
		),
		validation.Field(
			&company.Address,
			validation.Required,
		),
		validation.Field(
			&company.PhoneNumbers,
			validation.Required,
			validation.Match(regexp.MustCompile("^((8|\\+7)[- ]?)?((\\?d{3})?[- ]?)?[\\d- ]{7,10}$")).Error(
				"Unknown phone format (+77471111111)",
				),
		),
	)
}
