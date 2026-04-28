# CLAUDE.md

## Current source of truth for development

### Branch strategy
- **`main` is now the active development mainline**, based on upstream `v0.1.119` with local customizations.
- Start new work from `main`.
- Old flat-import history preserved in `archive/main-import-v0.1.113`.

### Upstream merge strategy
- `origin` = `yqMac/sub2api`
- `upstream` = `Wei-Shaw/sub2api`
- This repo originally imported upstream as a flat baseline commit, so the **first** upstream history alignment must use:
  1. archive current `main`
  2. create a new branch from `upstream/main`
  3. cherry-pick local customization commits
- After `main` is eventually moved onto real upstream history, normal `git merge upstream/main` / `git rebase main` flows are fine.

## Deployment safety rules

### Production data source
Production data currently lives in **host directories**, not Docker named volumes:
- PostgreSQL: `/data/cc/sub2api/deploy/postgres_data`
- Redis: `/data/cc/sub2api/deploy/redis_data`
- App config/data: `/data/cc/sub2api/deploy/data`

### Critical warning
- **Do not use `deploy/docker-compose.yml` to cut production traffic right now.**
- That file uses Docker named volumes and host port mappings for postgres/redis.
- Using it can switch the app onto a new empty database and/or fail on host port conflicts.

### Current safe release method
For now, production releases should:
1. build a new image tag
2. run/update preview container `sub2api-test` on `8181`
3. validate via cookie preview on `https://aiapi.yqmac.com`
4. replace only the `sub2api` app container on `8180`
5. **do not recreate postgres/redis during normal app releases**

### Preview mechanism
- Enable preview: `https://aiapi.yqmac.com/__preview_enable`
- Disable preview: `https://aiapi.yqmac.com/__preview_disable`
- Browsers with cookie `preview_aiapi=1` go to `8181`
- Normal traffic goes to `8180`

## Required post-release validation
After any release, do not stop at HTTP 200. Verify:
- app page loads
- key UI path works
- users/accounts/api_keys counts are sane
- key admin user exists

Recommended DB checks:
```bash
sudo docker exec sub2api-postgres psql -U sub2api -d sub2api -Atc "select count(*) from users;"
sudo docker exec sub2api-postgres psql -U sub2api -d sub2api -Atc "select count(*) from accounts;"
sudo docker exec sub2api-postgres psql -U sub2api -d sub2api -Atc "select count(*) from api_keys;"
```

## Local customization markers
- Keep local fork-specific code marked with `[bmai-fork]` comments.
- Before/after upstream work, grep for them:
```bash
grep -R "\[bmai-fork\]" -n backend frontend
```

## Runbook
- Detailed operational procedure lives in: `deploy/OPERATIONS_UPGRADE_RUNBOOK.md`
- Read it before any future upgrade, release, rollback, or database/container recovery work.
