// [bmai-fork] organization repository — raw SQL on *sql.DB
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type organizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) service.OrganizationRepository {
	return &organizationRepository{db: db}
}

func (r *organizationRepository) GetByID(ctx context.Context, id int64) (*domain.Organization, error) {
	const q = `SELECT id, tenant_key, name, type, metadata, created_at, updated_at FROM organizations WHERE id = $1 AND deleted_at IS NULL`
	org := &domain.Organization{}
	var metaJSON []byte
	err := r.db.QueryRowContext(ctx, q, id).Scan(&org.ID, &org.TenantKey, &org.Name, &org.Type, &metaJSON, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err == nil && len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &org.Metadata)
	}
	return org, err
}

func (r *organizationRepository) GetByTenantKey(ctx context.Context, tenantKey string) (*domain.Organization, error) {
	const q = `SELECT id, tenant_key, name, type, metadata, created_at, updated_at FROM organizations WHERE tenant_key = $1 AND deleted_at IS NULL`
	org := &domain.Organization{}
	var metaJSON []byte
	err := r.db.QueryRowContext(ctx, q, tenantKey).Scan(&org.ID, &org.TenantKey, &org.Name, &org.Type, &metaJSON, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err == nil && len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &org.Metadata)
	}
	return org, err
}

func (r *organizationRepository) UpsertOrganization(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	metaJSON, _ := json.Marshal(org.Metadata)
	if org.Metadata == nil {
		metaJSON = nil
	}
	const q = `INSERT INTO organizations (tenant_key, name, type, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (tenant_key) DO UPDATE SET name = EXCLUDED.name, type = EXCLUDED.type, metadata = EXCLUDED.metadata, updated_at = NOW()
		RETURNING id, tenant_key, name, type, metadata, created_at, updated_at`
	result := &domain.Organization{}
	var resultMeta []byte
	err := r.db.QueryRowContext(ctx, q, org.TenantKey, org.Name, org.Type, metaJSON).Scan(
		&result.ID, &result.TenantKey, &result.Name, &result.Type, &resultMeta, &result.CreatedAt, &result.UpdatedAt,
	)
	if err == nil && len(resultMeta) > 0 {
		_ = json.Unmarshal(resultMeta, &result.Metadata)
	}
	return result, err
}

