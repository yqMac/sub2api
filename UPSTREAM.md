# Upstream Tracking

## Upstream Repository
- **Repo**: [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- **Baseline Version**: v0.1.113
- **Fork**: [yqMac/sub2api](https://github.com/yqMac/sub2api)

## Custom Modifications (search `[bmai-fork]` in code)

### Modified upstream files (conflict surface on upgrade)
| File | Change |
|---|---|
| `backend/internal/service/ops_dashboard_models.go` | Added `AccountID *int64` to `OpsDashboardFilter` and `OpsDashboardOverview` |
| `backend/internal/handler/admin/ops_dashboard_handler.go` | Extracted `parseOpsDashboardFilter()` with `account_id` param; replaced 5 inline filter blocks |
| `backend/internal/repository/ops_repo_dashboard.go` | Added `account_id` WHERE clause in `buildUsageWhere()` and `buildErrorWhere()`; force raw mode when account_id set |
| `frontend/src/api/admin/ops.ts` | Added `account_id` param to 5 dashboard API functions |
| `frontend/src/views/admin/ops/OpsDashboard.vue` | Added `accountId` ref, query key, watcher, API param pass-through |
| `frontend/src/views/admin/ops/components/OpsDashboardHeader.vue` | Added account selector (prop, emit, Select dropdown, account list fetch) |

### Private migration number range
- Reserved: `9000+` (not used yet — existing `ops_error_logs` table already has all needed fields)

### Disabled upstream features
- (Planned) UI "one-click upgrade" button — to be disabled to prevent overwriting custom binary

## Upgrade Log

| Date | Upstream Version | Notes |
|---|---|---|
| 2026-04-27 | v0.1.113 | Initial baseline import + vendor account dashboard filter |
