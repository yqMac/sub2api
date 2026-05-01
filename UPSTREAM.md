# Upstream Tracking

## Upstream Repository
- **Repo**: [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- **Baseline Version**: v0.1.121
- **Fork**: [yqMac/sub2api](https://github.com/yqMac/sub2api)
- **Management Tool**: `/data/service-upgrade/sub2api/sub2api-manage.sh`

## Custom Modifications (search `[bmai-fork]` in code)

### Feature 1: Ops Dashboard Account Filter (bmai-1, 2026-04-27)
| File | Change |
|---|---|
| `backend/internal/service/ops_dashboard_models.go` | Added `AccountID *int64` to `OpsDashboardFilter` and `OpsDashboardOverview` |
| `backend/internal/handler/admin/ops_dashboard_handler.go` | Extracted `parseOpsDashboardFilter()` with `account_id` param |
| `backend/internal/handler/admin/ops_snapshot_v2_handler.go` | Snapshot-v2 reuses `parseOpsDashboardFilter()` |
| `backend/internal/repository/ops_repo_dashboard.go` | Added `account_id` WHERE clause; force raw mode when set |
| `frontend/src/api/admin/ops.ts` | Added `account_id` param to 5 dashboard API functions |
| `frontend/src/views/admin/ops/OpsDashboard.vue` | Added `accountId` ref, query key, watcher, API pass-through |
| `frontend/src/views/admin/ops/components/OpsDashboardHeader.vue` | Added account selector dropdown |

### Feature 2: Feishu OAuth Login (bmai-3, 2026-04-28 ~ 2026-04-30)
| File | Change |
|---|---|
| `backend/internal/handler/auth_feishu_oauth.go` | Feishu OAuth start/callback/complete handlers (new) |
| `backend/internal/handler/auth_feishu_oauth_test.go` | Feishu auth tests (new) |
| `backend/internal/service/auth_oauth_feishu.go` | Feishu OAuth service layer (new) |
| `backend/internal/service/auth_service.go` | Register feishu provider |
| `backend/internal/handler/auth_oauth_pending_flow.go` | Wire feishu into pending OAuth flow |
| `backend/internal/server/routes/auth.go` | Register feishu OAuth routes |
| `backend/internal/config/config.go` | FeishuConnectConfig struct + viper defaults + startup validation |
| `backend/internal/service/setting_service.go` | Expose feishu_oauth_enabled in public settings |
| `backend/internal/service/settings_view.go` | PublicSettings.FeishuOAuthEnabled field |
| `backend/internal/handler/dto/settings.go` | DTO FeishuOAuthEnabled field |
| `backend/internal/handler/setting_handler.go` | Settings handler adaptation |
| `backend/internal/service/domain_constants.go` | FeishuConnectSyntheticEmailDomain constant |
| `backend/ent/schema/user.go` | Allow feishu provider type |
| `backend/ent/schema/auth_identity.go` | Allow feishu provider type |
| `backend/migrations/134_extend_provider_type_feishu.sql` | Extend provider_type CHECK constraint (new) |
| `frontend/src/components/auth/FeishuOAuthSection.vue` | Feishu login button component (new) |
| `frontend/src/views/auth/LoginView.vue` | Integrate feishu button |
| `frontend/src/types/index.ts` | PublicSettings.feishu_oauth_enabled |
| `backend/internal/handler/auth_oidc_oauth.go` | Cleanup OIDC duplicates |
| `backend/internal/handler/auth_linuxdo_oauth.go` | Reference implementation |

### Environment Variable Dependencies
| Feature | Required Env Vars |
|---|---|
| Ops Dashboard Filter | (none) |
| Feishu OAuth | `FEISHU_CONNECT_ENABLED`, `FEISHU_CONNECT_CLIENT_ID`, `FEISHU_CONNECT_CLIENT_SECRET`, `FEISHU_CONNECT_REDIRECT_URL` |

### Private migration number range
- Reserved: `9000+` (not used yet)
- Current local migration: `134_extend_provider_type_feishu.sql`

### Disabled upstream features
- (Planned) UI "one-click upgrade" button — to be disabled to prevent overwriting custom binary

## Upgrade Log

| Date | From | To | Strategy | Notes |
|---|---|---|---|---|
| 2026-04-27 | — | v0.1.113 | flat-import | Initial baseline + ops dashboard filter |
| 2026-04-28 | v0.1.113 | v0.1.119 | cherry-pick onto upstream history | Migrated to real upstream git history |
| 2026-05-02 | v0.1.119 | v0.1.121 | git merge --no-ff | Real merge, 0 conflicts, VERSION fix, feishu .env fix |