func (r *organizationRepository) ListOrganizations(ctx context.Context) ([]*domain.Organization, error) {
	const q = `SELECT id, tenant_key, name, type, metadata, created_at, updated_at FROM organizations WHERE deleted_at IS NULL ORDER BY id`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orgs []*domain.Organization
	for rows.Next() {
		o := &domain.Organization{}
		var metaJSON []byte
		if err := rows.Scan(&o.ID, &o.TenantKey, &o.Name, &o.Type, &metaJSON, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &o.Metadata)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (r *organizationRepository) DeleteOrganization(ctx context.Context, id int64) error {
	const q = `UPDATE organizations SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *organizationRepository) GetDepartmentByID(ctx context.Context, id int64) (*domain.Department, error) {
	const q = `SELECT id, organization_id, parent_id, external_id, name, full_path, level, created_at, updated_at
		FROM departments WHERE id = $1 AND deleted_at IS NULL`
	d := &domain.Department{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&d.ID, &d.OrganizationID, &d.ParentID, &d.ExternalID, &d.Name, &d.FullPath, &d.Level, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (r *organizationRepository) UpsertDepartment(ctx context.Context, dept *domain.Department) (*domain.Department, error) {
	const q = `INSERT INTO departments (organization_id, parent_id, external_id, name, full_path, level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (organization_id, external_id) DO UPDATE SET
			parent_id = EXCLUDED.parent_id, name = EXCLUDED.name,
			full_path = EXCLUDED.full_path, level = EXCLUDED.level, updated_at = NOW()
		RETURNING id, organization_id, parent_id, external_id, name, full_path, level, created_at, updated_at`
	result := &domain.Department{}
	err := r.db.QueryRowContext(ctx, q,
		dept.OrganizationID, dept.ParentID, dept.ExternalID, dept.Name, dept.FullPath, dept.Level,
	).Scan(
		&result.ID, &result.OrganizationID, &result.ParentID, &result.ExternalID,
		&result.Name, &result.FullPath, &result.Level, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *organizationRepository) ListDepartmentTree(ctx context.Context, orgID int64) ([]*domain.Department, error) {
	const q = `SELECT id, organization_id, parent_id, external_id, name, full_path, level, created_at, updated_at
		FROM departments WHERE organization_id = $1 AND deleted_at IS NULL ORDER BY level, name`
	rows, err := r.db.QueryContext(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var depts []*domain.Department
	for rows.Next() {
		d := &domain.Department{}
		if err := rows.Scan(&d.ID, &d.OrganizationID, &d.ParentID, &d.ExternalID, &d.Name, &d.FullPath, &d.Level, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		depts = append(depts, d)
	}
	return depts, rows.Err()
}

func (r *organizationRepository) DeleteDepartment(ctx context.Context, id int64) error {
	const q = `UPDATE departments SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *organizationRepository) AssignUserToDepartment(ctx context.Context, ud *domain.UserDepartment) error {
	const q = `INSERT INTO user_departments (user_id, department_id, is_primary, role, employee_id, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, department_id) DO UPDATE SET
			is_primary = EXCLUDED.is_primary, role = EXCLUDED.role, employee_id = EXCLUDED.employee_id`
	_, err := r.db.ExecContext(ctx, q, ud.UserID, ud.DepartmentID, ud.IsPrimary, ud.Role, ud.EmployeeID)
	return err
}

func (r *organizationRepository) RemoveUserFromDepartment(ctx context.Context, userID, departmentID int64) error {
	const q = `DELETE FROM user_departments WHERE user_id = $1 AND department_id = $2`
	_, err := r.db.ExecContext(ctx, q, userID, departmentID)
	return err
}

func (r *organizationRepository) GetUserDepartments(ctx context.Context, userID int64) ([]*domain.Department, error) {
	const q = `SELECT d.id, d.organization_id, d.parent_id, d.external_id, d.name, d.full_path, d.level, d.created_at, d.updated_at
		FROM departments d JOIN user_departments ud ON d.id = ud.department_id
		WHERE ud.user_id = $1 AND d.deleted_at IS NULL ORDER BY ud.is_primary DESC, d.level`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var depts []*domain.Department
	for rows.Next() {
		d := &domain.Department{}
		if err := rows.Scan(&d.ID, &d.OrganizationID, &d.ParentID, &d.ExternalID, &d.Name, &d.FullPath, &d.Level, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		depts = append(depts, d)
	}
	return depts, rows.Err()
}

func (r *organizationRepository) GetDepartmentUsers(ctx context.Context, deptID int64, page, pageSize int) ([]*domain.UserDepartment, int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM user_departments WHERE department_id = $1`, deptID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	const q = `SELECT user_id, department_id, is_primary, COALESCE(role,''), COALESCE(employee_id,''), created_at
		FROM user_departments WHERE department_id = $1 ORDER BY created_at LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, q, deptID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var uds []*domain.UserDepartment
	for rows.Next() {
		ud := &domain.UserDepartment{}
		if err := rows.Scan(&ud.UserID, &ud.DepartmentID, &ud.IsPrimary, &ud.Role, &ud.EmployeeID, &ud.CreatedAt); err != nil {
			return nil, 0, err
		}
		uds = append(uds, ud)
	}
	return uds, total, rows.Err()
}

func (r *organizationRepository) UpdateUserDenormalizedOrg(ctx context.Context, userID int64, orgID *int64, primaryDeptID *int64) error {
	const q = `UPDATE users SET organization_id = $1, primary_department_id = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, q, orgID, primaryDeptID, userID)
	return err
}

// [bmai-fork] GetUserOrgInfo loads the user's organization_id, primary_department_id, and full_path.
// Used by audit log writer to denormalize org dimensions onto each row.
// Returns (nil, nil, "", nil) when user has no org assignment.
func (r *organizationRepository) GetUserOrgInfo(ctx context.Context, userID int64) (*int64, *int64, string, error) {
	const q = `
		SELECT u.organization_id, u.primary_department_id, COALESCE(d.full_path, '')
		FROM users u
		LEFT JOIN departments d ON d.id = u.primary_department_id AND d.deleted_at IS NULL
		WHERE u.id = $1
	`
	var orgID, deptID *int64
	var deptPath string
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&orgID, &deptID, &deptPath)
	if err == sql.ErrNoRows {
		return nil, nil, "", nil
	}
	return orgID, deptID, deptPath, err
}

// suppress unused import warnings
var (
	_ = fmt.Sprintf
	_ = time.Now
	_ = pq.Array
)
