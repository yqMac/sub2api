// [bmai-fork] audit log service — async write with hot-reloadable config
package service

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type AuditLogService struct {
	repo AuditLogRepository
	cfg  atomic.Value // *config.AuditConfig
	log  *slog.Logger
}

func NewAuditLogService(repo AuditLogRepository, cfg *config.Config) *AuditLogService {
	s := &AuditLogService{
		repo: repo,
		log:  slog.Default().With("component", "audit"),
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
