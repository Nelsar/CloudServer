package models

type ActionLogResult struct {
	ActionID    int64  `json:"actionId"`
	Action      string `json:"action"`
	UserID      int64  `json:"userId"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	CompanyID   int64  `json:"companyId"`
	CompanyName string `json:"companyName"`
	Time        int64  `json:"time"`
}
