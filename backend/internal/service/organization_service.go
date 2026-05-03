// [bmai-fork] organization service — department path calculation + CRUD delegation
package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type OrganizationService struct {
	repo OrganizationRepository
}

func NewOrganizationService(repo OrganizationRepository) *OrganizationService {
	return &OrganizationService{repo: repo}
}

func (s *OrganizationService) GetByID(ctx context.Context, id int64) (*domain.Organization, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OrganizationService) GetByTenantKey(ctx context.Context, tenantKey string) (*domain.Organization, error) {
	return s.repo.GetByTenantKey(ctx, tenantKey)
}

func (s *OrganizationService) UpsertOrganization(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	return s.repo.UpsertOrganization(ctx, org)
}

func (s *OrganizationService) ListOrganizations(ctx context.Context) ([]*domain.Organization, error) {
	return s.repo.ListOrganizations(ctx)
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, id int64) error {
	return s.repo.DeleteOrganization(ctx, id)
}

func (s *OrganizationService) UpsertDepartmentWithPath(ctx context.Context, dept *domain.Department) (*domain.Department, error) {
	if dept.ParentID != nil {
		parent, err := s.repo.GetDepartmentByID(ctx, *dept.ParentID)
		if err != nil {
			return nil, err
		}
		if parent != nil {
			dept.FullPath = parent.FullPath + "/" + dept.Name
			dept.Level = parent.Level + 1
		}
	}
	if dept.FullPath == "" {
		dept.FullPath = dept.Name
		dept.Level = 1
	}
	return s.repo.UpsertDepartment(ctx, dept)
}

func (s *OrganizationService) GetDepartmentByID(ctx context.Context, id int64) (*domain.Department, error) {
	return s.repo.GetDepartmentByID(ctx, id)
}

func (s *OrganizationService) ListDepartmentTree(ctx context.Context, orgID int64) ([]*domain.Department, error) {
	return s.repo.ListDepartmentTree(ctx, orgID)
}

func (s *OrganizationService) DeleteDepartment(ctx context.Context, id int64) error {
	return s.repo.DeleteDepartment(ctx, id)
}

func (s *OrganizationService) AssignUserToDepartment(ctx context.Context, ud *domain.UserDepartment) error {
	if err := s.repo.AssignUserToDepartment(ctx, ud); err != nil {
		return err
	}
	if ud.IsPrimary {
		dept, err := s.repo.GetDepartmentByID(ctx, ud.DepartmentID)
		if err != nil {
			return err
		}
		if dept != nil {
			return s.repo.UpdateUserDenormalizedOrg(ctx, ud.UserID, &dept.OrganizationID, &dept.ID)
		}
	}
	return nil
}

func (s *OrganizationService) RemoveUserFromDepartment(ctx context.Context, userID, departmentID int64) error {
	return s.repo.RemoveUserFromDepartment(ctx, userID, departmentID)
}

func (s *OrganizationService) GetUserDepartments(ctx context.Context, userID int64) ([]*domain.Department, error) {
	return s.repo.GetUserDepartments(ctx, userID)
}

func (s *OrganizationService) GetDepartmentUsers(ctx context.Context, deptID int64, page, pageSize int) ([]*domain.UserDepartment, int64, error) {
	return s.repo.GetDepartmentUsers(ctx, deptID, page, pageSize)
}
