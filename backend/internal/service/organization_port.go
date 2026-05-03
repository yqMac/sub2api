// [bmai-fork] organization repository interface
package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type OrganizationRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Organization, error)
	GetByTenantKey(ctx context.Context, tenantKey string) (*domain.Organization, error)
	UpsertOrganization(ctx context.Context, org *domain.Organization) (*domain.Organization, error)
	ListOrganizations(ctx context.Context) ([]*domain.Organization, error)
	DeleteOrganization(ctx context.Context, id int64) error

	GetDepartmentByID(ctx context.Context, id int64) (*domain.Department, error)
	UpsertDepartment(ctx context.Context, dept *domain.Department) (*domain.Department, error)
	ListDepartmentTree(ctx context.Context, orgID int64) ([]*domain.Department, error)
	DeleteDepartment(ctx context.Context, id int64) error

	AssignUserToDepartment(ctx context.Context, ud *domain.UserDepartment) error
	RemoveUserFromDepartment(ctx context.Context, userID, departmentID int64) error
	GetUserDepartments(ctx context.Context, userID int64) ([]*domain.Department, error)
	GetDepartmentUsers(ctx context.Context, deptID int64, page, pageSize int) ([]*domain.UserDepartment, int64, error)
	UpdateUserDenormalizedOrg(ctx context.Context, userID int64, orgID *int64, primaryDeptID *int64) error
	GetUserOrgInfo(ctx context.Context, userID int64) (orgID *int64, deptID *int64, deptPath string, err error)
}
