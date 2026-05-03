// [bmai-fork] audit log repository — raw SQL on *sql.DB
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type auditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) service.AuditLogRepository {
	return &auditLogRepository{db: db}
}

const auditLogColumns = `id, request_id, user_id, api_key_id, account_id,
	organization_id, department_id, department_path, content_type,
	request_preview, request_bytes, request_truncated,
	response_preview, response_bytes, response_truncated,
	upstream_request_preview, upstream_request_bytes, upstream_request_truncated,
	upstream_response_preview, upstream_response_bytes, upstream_response_truncated,
	model, platform, endpoint, stream, duration_ms, input_tokens, output_tokens, status_code,
	intercepted, intercept_reason, risk_level, risk_tags, created_at`

func (r *auditLogRepository) Insert(ctx context.Context, log *domain.AuditLog) error {
	const q = `INSERT INTO audit_logs (
		request_id, user_id, api_key_id, account_id,
		organization_id, department_id, department_path, content_type,
		request_preview, request_bytes, request_truncated,
		response_preview, response_bytes, response_truncated,
		upstream_request_preview, upstream_request_bytes, upstream_request_truncated,
		upstream_response_preview, upstream_response_bytes, upstream_response_truncated,
		model, platform, endpoint, stream, duration_ms, input_tokens, output_tokens, status_code,
		intercepted, intercept_reason, risk_level, risk_tags, created_at
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33
	)`
	_, err := r.db.ExecContext(ctx, q,
		log.RequestID, log.UserID, log.APIKeyID, log.AccountID,
		log.OrganizationID, log.DepartmentID, nilIfEmpty(log.DepartmentPath), nilIfEmpty(log.ContentType),
		nilIfEmpty(log.RequestPreview), log.RequestBytes, log.RequestTruncated,
		nilIfEmpty(log.ResponsePreview), log.ResponseBytes, log.ResponseTruncated,
		nilIfEmpty(log.UpstreamRequestPreview), log.UpstreamRequestBytes, log.UpstreamRequestTruncated,
		nilIfEmpty(log.UpstreamResponsePreview), log.UpstreamResponseBytes, log.UpstreamResponseTruncated,
		nilIfEmpty(log.Model), nilIfEmpty(log.Platform), nilIfEmpty(log.Endpoint),
		log.Stream, log.DurationMs, log.InputTokens, log.OutputTokens, log.StatusCode,
		log.Intercepted, nilIfEmpty(log.InterceptReason), nilIfEmpty(log.RiskLevel),
		pq.Array(log.RiskTags), log.CreatedAt,
	)
	return err
}

