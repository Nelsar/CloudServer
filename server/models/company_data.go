package models

type CompanyData struct {
	CompanyID int64  `json:"company_id"`
	Value     string `json:"value"`
	DataType  string `json:"data_type"`
}
