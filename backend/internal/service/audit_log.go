// [bmai-fork] audit log service — async write with hot-reloadable config
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
)

const auditSettingsKey = "audit_config"

type AuditLogService struct {
	repo        AuditLogRepository
	settingRepo SettingRepository          // [bmai-fork] for config persistence
	orgRepo     OrganizationRepository     // [bmai-fork] for user org/dept lookup
	cfg         atomic.Value               // *config.AuditConfig
	log         *slog.Logger
	submitCh    chan *domain.AuditLog       // bounded worker channel
	wg          sync.WaitGroup
}

func NewAuditLogService(repo AuditLogRepository, orgRepo OrganizationRepository, settingRepo SettingRepository, cfg *config.Config) *AuditLogService {
	s := &AuditLogService{
		repo:        repo,
		orgRepo:     orgRepo,
		settingRepo: settingRepo,
		log:         slog.Default().With("component", "audit"),
		submitCh:    make(chan *domain.AuditLog, 256),
	}

	// Load config: DB settings override env/file config
	auditCfg := cfg.Audit
	s.applyDefaults(&auditCfg)
	if settingRepo != nil {
		if dbCfg, err := s.loadConfigFromDB(context.Background()); err == nil && dbCfg != nil {
			auditCfg = *dbCfg
			s.applyDefaults(&auditCfg)
		}
	}
	s.cfg.Store(&auditCfg)

	// Start worker pool (4 workers, bounded channel)
	for i := 0; i < 4; i++ {
		s.wg.Add(1)
		go s.worker()
	}

	return s
}

func (s *AuditLogService) applyDefaults(cfg *config.AuditConfig) {
	if cfg.MaxRequestBytes == 0 {
		cfg.MaxRequestBytes = 32768
	}
	if cfg.MaxResponseBytes == 0 {
		cfg.MaxResponseBytes = 32768
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 30
	}
}

func (s *AuditLogService) loadConfigFromDB(ctx context.Context) (*config.AuditConfig, error) {
	val, err := s.settingRepo.GetValue(ctx, auditSettingsKey)
	if err != nil || val == "" {
		return nil, err
	}
	var cfg config.AuditConfig
	if err := json.Unmarshal([]byte(val), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *AuditLogService) saveConfigToDB(ctx context.Context, cfg *config.AuditConfig) error {
	if s.settingRepo == nil {
		return nil
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, auditSettingsKey, string(data))
}

func (s *AuditLogService) UpdateConfig(cfg *config.AuditConfig) {
	s.cfg.Store(cfg)
	if err := s.saveConfigToDB(context.Background(), cfg); err != nil {
		s.log.Warn("failed to persist audit config", "err", err)
	}
}

func (s *AuditLogService) Config() *config.AuditConfig {
	return s.cfg.Load().(*config.AuditConfig)
}

func (s *AuditLogService) Enabled() bool          { return s.Config().Enabled }
func (s *AuditLogService) MaxRequestBytes() int    { return s.Config().MaxRequestBytes }
func (s *AuditLogService) MaxResponseBytes() int   { return s.Config().MaxResponseBytes }
func (s *AuditLogService) CaptureUpstream() bool   { return s.Config().CaptureUpstream }
func (s *AuditLogService) ClassifyResponse() bool  { return s.Config().ClassifyResponse }

// Submit enqueues an audit log for async writing. Non-blocking; drops if channel full.
func (s *AuditLogService) Submit(log *domain.AuditLog) {
	if !s.Enabled() {
		return
	}
	select {
	case s.submitCh <- log:
	default:
		s.log.Warn("audit submit channel full, dropping", "request_id", log.RequestID)
	}
}

func (s *AuditLogService) worker() {
	defer s.wg.Done()
	for log := range s.submitCh {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		cancel()
	}
}

// Shutdown drains the worker pool. Call on graceful shutdown.
func (s *AuditLogService) Shutdown() {
	close(s.submitCh)
	s.wg.Wait()
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
