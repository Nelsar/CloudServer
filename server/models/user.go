package models

import (
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation"
)

const (
	RoleSuper    = "super"
	RoleAdmin    = "admin"
	RoleManager  = "manager"
	RoleOperator = "operator"
	RoleGuest    = "guest"
)

type User struct {
	UserID      int64  `json:"userId"`
	CompanyID   int64  `json:"companyId"`
	CompanyName string `json:"companyName"`
	Email       string `json:"email"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Token       string `json:"token"`
	Role        string `json:"role"`
	IsDeleted   bool   `json:"isDeleted"`
	CreatedTs   int64  `json:"createdTs"`
	UpdatedTs   int64  `json:"updatedTs"`
}

type UserResult struct {
	UserID      int64      `json:"userId"`
	CompanyID   int64      `json:"companyId"`
	CompanyName string     `json:"companyName"`
	Email       string     `json:"email"`
	FirstName   string     `json:"firstName"`
	LastName    string     `json:"lastName"`
	Role        RoleResult `json:"role"`
	IsDeleted   bool       `json:"isDeleted"`
	IsOnline    bool       `json:"isOnline"`
	CreatedTs   int64      `json:"createdTs"`
	UpdatedTs   int64      `json:"updatedTs"`
}

type CreateUser struct {
	CompanyID int64  `json:"companyId"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Role      string `json:"role"`
	Password  string `json:"password"`
}

type UpdateUser struct {
	UserID    int64  `json:"userId"`
	CompanyID int64  `json:"companyId"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Role      string `json:"role"`
	IsDeleted bool   `json:"isDeleted"`
}

type UserPassword struct {
	UserID   int64  `json:"userId"`
	Password string `json:"password"`
}

func (user *CreateUser) Validate() error {
	return validation.ValidateStruct(
		user,
		validation.Field(
			&user.CompanyID,
			validation.Required,
		),
		validation.Field(
			&user.Email,
			validation.Required,
			validation.Match(regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")).Error(
				"Email format error",
			),
		),
		validation.Field(
			&user.FirstName,
			validation.Required,
		),
		validation.Field(
			&user.LastName,
			validation.Required,
		),
		validation.Field(
			&user.Role,
			validation.Required,
			validation.Match(regexp.MustCompile("^(admin|manager|operator)$")).Error("allow types admin, manager, operator"),
		),
		validation.Field(
			&user.Password,
			validation.Required,
		),
	)
}

func (user *UpdateUser) Validate() error {
	return validation.ValidateStruct(
		user,
		validation.Field(
			&user.UserID,
			validation.Required,
		),
		validation.Field(
			&user.CompanyID,
			validation.Required,
		),
		validation.Field(
			&user.Email,
			validation.Required,
			validation.Match(regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")).Error(
				"Email format error",
			),
		),
		validation.Field(
			&user.FirstName,
			validation.Required,
		),
		validation.Field(
			&user.LastName,
			validation.Required,
		),
		validation.Field(
			&user.Role,
			validation.Required,
			validation.Match(regexp.MustCompile("^(admin|manager|operator)$")).Error("allow types admin, manager, operator"),
		),
	)
}

func (user *UserPassword) Validate() error {
	return validation.ValidateStruct(
		user,
		validation.Field(
			&user.UserID,
			validation.Required,
		),
		validation.Field(
			&user.Password,
			validation.Required,
		),
	)
}

type RoleResult struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func (user *User) IsSuperUser() bool {
	return user.Role == superRole.Name
}

func (user *User) IsAdminUser() bool {
	return user.Role == adminRole.Name
}

func (user *User) GetResultRole() RoleResult {
	role := user.GetRole()
	return RoleResult{
		user.Role,
		role.GetPermissions(make([]string, 0, 10)),
	}
}

func (user *User) GetRole() Role {
	if user.Role == RoleSuper {
		return superRole
	} else if user.Role == RoleAdmin {
		return adminRole
	} else if user.Role == RoleManager {
		return managerRole
	} else if user.Role == RoleOperator {
		return operatorRole
	}

	return guestRole
}

func (role *Role) GetPermissions(permissions []string) []string {
	for _, allowedPage := range role.Permissions {
		permissions = append(permissions, allowedPage)
	}
	if role.Child != nil {
		role.Child.GetPermissions(permissions)
	}

	return permissions
}

func (role *Role) Can(path string) bool {
	for _, allowedPage := range role.Permissions {
		if strings.HasPrefix(path, allowedPage) {
			return true
		}
	}

	if role.Child != nil {
		return role.Child.Can(path)
	}

	return false
}

type Role struct {
	Name        string
	Child       *Role
	Permissions []string
}

var superRole = Role{
	Name: RoleSuper,
	Permissions: []string{
		"/",
	},
}
var adminRole = Role{
	Name:  RoleAdmin,
	Child: &managerRole,
	Permissions: []string{
		"/users",
	},
}
var managerRole = Role{
	Name:  RoleManager,
	Child: &operatorRole,
	Permissions: []string{
		"/users/list",
		"/oil_fields",
		"/sensors/list",
		"/mnemoschemes",
		"/pages",
	},
}
var operatorRole = Role{
	Name:  RoleOperator,
	Child: &guestRole,
	Permissions: []string{
		"/companies/list",
		"/oil_fields/list",
		"/controllers/list",
		"/controllers/data",
		"/mnemoschemes/data",
		"/mnemoschemes/list",
		"/alarms",
		"/files",
		"/connect",
		"/companyData",
	},
}
var guestRole = Role{
	Name:        RoleGuest,
	Permissions: []string{},
}
