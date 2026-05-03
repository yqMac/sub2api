-- [bmai-fork] organization & department schema for audit dimensions
-- Purpose: capture Feishu organizational hierarchy (tenant + departments) so that
--          audit logs and statistics can be aggregated by organization/department.
--          P0 only creates the tables; automatic Feishu sync arrives in P1+.

-- 1. organizations: one row per Feishu tenant (or manually created org)
CREATE TABLE IF NOT EXISTS organizations (
    id              BIGSERIAL PRIMARY KEY,
    tenant_key      VARCHAR(100) NOT NULL UNIQUE,
    name            VARCHAR(200) NOT NULL,
    type            VARCHAR(30) NOT NULL DEFAULT 'feishu',
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT organizations_type_check CHECK (type IN ('feishu', 'manual', 'oidc'))
);

CREATE INDEX IF NOT EXISTS idx_organizations_type ON organizations(type) WHERE deleted_at IS NULL;

-- 2. departments: nested hierarchy under organization. full_path enables fast
--    prefix-match queries like "技术部/%" for department-level statistics.
CREATE TABLE IF NOT EXISTS departments (
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
    deleted_at          TIMESTAMPTZ,
    UNIQUE (organization_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_departments_org ON departments(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_departments_parent ON departments(parent_id);
CREATE INDEX IF NOT EXISTS idx_departments_path ON departments(full_path varchar_pattern_ops);

-- 3. user_departments: many-to-many user <-> department mapping.
--    A user can belong to multiple departments; is_primary flags the main one.
CREATE TABLE IF NOT EXISTS user_departments (
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id   BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    role            VARCHAR(50),
    employee_id     VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, department_id)
);

CREATE INDEX IF NOT EXISTS idx_user_departments_dept ON user_departments(department_id);
CREATE INDEX IF NOT EXISTS idx_user_departments_primary ON user_departments(user_id) WHERE is_primary = TRUE;

-- 4. denormalized columns on users table for fast filtering and audit-write
--    population. Updated by org-sync logic (P1+); writes are idempotent/optional.
ALTER TABLE users ADD COLUMN IF NOT EXISTS organization_id BIGINT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS primary_department_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_users_organization ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_primary_department ON users(primary_department_id);
