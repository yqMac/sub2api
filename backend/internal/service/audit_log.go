// [bmai-fork] audit log service — async write with hot-reloadable config
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type AuditLogService struct {
	repo    AuditLogRepository
	orgRepo OrganizationRepository // [bmai-fork] for user org/dept lookup
	cfg     atomic.Value           // *config.AuditConfig
	log     *slog.Logger
}

func NewAuditLogService(repo AuditLogRepository, orgRepo OrganizationRepository, cfg *config.Config) *AuditLogService {
	s := &AuditLogService{
		repo:    repo,
		orgRepo: orgRepo,
		log:     slog.Default().With("component", "audit"),
	}
	auditCfg := cfg.Audit
	if auditCfg.MaxRequestBytes == 0 {
		auditCfg.MaxRequestBytes = 32768
	}
	if auditCfg.MaxResponseBytes == 0 {
		auditCfg.MaxResponseBytes = 32768
	}
	if auditCfg.RetentionDays == 0 {
		auditCfg.RetentionDays = 30
	}
	s.cfg.Store(&auditCfg)
	return s
}

func (s *AuditLogService) UpdateConfig(cfg *config.AuditConfig) {
	s.cfg.Store(cfg)
}

func (s *AuditLogService) Config() *config.AuditConfig {
	return s.cfg.Load().(*config.AuditConfig)
}

func (s *AuditLogService) Enabled() bool {
	return s.Config().Enabled
}

func (s *AuditLogService) MaxRequestBytes() int {
	return s.Config().MaxRequestBytes
}

func (s *AuditLogService) MaxResponseBytes() int {
	return s.Config().MaxResponseBytes
}

func (s *AuditLogService) CaptureUpstream() bool {
	return s.Config().CaptureUpstream
}

func (s *AuditLogService) ClassifyResponse() bool {
	return s.Config().ClassifyResponse
}

func (s *AuditLogService) Submit(log *domain.AuditLog) {
	if !s.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// [bmai-fork] enrich with org info if not already populated by caller
		if log.OrganizationID == nil && log.DepartmentID == nil && s.orgRepo != nil && log.UserID > 0 {
			if orgID, deptID, deptPath, err := s.orgRepo.GetUserOrgInfo(ctx, log.UserID); err == nil {
				log.OrganizationID = orgID
				log.DepartmentID = deptID
				if log.DepartmentPath == "" {
					log.DepartmentPath = deptPath
				}
			}
		}
		if err := s.repo.Insert(ctx, log); err != nil {
			s.log.Error("audit log insert failed", "err", err, "request_id", log.RequestID)
		}
	}()
}

func (s *AuditLogService) Get(ctx context.Context, id int64) (*domain.AuditLog, error) {
	return s.repo.Get(ctx, id)
}

func (s *AuditLogService) List(ctx context.Context, filter AuditLogFilter) ([]*domain.AuditLog, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *AuditLogService) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return s.repo.DeleteBefore(ctx, before)
}

func (s *AuditLogService) StorageInfo(ctx context.Context) (*AuditStorageInfo, error) {
	return s.repo.StorageInfo(ctx)
}

// [bmai-fork] EnsurePartitions creates the current and next month partitions if they don't exist.
func (s *AuditLogService) EnsurePartitions(ctx context.Context) {
	now := time.Now()
	months := []time.Time{
		time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(now.Year(), now.Month()+2, 1, 0, 0, 0, 0, time.UTC),
	}
	for i := 0; i < len(months)-1; i++ {
		name := months[i].Format("audit_logs_y2006m01")
		from := months[i].Format("2006-01-02")
		to := months[i+1].Format("2006-01-02")
		ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_logs FOR VALUES FROM ('%s') TO ('%s')`, name, from, to)
		if err := s.repo.ExecDDL(ctx, ddl); err != nil {
			s.log.Warn("failed to ensure audit partition", "name", name, "err", err)
		}
	}
	for i := 0; i < len(months)-1; i++ {
		name := months[i].Format("audit_rule_events_y2006m01")
		from := months[i].Format("2006-01-02")
		to := months[i+1].Format("2006-01-02")
		ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_rule_events FOR VALUES FROM ('%s') TO ('%s')`, name, from, to)
		if err := s.repo.ExecDDL(ctx, ddl); err != nil {
			s.log.Warn("failed to ensure audit_rule_events partition", "name", name, "err", err)
		}
	}
	s.log.Info("audit partitions ensured", "months", len(months)-1)
}
