-- [bmai-fork] feishu: extend provider_type CHECK to include 'feishu'
-- The original constraint is in 110_pending_auth_and_provider_default_grants.sql

ALTER TABLE user_provider_default_grants
    DROP CONSTRAINT IF EXISTS user_provider_default_grants_provider_type_check;

ALTER TABLE user_provider_default_grants
    ADD CONSTRAINT user_provider_default_grants_provider_type_check
    CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc', 'feishu'));
