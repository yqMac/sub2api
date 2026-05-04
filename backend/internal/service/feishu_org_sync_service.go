// [bmai-fork] Feishu organization sync service
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type FeishuOrgSyncService struct {
	orgRepo  OrganizationRepository
	userRepo UserRepository
	log      *slog.Logger
}

func NewFeishuOrgSyncService(orgRepo OrganizationRepository, userRepo UserRepository) *FeishuOrgSyncService {
	return &FeishuOrgSyncService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
		log:      slog.Default().With("component", "feishu_sync"),
	}
}

type SyncResult struct {
	DepartmentsSynced int      `json:"departments_synced"`
	UsersSynced       int      `json:"users_synced"`
	UsersMatched      int      `json:"users_matched"`
	UsersUnmatched    int      `json:"users_unmatched"`
	Errors            []string `json:"errors,omitempty"`
}

// TestConnection validates credentials by getting a tenant token and counting departments.
func (s *FeishuOrgSyncService) TestConnection(ctx context.Context, appID, appSecret string) (int, error) {
	client := NewFeishuContactClient(appID, appSecret)
	token, err := client.GetTenantAccessToken(ctx)
	if err != nil {
		return 0, err
	}
	depts, err := client.ListDepartments(ctx, token)
	if err != nil {
		return 0, err
	}
	return len(depts), nil
}

// SyncOrganization performs a full sync of departments and users from Feishu.
func (s *FeishuOrgSyncService) SyncOrganization(ctx context.Context, org *domain.Organization, appID, appSecret string) (*SyncResult, error) {
	result := &SyncResult{}

	client := NewFeishuContactClient(appID, appSecret)
	token, err := client.GetTenantAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	// 1. Sync departments
	feishuDepts, err := client.ListDepartments(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}

	deptMap := make(map[string]int64) // feishu dept_id -> our dept.ID
	for _, fd := range feishuDepts {
		dept := &domain.Department{
			OrganizationID: org.ID,
			ExternalID:     fd.DepartmentID,
			Name:           fd.Name,
		}
		// Resolve parent
		if fd.ParentDepartmentID != "" && fd.ParentDepartmentID != "0" {
			if parentID, ok := deptMap[fd.ParentDepartmentID]; ok {
				dept.ParentID = &parentID
			}
		}
		// Calculate path
		dept.FullPath = s.buildFullPath(fd, feishuDepts)
		dept.Level = strings.Count(dept.FullPath, "/") + 1

		saved, err := s.orgRepo.UpsertDepartment(ctx, dept)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("dept %s: %v", fd.Name, err))
			continue
		}
		deptMap[fd.DepartmentID] = saved.ID
		result.DepartmentsSynced++
	}

	// 2. Sync users per department
	seenUsers := make(map[string]bool)
	for _, fd := range feishuDepts {
		deptDBID, ok := deptMap[fd.DepartmentID]
		if !ok {
			continue
		}

		users, err := client.ListUsersByDepartment(ctx, token, fd.DepartmentID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("users in %s: %v", fd.Name, err))
			continue
		}

		for _, fu := range users {
			if seenUsers[fu.UserID] {
				continue
			}
			seenUsers[fu.UserID] = true
			result.UsersSynced++

			// Try to match by email
			email := fu.EnterpriseEmail
			if email == "" {
				email = fu.Email
			}
			if email == "" {
				result.UsersUnmatched++
				continue
			}

			dbUser, err := s.userRepo.GetByEmail(ctx, email)
			if err != nil || dbUser == nil {
				result.UsersUnmatched++
				continue
			}
			result.UsersMatched++

			// Assign user to department
			isPrimary := len(fu.DepartmentIDs) > 0 && fu.DepartmentIDs[0] == fd.DepartmentID
			ud := &domain.UserDepartment{
				UserID:       dbUser.ID,
				DepartmentID: deptDBID,
				IsPrimary:    isPrimary,
				Role:         fu.JobTitle,
				EmployeeID:   fu.EmployeeNo,
			}
			if err := s.orgRepo.AssignUserToDepartment(ctx, ud); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("assign user %s: %v", email, err))
				continue
			}

			// Update denormalized fields if primary
			if isPrimary {
				_ = s.orgRepo.UpdateUserDenormalizedOrg(ctx, dbUser.ID, &org.ID, &deptDBID)
			}
		}
	}

	// 3. Update org metadata with sync result
	s.updateSyncMetadata(ctx, org, result, nil)

	return result, nil
}

func (s *FeishuOrgSyncService) buildFullPath(target FeishuDepartment, all []FeishuDepartment) string {
	byID := make(map[string]FeishuDepartment, len(all))
	for _, d := range all {
		byID[d.DepartmentID] = d
	}

	var parts []string
	current := target
	for i := 0; i < 20; i++ { // depth limit
		parts = append([]string{current.Name}, parts...)
		if current.ParentDepartmentID == "" || current.ParentDepartmentID == "0" {
			break
		}
		parent, ok := byID[current.ParentDepartmentID]
		if !ok {
			break
		}
		current = parent
	}
	return strings.Join(parts, "/")
}

func (s *FeishuOrgSyncService) updateSyncMetadata(ctx context.Context, org *domain.Organization, result *SyncResult, syncErr error) {
	if org.Metadata == nil {
		org.Metadata = make(map[string]any)
	}
	org.Metadata["last_sync_at"] = time.Now().Format(time.RFC3339)
	if result != nil {
		resultJSON, _ := json.Marshal(result)
		org.Metadata["last_sync_result"] = json.RawMessage(resultJSON)
	}
	if syncErr != nil {
		org.Metadata["last_sync_error"] = syncErr.Error()
	} else {
		delete(org.Metadata, "last_sync_error")
	}
	if _, err := s.orgRepo.UpsertOrganization(ctx, org); err != nil {
		s.log.Warn("failed to update sync metadata", "org_id", org.ID, "err", err)
	}
}

// GetSyncStatus returns the last sync metadata from the organization.
func (s *FeishuOrgSyncService) GetSyncStatus(ctx context.Context, orgID int64) (map[string]any, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil || org == nil {
		return nil, err
	}
	status := make(map[string]any)
	if org.Metadata != nil {
		if v, ok := org.Metadata["last_sync_at"]; ok {
			status["last_sync_at"] = v
		}
		if v, ok := org.Metadata["last_sync_result"]; ok {
			status["last_sync_result"] = v
		}
		if v, ok := org.Metadata["last_sync_error"]; ok {
			status["last_sync_error"] = v
		}
		if v, ok := org.Metadata["feishu_app_id"]; ok {
			status["feishu_app_id"] = v
		}
		status["feishu_configured"] = org.Metadata["feishu_app_id"] != nil && org.Metadata["feishu_app_id"] != ""
	}
	return status, nil
}

var _ = sql.ErrNoRows