func (r *auditLogRepository) Get(ctx context.Context, id int64) (*domain.AuditLog, error) {
	q := `SELECT ` + auditLogColumns + ` FROM audit_logs WHERE id = $1`
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

func (r *auditLogRepository) List(ctx context.Context, f service.AuditLogFilter) ([]*domain.AuditLog, int64, error) {
	where, args := buildAuditWhere(f)

	var total int64
	countQ := `SELECT count(*) FROM audit_logs` + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	sortCol := "created_at"
	switch f.SortBy {
	case "duration_ms", "input_tokens", "output_tokens":
		sortCol = f.SortBy
	}
	sortDir := "DESC"
	if strings.EqualFold(f.SortOrder, "asc") {
		sortDir = "ASC"
	}

	page, pageSize := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	dataQ := fmt.Sprintf(`SELECT %s FROM audit_logs%s ORDER BY %s %s LIMIT %d OFFSET %d`,
		auditLogColumns, where, sortCol, sortDir, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		l, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

func (r *auditLogRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM audit_logs WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *auditLogRepository) StorageInfo(ctx context.Context) (*service.AuditStorageInfo, error) {
	info := &service.AuditStorageInfo{}

	r.db.QueryRowContext(ctx, `SELECT count(*), COALESCE(min(created_at), NOW()) FROM audit_logs`).Scan(&info.TotalRows, &info.EarliestRecord)

	r.db.QueryRowContext(ctx, `SELECT COALESCE(pg_total_relation_size('audit_logs'), 0)`).Scan(&info.TotalBytes)

	rows, err := r.db.QueryContext(ctx, `
		SELECT c.relname, COALESCE(c.reltuples::bigint, 0), COALESCE(pg_total_relation_size(c.oid), 0)
		FROM pg_class c JOIN pg_inherits i ON c.oid = i.inhrelid
		WHERE i.inhparent = 'audit_logs'::regclass ORDER BY c.relname`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p service.AuditPartitionInfo
			rows.Scan(&p.Name, &p.Rows, &p.Bytes)
			info.Partitions = append(info.Partitions, p)
		}
	}
	return info, nil
}

func buildAuditWhere(f service.AuditLogFilter) (string, []any) {
	var conds []string
	var args []any
	n := 0
	add := func(cond string, val any) {
		n++
		conds = append(conds, fmt.Sprintf(cond, n))
		args = append(args, val)
	}

	if f.StartTime != nil {
		add("created_at >= $%d", *f.StartTime)
	}
	if f.EndTime != nil {
		add("created_at <= $%d", *f.EndTime)
	}
	if len(f.UserIDs) > 0 {
		add("user_id = ANY($%d)", pq.Array(f.UserIDs))
	}
	if f.OrganizationID != nil {
		add("organization_id = $%d", *f.OrganizationID)
	}
	if f.DepartmentID != nil {
		add("department_id = $%d", *f.DepartmentID)
	}
	if f.DepartmentPath != "" {
		add("department_path LIKE $%d", f.DepartmentPath+"%")
	}
	if len(f.ContentTypes) > 0 {
		add("content_type = ANY($%d)", pq.Array(f.ContentTypes))
	}
	if len(f.Models) > 0 {
		add("model = ANY($%d)", pq.Array(f.Models))
	}
	if len(f.Platforms) > 0 {
		add("platform = ANY($%d)", pq.Array(f.Platforms))
	}
	if f.Keyword != "" {
		n++
		kw := "%" + escLike(f.Keyword) + "%"
		conds = append(conds, fmt.Sprintf("(request_preview LIKE $%d OR response_preview LIKE $%d)", n, n))
		args = append(args, kw)
	}
	if f.RiskLevel != "" {
		add("risk_level = $%d", f.RiskLevel)
	}
	if f.Intercepted != nil {
		add("intercepted = $%d", *f.Intercepted)
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func escLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (r *auditLogRepository) scanOne(row *sql.Row) (*domain.AuditLog, error) {
	l := &domain.AuditLog{}
	var riskTags pq.StringArray
	err := row.Scan(
		&l.ID, &l.RequestID, &l.UserID, &l.APIKeyID, &l.AccountID,
		&l.OrganizationID, &l.DepartmentID, &l.DepartmentPath, &l.ContentType,
		&l.RequestPreview, &l.RequestBytes, &l.RequestTruncated,
		&l.ResponsePreview, &l.ResponseBytes, &l.ResponseTruncated,
		&l.UpstreamRequestPreview, &l.UpstreamRequestBytes, &l.UpstreamRequestTruncated,
		&l.UpstreamResponsePreview, &l.UpstreamResponseBytes, &l.UpstreamResponseTruncated,
		&l.Model, &l.Platform, &l.Endpoint, &l.Stream, &l.DurationMs, &l.InputTokens, &l.OutputTokens, &l.StatusCode,
		&l.Intercepted, &l.InterceptReason, &l.RiskLevel, &riskTags, &l.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	l.RiskTags = riskTags
	return l, err
}

func (r *auditLogRepository) scanRow(rows *sql.Rows) (*domain.AuditLog, error) {
	l := &domain.AuditLog{}
	var riskTags pq.StringArray
	err := rows.Scan(
		&l.ID, &l.RequestID, &l.UserID, &l.APIKeyID, &l.AccountID,
		&l.OrganizationID, &l.DepartmentID, &l.DepartmentPath, &l.ContentType,
		&l.RequestPreview, &l.RequestBytes, &l.RequestTruncated,
		&l.ResponsePreview, &l.ResponseBytes, &l.ResponseTruncated,
		&l.UpstreamRequestPreview, &l.UpstreamRequestBytes, &l.UpstreamRequestTruncated,
		&l.UpstreamResponsePreview, &l.UpstreamResponseBytes, &l.UpstreamResponseTruncated,
		&l.Model, &l.Platform, &l.Endpoint, &l.Stream, &l.DurationMs, &l.InputTokens, &l.OutputTokens, &l.StatusCode,
		&l.Intercepted, &l.InterceptReason, &l.RiskLevel, &riskTags, &l.CreatedAt,
	)
	l.RiskTags = riskTags
	return l, err
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var _ = time.Now

// [bmai-fork] ExecDDL runs arbitrary DDL (used for partition creation)
func (r *auditLogRepository) ExecDDL(ctx context.Context, ddl string) error {
	_, err := r.db.ExecContext(ctx, ddl)
	return err
}
