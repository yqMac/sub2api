-- [bmai-fork] audit rules for interception and alerting
-- Purpose: P0 creates the schema only; rule engine implementation arrives in P2.
--          Rules define match conditions and actions (block/warn/alert/tag) that
--          are evaluated at request-time (pre_block) or response-time (post_alert/post_tag).

CREATE TABLE IF NOT EXISTS audit_rules (
    id                      BIGSERIAL PRIMARY KEY,
    name                    VARCHAR(200) NOT NULL,
    description             TEXT,
    enabled                 BOOLEAN NOT NULL DEFAULT TRUE,
    priority                INTEGER NOT NULL DEFAULT 100,

    -- rule_type determines when the rule is evaluated
    rule_type               VARCHAR(30) NOT NULL,
    CONSTRAINT audit_rules_type_check CHECK (rule_type IN ('pre_block', 'post_alert', 'post_tag')),

    -- scope: which requests this rule applies to (empty array = all)
    scope_organization_ids  BIGINT[],
    scope_department_ids    BIGINT[],
    scope_user_ids          BIGINT[],
    scope_models            TEXT[],
    scope_content_types     TEXT[],

    -- match conditions (flexible JSONB for extensibility)
    -- examples:
    --   {"keyword": {"in_field": "request", "patterns": ["password", "secret"]}}
    --   {"frequency": {"window_seconds": 60, "max_count": 100}}
    --   {"size": {"in_field": "request", "max_bytes": 1048576}}
    match_conditions        JSONB NOT NULL DEFAULT '{}',

    -- action to take when rule matches
    action                  VARCHAR(30) NOT NULL,
    CONSTRAINT audit_rules_action_check CHECK (action IN ('block', 'warn', 'alert', 'log', 'tag')),

    -- action-specific configuration
    -- examples:
    --   {"block_message": "请勿在请求中包含密码"}
    --   {"alert_severity": "high", "alert_recipients": ["admin@example.com"]}
    --   {"tags": ["sensitive_data"]}
    action_config           JSONB,

    created_by              BIGINT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_rules_enabled ON audit_rules(enabled, priority) WHERE enabled = TRUE;

-- audit_rule_events: one row per rule hit. Partitioned by month for retention.
CREATE TABLE IF NOT EXISTS audit_rule_events (
    id              BIGSERIAL,
    rule_id         BIGINT NOT NULL,
    audit_log_id    BIGINT,
    request_id      VARCHAR(64),
    user_id         BIGINT,
    department_id   BIGINT,
    organization_id BIGINT,
    action_taken    VARCHAR(30) NOT NULL,
    matched_field   VARCHAR(50),
    matched_value   TEXT,
    severity        VARCHAR(20),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS audit_rule_events_y2026m05 PARTITION OF audit_rule_events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS audit_rule_events_y2026m06 PARTITION OF audit_rule_events
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE INDEX IF NOT EXISTS idx_audit_rule_events_rule ON audit_rule_events(rule_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_rule_events_user ON audit_rule_events(user_id, created_at DESC);
