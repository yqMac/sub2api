-- [bmai-fork] feishu: extend pending_auth_sessions provider_type CHECK to include 'feishu'
-- The original constraint is in 110_pending_auth_and_provider_default_grants.sql

ALTER TABLE pending_auth_sessions
    DROP CONSTRAINT IF EXISTS pending_auth_sessions_provider_type_check;

ALTER TABLE pending_auth_sessions
    ADD CONSTRAINT pending_auth_sessions_provider_type_check
    CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc', 'feishu'));
