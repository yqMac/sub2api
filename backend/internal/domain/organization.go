// [bmai-fork] organization domain model
package domain

import "time"

type Organization struct {
	ID        int64
	TenantKey string
	Name      string
	Type      string // feishu / manual / oidc
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type Department struct {
	ID             int64
	OrganizationID int64
	ParentID       *int64
	ExternalID     string
	Name           string
	FullPath       string
	Level          int
	Metadata       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type UserDepartment struct {
	UserID       int64
	DepartmentID int64
	IsPrimary    bool
	Role         string
	EmployeeID   string
	CreatedAt    time.Time
}
