// [bmai-fork] audit log repository interface
package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type AuditLogRepository interface {
	Insert(ctx context.Context, log *domain.AuditLog) error
	Get(ctx context.Context, id int64) (*domain.AuditLog, error)
	List(ctx context.Context, filter AuditLogFilter) ([]*domain.AuditLog, int64, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
	StorageInfo(ctx context.Context) (*AuditStorageInfo, error)
	ExecDDL(ctx context.Context, ddl string) error
}

type AuditLogFilter struct {
	StartTime      *time.Time
	EndTime        *time.Time
	UserIDs        []int64
	OrganizationID *int64
	DepartmentID   *int64
	DepartmentPath string
	ContentTypes   []string
	Models         []string
	Platforms      []string
	Keyword        string
	RiskLevel      string
	Intercepted    *bool
	Page           int
	PageSize       int
	SortBy         string
	SortOrder      string
}

type AuditStorageInfo struct {
	TotalRows      int64
	TotalBytes     int64
	EarliestRecord *time.Time
	Partitions     []AuditPartitionInfo
}

type AuditPartitionInfo struct {
	Name  string
	Rows  int64
	Bytes int64
}
