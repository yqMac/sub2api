# 审计功能 后端详细设计

> **范围**：sub2api 后端（Go），二开标记 `[bmai-fork]`
> **相关文档**：[PLAN.md](./PLAN.md) · [DESIGN-FRONTEND.md](./DESIGN-FRONTEND.md) · [CONVENTIONS.md](./CONVENTIONS.md)

## 目录

- [一、目录结构](#一目录结构)
- [二、数据模型](#二数据模型)
- [三、领域层](#三领域层)
- [四、服务层](#四服务层)
- [五、仓储层](#五仓储层)
- [六、Handler 层](#六handler-层)
- [七、API 规范](#七api-规范)
- [八、Hook 集成点](#八hook-集成点)
- [九、配置](#九配置)
- [十、依赖注入与启动](#十依赖注入与启动)
- [十一、测试策略](#十一测试策略)

---

## 一、目录结构

```
backend/
├── migrations/
│   ├── 135_create_organizations.sql       [bmai-fork]
│   ├── 136_create_audit_logs.sql          [bmai-fork]
│   └── 137_create_audit_rules.sql         [bmai-fork]
├── internal/
│   ├── domain/
│   │   ├── organization.go                [bmai-fork] 组织/部门领域模型
│   │   └── audit.go                       [bmai-fork] 审计领域模型 + 常量
│   ├── service/
│   │   ├── audit_log.go                   [bmai-fork] AuditLog 写入服务
│   │   ├── audit_capture.go               [bmai-fork] 流式 tee buffer
│   │   ├── audit_classify.go              [bmai-fork] 内容类型分类
│   │   ├── audit_rule_engine.go           [bmai-fork] 规则匹配引擎（P2）
│   │   ├── audit_pre_hook.go              [bmai-fork] 请求前拦截 hook（P2）
│   │   ├── audit_post_hook.go             [bmai-fork] 响应后异步 hook（P2）
│   │   ├── audit_stats_service.go         [bmai-fork] 统计聚合服务（P1）
│   │   ├── audit_aggregation_worker.go    [bmai-fork] 预聚合定时任务（P1）
│   │   ├── organization_service.go        [bmai-fork] 组织/部门服务
│   │   └── feishu_org_sync.go             [bmai-fork] 飞书部门同步
│   ├── repository/
│   │   ├── audit_log_repo.go              [bmai-fork] 审计日志读写
│   │   ├── audit_stats_repo.go            [bmai-fork] 预聚合表读写（P1）
│   │   ├── audit_rule_repo.go             [bmai-fork] 规则 CRUD（P2）
│   │   └── organization_repo.go           [bmai-fork] 组织/部门 CRUD
│   └── handler/admin/
│       ├── audit_handler.go               [bmai-fork] 日志列表/详情 API
│       ├── audit_settings_handler.go      [bmai-fork] 设置 API
│       ├── audit_stats_handler.go         [bmai-fork] 统计 API（P1）
│       ├── audit_rule_handler.go          [bmai-fork] 规则 API（P2）
│       └── organization_handler.go        [bmai-fork] 组织 API
```

---

## 二、数据模型

### 2.1 组织架构表

```sql
-- migrations/135_create_organizations.sql
-- [bmai-fork] organization & department schema for audit dimensions

-- 组织（一个飞书 tenant 一行）
CREATE TABLE organizations (
    id              BIGSERIAL PRIMARY KEY,
    tenant_key      VARCHAR(100) UNIQUE NOT NULL,
    name            VARCHAR(200) NOT NULL,
    type            VARCHAR(30) NOT NULL DEFAULT 'feishu',
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 部门（嵌套层级）
CREATE TABLE departments (
    id                  BIGSERIAL PRIMARY KEY,
    organization_id     BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    parent_id           BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    external_id         VARCHAR(200) NOT NULL,
    name                VARCHAR(200) NOT NULL,
    full_path           VARCHAR(1000) NOT NULL DEFAULT '',
    level               INTEGER NOT NULL DEFAULT 1,
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, external_id)
);

CREATE INDEX idx_departments_org ON departments (organization_id);
CREATE INDEX idx_departments_parent ON departments (parent_id);
CREATE INDEX idx_departments_path ON departments (full_path varchar_pattern_ops);

-- 用户-部门关联（一用户可属多部门）
CREATE TABLE user_departments (
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id   BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    role            VARCHAR(50),
    employee_id     VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, department_id)
);

CREATE INDEX idx_user_departments_dept ON user_departments (department_id);

-- users 表加冗余列（写入时填充，避免每次查询都 JOIN）
ALTER TABLE users ADD COLUMN organization_id BIGINT;
ALTER TABLE users ADD COLUMN primary_department_id BIGINT;
CREATE INDEX idx_users_org ON users (organization_id);
CREATE INDEX idx_users_primary_dept ON users (primary_department_id);
```

### 2.2 审计日志主表

```sql
-- migrations/136_create_audit_logs.sql
-- [bmai-fork] audit logs for request/response capture

CREATE TABLE audit_logs (
    id                              BIGSERIAL,
    request_id                      VARCHAR(64) NOT NULL,
    user_id                         BIGINT NOT NULL,
    api_key_id                      BIGINT NOT NULL,
    account_id                      BIGINT NOT NULL,

    -- 组织维度（写入时冗余）
    organization_id                 BIGINT,
    department_id                   BIGINT,
    department_path                 VARCHAR(1000),

    -- 内容分类
    content_type                    VARCHAR(30),

    -- 用户原始请求
    request_preview                 TEXT,
    request_bytes                   INTEGER,
    request_truncated               BOOLEAN NOT NULL DEFAULT FALSE,

    -- 用户收到的响应
    response_preview                TEXT,
    response_bytes                  INTEGER,
    response_truncated              BOOLEAN NOT NULL DEFAULT FALSE,

    -- 上游请求
    upstream_request_preview        TEXT,
    upstream_request_bytes          INTEGER,
    upstream_request_truncated      BOOLEAN NOT NULL DEFAULT FALSE,

    -- 上游响应
    upstream_response_preview       TEXT,
    upstream_response_bytes         INTEGER,
    upstream_response_truncated     BOOLEAN NOT NULL DEFAULT FALSE,

    -- 元数据
    model                           VARCHAR(100),
    platform                        VARCHAR(30),
    endpoint                        VARCHAR(200),
    stream                          BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms                     INTEGER,
    input_tokens                    INTEGER NOT NULL DEFAULT 0,
    output_tokens                   INTEGER NOT NULL DEFAULT 0,
    status_code                     INTEGER,

    -- 拦截/告警标记（P2 填充）
    intercepted                     BOOLEAN NOT NULL DEFAULT FALSE,
    intercept_reason                VARCHAR(200),
    risk_level                      VARCHAR(20),
    risk_tags                       TEXT[],

    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建初始分区（当月）
CREATE TABLE audit_logs_y2026m05 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE INDEX idx_audit_logs_request_id ON audit_logs (request_id);
CREATE INDEX idx_audit_logs_user_created ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_org_created ON audit_logs (organization_id, created_at DESC);
CREATE INDEX idx_audit_logs_dept_created ON audit_logs (department_id, created_at DESC);
CREATE INDEX idx_audit_logs_dept_path ON audit_logs (department_path varchar_pattern_ops, created_at DESC);
CREATE INDEX idx_audit_logs_content_type ON audit_logs (content_type, created_at DESC);
CREATE INDEX idx_audit_logs_model ON audit_logs (model, created_at DESC);
CREATE INDEX idx_audit_logs_intercepted ON audit_logs (intercepted, created_at DESC) WHERE intercepted = TRUE;

-- 多粒度多维度预聚合表（P1）
CREATE TABLE audit_stats (
    id                      BIGSERIAL,
    bucket_size             VARCHAR(10) NOT NULL,
    bucket_start            TIMESTAMPTZ NOT NULL,

    organization_id         BIGINT,
    department_id           BIGINT,
    user_id                 BIGINT,
    content_type            VARCHAR(30),
    model                   VARCHAR(100),
    platform                VARCHAR(30),

    request_count           BIGINT NOT NULL DEFAULT 0,
    success_count           BIGINT NOT NULL DEFAULT 0,
    error_count             BIGINT NOT NULL DEFAULT 0,
    intercepted_count       BIGINT NOT NULL DEFAULT 0,
    risk_count              BIGINT NOT NULL DEFAULT 0,
    total_input_tokens      BIGINT NOT NULL DEFAULT 0,
    total_output_tokens     BIGINT NOT NULL DEFAULT 0,
    total_duration_ms       BIGINT NOT NULL DEFAULT 0,
    p50_duration_ms         INTEGER,
    p95_duration_ms         INTEGER,
    p99_duration_ms         INTEGER,

    PRIMARY KEY (id, bucket_start)
) PARTITION BY RANGE (bucket_start);

CREATE TABLE audit_stats_y2026m05 PARTITION OF audit_stats
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE UNIQUE INDEX idx_audit_stats_unique ON audit_stats (
    bucket_size, bucket_start,
    COALESCE(organization_id, 0),
    COALESCE(department_id, 0),
    COALESCE(user_id, 0),
    COALESCE(content_type, ''),
    COALESCE(model, ''),
    COALESCE(platform, '')
);
CREATE INDEX idx_audit_stats_query ON audit_stats (bucket_size, bucket_start DESC, organization_id, department_id);
```

### 2.3 规则表

```sql
-- migrations/137_create_audit_rules.sql
-- [bmai-fork] audit rules for interception and alerting

CREATE TABLE audit_rules (
    id                      BIGSERIAL PRIMARY KEY,
    name                    VARCHAR(200) NOT NULL,
    description             TEXT,
    enabled                 BOOLEAN NOT NULL DEFAULT TRUE,
    priority                INTEGER NOT NULL DEFAULT 100,

    rule_type               VARCHAR(30) NOT NULL,
    -- pre_block / post_alert / post_tag

    scope_organization_ids  BIGINT[],
    scope_department_ids    BIGINT[],
    scope_user_ids          BIGINT[],
    scope_models            TEXT[],
    scope_content_types     TEXT[],

    match_conditions        JSONB NOT NULL,
    action                  VARCHAR(30) NOT NULL,
    action_config           JSONB,

    created_by              BIGINT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_rules_enabled ON audit_rules (enabled, priority);

CREATE TABLE audit_rule_events (
    id              BIGSERIAL,
    rule_id         BIGINT NOT NULL,
    audit_log_id    BIGINT,
    request_id      VARCHAR(64),
    user_id         BIGINT,
    department_id   BIGINT,
    organization_id BIGINT,
    action_taken    VARCHAR(30),
    matched_field   VARCHAR(50),
    matched_value   TEXT,
    severity        VARCHAR(20),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE audit_rule_events_y2026m05 PARTITION OF audit_rule_events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE INDEX idx_audit_rule_events_rule ON audit_rule_events (rule_id, created_at DESC);
CREATE INDEX idx_audit_rule_events_user ON audit_rule_events (user_id, created_at DESC);
```

---

## 三、领域层

### 3.1 `domain/organization.go`

```go
package domain

import "time"

// [bmai-fork] organization domain model
type Organization struct {
    ID         int64
    TenantKey  string
    Name       string
    Type       string  // feishu / manual / oidc
    Metadata   map[string]any
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type Department struct {
    ID                  int64
    OrganizationID      int64
    ParentID            *int64
    ExternalID          string
    Name                string
    FullPath            string
    Level               int
    Metadata            map[string]any
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

type UserDepartment struct {
    UserID       int64
    DepartmentID int64
    IsPrimary    bool
    Role         string
    EmployeeID   string
    CreatedAt    time.Time
}
```

### 3.2 `domain/audit.go`

```go
package domain

import "time"

// [bmai-fork] audit content type constants
const (
    ContentTypeConversation = "conversation"
    ContentTypeCode         = "code"
    ContentTypeImage        = "image"
    ContentTypeToolUse      = "tool_use"
    ContentTypeReasoning    = "reasoning"
    ContentTypePlan         = "plan"
    ContentTypeScript       = "script"
)

const (
    RiskLevelLow    = "low"
    RiskLevelMedium = "medium"
    RiskLevelHigh   = "high"
)

const (
    AuditRuleTypePreBlock  = "pre_block"
    AuditRuleTypePostAlert = "post_alert"
    AuditRuleTypePostTag   = "post_tag"
)

const (
    AuditActionBlock  = "block"
    AuditActionWarn   = "warn"
    AuditActionAlert  = "alert"
    AuditActionLog    = "log"
    AuditActionTag    = "tag"
)

type AuditLog struct {
    ID              int64
    RequestID       string
    UserID          int64
    APIKeyID        int64
    AccountID       int64
    OrganizationID  *int64
    DepartmentID    *int64
    DepartmentPath  string
    ContentType     string

    RequestPreview            string
    RequestBytes              int
    RequestTruncated          bool
    ResponsePreview           string
    ResponseBytes             int
    ResponseTruncated         bool
    UpstreamRequestPreview    string
    UpstreamRequestBytes      int
    UpstreamRequestTruncated  bool
    UpstreamResponsePreview   string
    UpstreamResponseBytes     int
    UpstreamResponseTruncated bool

    Model        string
    Platform     string
    Endpoint     string
    Stream       bool
    DurationMs   int
    InputTokens  int
    OutputTokens int
    StatusCode   int

    Intercepted     bool
    InterceptReason string
    RiskLevel       string
    RiskTags        []string

    CreatedAt time.Time
}
```

---

## 四、服务层

### 4.1 AuditCaptureBuffer（流式 tee）

`service/audit_capture.go`：

```go
package service

import (
    "bytes"
    "sync"
)

// [bmai-fork] capped buffer used to capture streaming response previews.
// When written-bytes exceed Limit, further writes are dropped but Total still increments.
type AuditCaptureBuffer struct {
    buf   *bytes.Buffer
    Total int
    Limit int
}

var auditCaptureBufferPool = sync.Pool{
    New: func() any {
        return &AuditCaptureBuffer{buf: bytes.NewBuffer(nil)}
    },
}

func AcquireAuditCaptureBuffer(limit int) *AuditCaptureBuffer {
    b := auditCaptureBufferPool.Get().(*AuditCaptureBuffer)
    b.buf.Reset()
    b.Total = 0
    b.Limit = limit
    return b
}

func ReleaseAuditCaptureBuffer(b *AuditCaptureBuffer) {
    auditCaptureBufferPool.Put(b)
}

func (b *AuditCaptureBuffer) Write(p []byte) (int, error) {
    b.Total += len(p)
    if b.buf.Len() >= b.Limit {
        return len(p), nil
    }
    remain := b.Limit - b.buf.Len()
    if remain >= len(p) {
        b.buf.Write(p)
    } else {
        b.buf.Write(p[:remain])
    }
    return len(p), nil
}

func (b *AuditCaptureBuffer) Bytes() []byte    { return b.buf.Bytes() }
func (b *AuditCaptureBuffer) Truncated() bool  { return b.Total > b.buf.Len() }
```

### 4.2 内容分类

`service/audit_classify.go`：

```go
package service

import "regexp"

// [bmai-fork] content type classification at request and response time.

type AuditClassifyInput struct {
    Endpoint     string
    Model        string
    BillingMode  string
    HasTools     bool
    HasThinking  bool
    HasReasoning bool
}

func ClassifyContentType(in AuditClassifyInput) string {
    switch {
    case in.BillingMode == "image" || isImageEndpoint(in.Endpoint):
        return ContentTypeImage
    case in.HasTools:
        return ContentTypeToolUse
    case in.HasThinking || in.HasReasoning:
        return ContentTypeReasoning
    default:
        return ContentTypeConversation
    }
}

var (
    codeFencePattern = regexp.MustCompile("```(python|go|javascript|typescript|java|cpp|c|rust|bash|sh|sql|json|yaml|html|css|vue|jsx|tsx)")
    scriptPattern    = regexp.MustCompile(`(?m)^#!/(usr/)?bin/`)
    planPattern      = regexp.MustCompile(`(?m)^##\s+\S+`)
)

// RefineContentType: response-time refinement based on captured preview.
// Returns the refined type or the original if no refinement matched.
func RefineContentType(original string, responsePreview []byte) string {
    if len(responsePreview) == 0 {
        return original
    }
    if codeFencePattern.Match(responsePreview) {
        return ContentTypeCode
    }
    if scriptPattern.Match(responsePreview) {
        return ContentTypeScript
    }
    if planPattern.Match(responsePreview) {
        return ContentTypePlan
    }
    return original
}

func isImageEndpoint(endpoint string) bool {
    return endpoint == "/v1/images/generations" || endpoint == "/v1/images/edits"
}
```

### 4.3 AuditLogService

`service/audit_log.go`：

```go
package service

import (
    "context"

    "sub2api/backend/internal/domain"
)

// [bmai-fork] AuditLogService writes audit logs asynchronously.
type AuditLogService struct {
    repo    AuditLogRepository
    cfg     *AuditConfig
    workers *UsageRecordWorkerPool  // 复用现有 worker pool
}

type AuditLogRepository interface {
    Insert(ctx context.Context, log *domain.AuditLog) error
    BatchInsert(ctx context.Context, logs []*domain.AuditLog) error
    List(ctx context.Context, filter AuditLogFilter) ([]*domain.AuditLog, int64, error)
    Get(ctx context.Context, id int64) (*domain.AuditLog, error)
}

type AuditLogFilter struct {
    StartTime      *time.Time
    EndTime        *time.Time
    UserIDs        []int64
    OrganizationID *int64
    DepartmentID   *int64
    DepartmentPath string  // prefix match
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

func (s *AuditLogService) Submit(ctx context.Context, log *domain.AuditLog) {
    if !s.cfg.Enabled {
        return
    }
    s.workers.Submit(func() {
        if err := s.repo.Insert(ctx, log); err != nil {
            // log only, never fail the request
        }
    })
}
```

### 4.4 OrganizationService

`service/organization_service.go`：

```go
type OrganizationService struct {
    repo OrganizationRepository
}

type OrganizationRepository interface {
    GetByTenantKey(ctx context.Context, tenantKey string) (*domain.Organization, error)
    UpsertOrganization(ctx context.Context, org *domain.Organization) error
    UpsertDepartment(ctx context.Context, dept *domain.Department) error
    AssignUserToDepartment(ctx context.Context, ud *domain.UserDepartment) error
    GetUserDepartments(ctx context.Context, userID int64) ([]*domain.Department, error)
    ListDepartmentTree(ctx context.Context, orgID int64) ([]*domain.Department, error)
}
```

### 4.5 飞书部门同步

`service/feishu_org_sync.go`：

```go
// [bmai-fork] sync user organization & department info from Feishu after OAuth login.
// 在飞书 OAuth 回调完成后异步触发，失败不影响登录。
type FeishuOrgSyncService struct {
    orgRepo  OrganizationRepository
    feishu   FeishuClient   // 调用飞书 contact API
    logger   Logger
}

func (s *FeishuOrgSyncService) SyncForUser(ctx context.Context, userID int64, claims FeishuUpstreamClaims) {
    // 1. UpsertOrganization(tenant_key, ...)
    // 2. 调用飞书 API: GET /open-apis/contact/v3/users/{open_id}/departments
    // 3. UpsertDepartment for each
    // 4. AssignUserToDepartment(user_id, dept_id, is_primary)
}
```

---

## 五、仓储层

### 5.1 AuditLogRepo

`repository/audit_log_repo.go`：

```go
// [bmai-fork] audit_logs read/write
type auditLogRepo struct {
    db *sqlx.DB
}

func (r *auditLogRepo) Insert(ctx context.Context, log *domain.AuditLog) error {
    const q = `INSERT INTO audit_logs (
        request_id, user_id, api_key_id, account_id,
        organization_id, department_id, department_path,
        content_type,
        request_preview, request_bytes, request_truncated,
        response_preview, response_bytes, response_truncated,
        upstream_request_preview, upstream_request_bytes, upstream_request_truncated,
        upstream_response_preview, upstream_response_bytes, upstream_response_truncated,
        model, platform, endpoint, stream, duration_ms, input_tokens, output_tokens, status_code,
        intercepted, intercept_reason, risk_level, risk_tags,
        created_at
    ) VALUES (...)`
    // 参数化绑定
}

// List 支持复合筛选 + 分页
func (r *auditLogRepo) List(ctx context.Context, f AuditLogFilter) ([]*domain.AuditLog, int64, error) {
    // 动态构建 WHERE：
    // - 时间范围（必填，默认 24h）
    // - 用户/组织/部门/内容类型/模型/平台
    // - department_path LIKE '部门路径/%'
    // - keyword 匹配 request_preview/response_preview（pg_trgm 或 LIKE）
    // - 风险/拦截标记
    // 排序：created_at DESC（默认）/ duration_ms / input_tokens / output_tokens
}
```

---

## 六、Handler 层

### 6.1 路由注册

在 `backend/internal/server/router.go`（或等效位置）添加：

```go
// [bmai-fork] audit routes
audit := admin.Group("/audit")
{
    audit.GET("/logs", auditHandler.List)
    audit.GET("/logs/:id", auditHandler.Get)
    audit.DELETE("/logs", auditHandler.BulkDelete)

    audit.GET("/stats/overview", auditStatsHandler.Overview)         // P1
    audit.GET("/stats/trend", auditStatsHandler.Trend)               // P1
    audit.GET("/stats/distribution", auditStatsHandler.Distribution) // P1
    audit.GET("/stats/users", auditStatsHandler.Users)               // P1
    audit.GET("/stats/departments", auditStatsHandler.Departments)   // P1

    audit.GET("/settings", auditSettingsHandler.Get)
    audit.PUT("/settings", auditSettingsHandler.Update)
    audit.GET("/storage", auditSettingsHandler.Storage)
    audit.POST("/cleanup", auditSettingsHandler.Cleanup)

    audit.GET("/rules", auditRuleHandler.List)             // P2
    audit.POST("/rules", auditRuleHandler.Create)          // P2
    audit.PUT("/rules/:id", auditRuleHandler.Update)       // P2
    audit.DELETE("/rules/:id", auditRuleHandler.Delete)    // P2
}

// [bmai-fork] organization routes
org := admin.Group("/organizations")
{
    org.GET("", organizationHandler.List)
    org.GET("/:id/departments", organizationHandler.DepartmentTree)
    org.GET("/:id/users", organizationHandler.Users)
    org.POST("/:id/sync-feishu", organizationHandler.SyncFromFeishu)
    org.GET("/departments/:id", organizationHandler.GetDepartment)
    org.PUT("/departments/:id", organizationHandler.UpdateDepartment)
    org.GET("/departments/:id/users", organizationHandler.DepartmentUsers)
}
```

---

## 七、API 规范

### 7.1 通用约定

- **路径前缀**：`/api/admin/audit/*`、`/api/admin/organizations/*`
- **认证**：管理员 JWT
- **响应格式**：复用现有 `ApiResponse` wrapper
- **错误码**：复用现有 error code
- **分页参数**：`page`（从 1 开始）、`page_size`（默认 50，最大 200）
- **时间格式**：RFC3339 ISO8601（`2026-05-03T14:22:00Z`）

### 7.2 GET /api/admin/audit/logs

**Query params**：

| 参数 | 类型 | 说明 |
|------|------|------|
| start_time | string | RFC3339，默认 `now - 24h` |
| end_time | string | RFC3339，默认 `now` |
| user_id | int64 | 多个用 `,` 分隔 |
| organization_id | int64 | |
| department_id | int64 | |
| department_path | string | 前缀匹配 |
| content_type | string | 多个用 `,` 分隔 |
| model | string | 多个用 `,` 分隔 |
| platform | string | |
| keyword | string | 模糊匹配 request/response preview |
| risk_level | string | low/medium/high |
| intercepted | bool | |
| page | int | 默认 1 |
| page_size | int | 默认 50，最大 200 |
| sort_by | string | created_at(默认)/duration_ms/input_tokens/output_tokens |
| sort_order | string | asc/desc(默认) |

**响应**（列表项不含完整 preview，只含 100 字符摘要）：

```json
{
  "code": 0,
  "data": {
    "items": [{
      "id": 12345,
      "request_id": "req_xxx",
      "user_id": 1,
      "user_email": "admin@xx.com",
      "department_path": "技术部/平台组",
      "content_type": "code",
      "model": "claude-3.5-sonnet",
      "platform": "anthropic",
      "endpoint": "/v1/messages",
      "request_summary": "前100字符...",
      "response_summary": "前100字符...",
      "request_bytes": 5000,
      "response_bytes": 12000,
      "input_tokens": 500,
      "output_tokens": 1200,
      "duration_ms": 1234,
      "stream": true,
      "status_code": 200,
      "risk_level": null,
      "intercepted": false,
      "created_at": "2026-05-03T14:22:00Z"
    }],
    "total": 12345,
    "page": 1,
    "page_size": 50
  }
}
```

### 7.3 GET /api/admin/audit/logs/:id

返回单条完整记录（含完整 preview，不截断到 100 字符）。

### 7.4 GET /api/admin/audit/settings

```json
{
  "code": 0,
  "data": {
    "enabled": true,
    "max_request_bytes": 32768,
    "max_response_bytes": 32768,
    "capture_upstream": false,
    "retention_days": 30,
    "classify_response": true
  }
}
```

### 7.5 GET /api/admin/audit/storage

```json
{
  "code": 0,
  "data": {
    "total_rows": 45000,
    "total_bytes": 1288490188,
    "earliest_record": "2026-04-15T00:00:00Z",
    "partitions": [
      {"name": "audit_logs_y2026m04", "rows": 30000, "bytes": 850000000},
      {"name": "audit_logs_y2026m05", "rows": 15000, "bytes": 438490188}
    ]
  }
}
```

### 7.6 GET /api/admin/audit/stats/trend (P1)

**Query**：`start_time`, `end_time`, `bucket=hour|day|week|month`, `dimension=none|user|department|organization|content_type|model|platform`, `organization_id`, `department_id`, `user_id`

**响应**：

```json
{
  "code": 0,
  "data": {
    "buckets": [{
      "bucket_start": "2026-05-03T14:00:00Z",
      "request_count": 120,
      "input_tokens": 50000,
      "output_tokens": 180000,
      "avg_duration_ms": 1100,
      "p95_duration_ms": 3200,
      "intercepted_count": 0,
      "risk_count": 2,
      "breakdown": {
        "code": 45,
        "conversation": 55,
        "image": 10,
        "tool_use": 10
      }
    }]
  }
}
```

---

## 八、Hook 集成点

### 8.1 修改 `gateway_service.go`

```go
// [bmai-fork] add response preview to ForwardResult
type ForwardResult struct {
    // ... 现有字段
    ResponseBodyPreview []byte  // 仅在审计开启时填充
    ResponseBodyBytes   int
}

// [bmai-fork] add optional capture buffer to streaming handler
func (s *GatewayService) handleStreamingResponse(
    ctx context.Context, resp *http.Response, c *gin.Context,
    account *Account, startTime time.Time,
    originalModel, mappedModel string, mimicClaudeCode bool,
    captureBuf *AuditCaptureBuffer,  // 新增参数，nil 表示不捕获
) (*streamingResult, error) {
    // ... 现有逻辑
    for scanner.Scan() {
        line := scanner.Bytes()
        // 写给客户端
        c.Writer.Write(line)
        c.Writer.Write([]byte("\n"))
        // [bmai-fork] tee 到捕获 buffer
        if captureBuf != nil {
            captureBuf.Write(line)
            captureBuf.Write([]byte{'\n'})
        }
        // ... 现有逻辑
    }
}

// [bmai-fork] streamingResult 加字段
type streamingResult struct {
    // 现有字段
    responsePreview []byte
    responseBytes   int
}
```

### 8.2 修改 `gateway_handler.go`

```go
// [bmai-fork] 在 Messages handler 中：
func (h *GatewayHandler) Messages(c *gin.Context) {
    // ... 现有逻辑：读取 body, 解析 ParsedRequest
    body, _ := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
    parsedReq, _ := service.ParseGatewayRequest(body, domain.PlatformAnthropic)

    // [bmai-fork] Pre-Audit Hook（P2 实现，P0 留空）
    if h.auditPreHook != nil {
        decision := h.auditPreHook.Evaluate(c, parsedReq)
        if decision.Block {
            // 写入 audit_log(intercepted=true)
            h.submitInterceptedAuditLog(parsedReq, decision)
            c.JSON(http.StatusForbidden, gin.H{"error": decision.Message})
            return
        }
    }

    // [bmai-fork] 申请 audit capture buffer（仅审计开启时）
    var captureBuf *service.AuditCaptureBuffer
    if h.auditService.Enabled() {
        captureBuf = service.AcquireAuditCaptureBuffer(h.auditService.MaxResponseBytes())
        defer service.ReleaseAuditCaptureBuffer(captureBuf)
    }

    // ... 现有路径：账号选择 → Forward
    result, err := h.gatewayService.Forward(ctx, ..., captureBuf)

    // [bmai-fork] 在 submitUsageRecordTask 之后追加审计写入
    h.submitUsageRecordTask(...)

    if h.auditService.Enabled() {
        h.submitAuditLogTask(c, parsedReq, result, captureBuf)
    }
}

// [bmai-fork] 构建并提交 AuditLog
func (h *GatewayHandler) submitAuditLogTask(
    c *gin.Context, parsedReq *service.ParsedRequest,
    result *service.ForwardResult, captureBuf *service.AuditCaptureBuffer,
) {
    user := getUserFromContext(c)
    log := &domain.AuditLog{
        RequestID:        result.RequestID,
        UserID:           user.ID,
        OrganizationID:   user.OrganizationID,
        DepartmentID:     user.PrimaryDepartmentID,
        DepartmentPath:   user.DepartmentPath,
        ContentType:      service.ClassifyContentType(...),
        RequestPreview:   string(truncate(parsedReq.Body, h.auditService.MaxRequestBytes())),
        RequestBytes:     len(parsedReq.Body),
        RequestTruncated: len(parsedReq.Body) > h.auditService.MaxRequestBytes(),
        ResponsePreview:  string(captureBuf.Bytes()),
        ResponseBytes:    captureBuf.Total,
        ResponseTruncated: captureBuf.Truncated(),
        Model:            result.Model,
        Platform:         account.Platform,
        Endpoint:         c.Request.URL.Path,
        Stream:           result.Stream,
        DurationMs:       int(result.Duration.Milliseconds()),
        InputTokens:      int(result.Usage.InputTokens),
        OutputTokens:     int(result.Usage.OutputTokens),
        StatusCode:       c.Writer.Status(),
        CreatedAt:        time.Now(),
    }
    // 内容类型修正（如果开启）
    if h.auditService.ClassifyResponse() {
        log.ContentType = service.RefineContentType(log.ContentType, captureBuf.Bytes())
    }
    h.auditService.Submit(c.Request.Context(), log)
}
```

---

## 九、配置

### 9.1 配置项

存储在现有 settings 表，key 前缀 `audit_`：

| Key | 类型 | 默认 | 说明 |
|-----|------|------|------|
| audit_enabled | bool | false | 总开关 |
| audit_max_request_bytes | int | 32768 | 请求 preview 上限（字节） |
| audit_max_response_bytes | int | 32768 | 响应 preview 上限（字节） |
| audit_capture_upstream | bool | false | 是否捕获上游请求/响应 |
| audit_retention_days | int | 30 | 保留天数 |
| audit_classify_response | bool | true | 是否做响应内容类型修正 |
| audit_pre_hook_enabled | bool | false | 请求前拦截 hook（P2） |
| audit_post_hook_enabled | bool | false | 响应后异步 hook（P2） |

### 9.2 内存缓存

启动时加载到 `AuditConfig` struct，settings API 修改时通过 setting_service 的 broadcast 机制刷新。

---

## 十、依赖注入与启动

### 10.1 wire.go 注册

```go
// [bmai-fork] register audit + organization services
wire.Build(
    // ... 现有
    NewAuditLogRepo,
    NewAuditLogService,
    NewOrganizationRepo,
    NewOrganizationService,
    NewFeishuOrgSyncService,
    NewAuditHandler,
    NewAuditSettingsHandler,
    NewOrganizationHandler,
    // P1
    NewAuditStatsRepo,
    NewAuditStatsService,
    NewAuditAggregationWorker,
    NewAuditStatsHandler,
    // P2
    NewAuditRuleRepo,
    NewAuditRuleEngine,
    NewAuditPreHook,
    NewAuditPostHook,
    NewAuditRuleHandler,
)
```

### 10.2 启动流程

`main.go`（或等效）：

```go
// [bmai-fork] start audit aggregation worker (P1)
if cfg.Audit.Enabled {
    go auditAggregationWorker.Run(ctx)
}

// [bmai-fork] start audit log cleanup worker
go auditCleanupWorker.Run(ctx)
```

---

## 十一、测试策略

### 11.1 单元测试

| 测试文件 | 覆盖 |
|---------|------|
| `audit_capture_test.go` | Buffer 写入、上限、Truncated 标记 |
| `audit_classify_test.go` | 各种输入下的分类正确性 + 响应修正 |
| `audit_log_repo_test.go` | List 多维度筛选、分页、排序 |
| `organization_service_test.go` | 部门树查询、用户-部门关联 |

### 11.2 集成测试

| 测试 | 覆盖 |
|------|------|
| `audit_e2e_test.go` | 发送请求 → 写入 → 查询 → 断言部门字段正确 |
| `audit_disabled_test.go` | 关闭审计后无新行写入 |
| `feishu_org_sync_test.go` | OAuth 登录后异步同步部门信息 |

### 11.3 验证脚本

```bash
# 部署后手动验证
# 1. 飞书登录验证
sudo docker exec sub2api-postgres psql -U sub2api -d sub2api -c \
  "select count(*) from organizations; select count(*) from departments; select count(*) from user_departments;"

# 2. 审计写入验证
sudo docker exec sub2api-postgres psql -U sub2api -d sub2api -c \
  "select count(*) from audit_logs where created_at > now() - interval '1 hour';"

# 3. 部门字段冗余验证
sudo docker exec sub2api-postgres psql -U sub2api -d sub2api -c \
  "select user_id, department_id, department_path from audit_logs limit 5;"
```
