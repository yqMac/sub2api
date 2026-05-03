-- [bmai-fork] audit logs main table
-- Purpose: capture user request/response, upstream request/response along with
--          content classification and risk markers. Partitioned by month for
--          efficient retention management (DROP PARTITION for old months).

CREATE TABLE IF NOT EXISTS audit_logs (
    id                              BIGSERIAL,
    request_id                      VARCHAR(64) NOT NULL,
    user_id                         BIGINT NOT NULL,
    api_key_id                      BIGINT NOT NULL,
    account_id                      BIGINT NOT NULL,

    -- organization dimensions (denormalized at write time to avoid JOINs)
    organization_id                 BIGINT,
    department_id                   BIGINT,
    department_path                 VARCHAR(1000),

    -- content classification
    content_type                    VARCHAR(30),

    -- user original request
    request_preview                 TEXT,
    request_bytes                   INTEGER,
    request_truncated               BOOLEAN NOT NULL DEFAULT FALSE,

    -- response received by user
    response_preview                TEXT,
    response_bytes                  INTEGER,
    response_truncated              BOOLEAN NOT NULL DEFAULT FALSE,

    -- request sent to upstream (only populated when capture_upstream=true)
    upstream_request_preview        TEXT,
    upstream_request_bytes          INTEGER,
    upstream_request_truncated      BOOLEAN NOT NULL DEFAULT FALSE,

    -- response from upstream
    upstream_response_preview       TEXT,
    upstream_response_bytes         INTEGER,
    upstream_response_truncated     BOOLEAN NOT NULL DEFAULT FALSE,

    -- metadata
    model                           VARCHAR(100),
    platform                        VARCHAR(30),
    endpoint                        VARCHAR(200),
    stream                          BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms                     INTEGER,
    input_tokens                    INTEGER NOT NULL DEFAULT 0,
    output_tokens                   INTEGER NOT NULL DEFAULT 0,
    status_code                     INTEGER,

    -- interception / risk markers (populated by rule engine in P2)
    intercepted                     BOOLEAN NOT NULL DEFAULT FALSE,
    intercept_reason                VARCHAR(200),
    risk_level                      VARCHAR(20),
    risk_tags                       TEXT[],

    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- initial partitions: current month + next month
CREATE TABLE IF NOT EXISTS audit_logs_y2026m05 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS audit_logs_y2026m06 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- indexes (automatically applied to each partition in PG 11+)
CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_created ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_dept_created ON audit_logs(department_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_dept_path ON audit_logs(department_path varchar_pattern_ops, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_content_type ON audit_logs(content_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_model ON audit_logs(model, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_intercepted ON audit_logs(intercepted, created_at DESC) WHERE intercepted = TRUE;
