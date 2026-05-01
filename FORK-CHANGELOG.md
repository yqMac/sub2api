# Fork Changelog

本文件记录 yqMac/sub2api 相对于上游 Wei-Shaw/sub2api 的所有本地变更。
随代码提交到 git，作为 fork 迭代的版本内记录。

查找所有本地改动：`grep -rn '[bmai-fork]' backend frontend --include='*.go' --include='*.ts' --include='*.vue' --include='*.sql'`

---

## [bmai-4] - 2026-05-02 — 升级到 upstream v0.1.121

**上游基线**: v0.1.119 → v0.1.121

### 本地变更
- chore: 真实 merge upstream v0.1.121（`4a4055eb`）
- chore: 修正 VERSION 文件为 0.1.121（上游漏更新）（`ea24dafe`）

### 上游新增（v0.1.120 + v0.1.121）
- Vertex AI Service Account 支持
- OpenAI Fast/Flex Policy（service_tier 路由）
- Anthropic 缓存 TTL 1h 注入开关
- 账号批量编辑增强
- 调度器竞态修复、stream EOF failover
- 分页 localStorage 持久化恢复

### 合并结果
- 5 个文件有冲突风险，git 自动合并成功
- 26 个 `[bmai-fork]` 文件全部保留
- migration 134 无冲突（上游最高 133）

---

## [bmai-3] - 2026-04-28 ~ 2026-04-30 — 飞书 OAuth 登录

**上游基线**: v0.1.119

### 新增功能
飞书（Feishu/Lark）OAuth 登录，允许企业用户通过飞书账号登录。

### 新增文件
- `backend/internal/handler/auth_feishu_oauth.go` — 飞书 OAuth 处理器
- `backend/internal/handler/auth_feishu_oauth_test.go` — 测试
- `backend/internal/service/auth_oauth_feishu.go` — 服务层
- `backend/migrations/134_extend_provider_type_feishu.sql` — DB 迁移
- `frontend/src/components/auth/FeishuOAuthSection.vue` — 登录按钮组件

### 修改文件（19 个，均标记 `[bmai-fork]`）
- `backend/internal/config/config.go` — FeishuConnectConfig + viper defaults
- `backend/internal/service/setting_service.go` — 暴露 feishu_oauth_enabled
- `backend/internal/service/settings_view.go` — PublicSettings 字段
- `backend/internal/service/domain_constants.go` — 合成邮箱域名常量
- `backend/internal/service/auth_service.go` — 注册 feishu provider
- `backend/internal/handler/dto/settings.go` — DTO 字段
- `backend/internal/handler/auth_oauth_pending_flow.go` — pending 流程
- `backend/internal/handler/setting_handler.go` — 设置处理器
- `backend/internal/server/routes/auth.go` — 路由注册
- `backend/ent/schema/user.go` — provider type 枚举
- `backend/ent/schema/auth_identity.go` — provider type 枚举
- `frontend/src/views/auth/LoginView.vue` — 集成飞书按钮
- `frontend/src/types/index.ts` — TypeScript 类型

### 环境变量
```
FEISHU_CONNECT_ENABLED=true
FEISHU_CONNECT_CLIENT_ID=<App ID>
FEISHU_CONNECT_CLIENT_SECRET=<App Secret>
FEISHU_CONNECT_REDIRECT_URL=https://aiapi.yqmac.com/api/v1/auth/oauth/feishu/callback
FEISHU_CONNECT_ALLOWED_TENANT_KEYS=<tenant_key>
FEISHU_CONNECT_REQUIRE_ENTERPRISE_EMAIL=false
FEISHU_CONNECT_AUTO_BIND_OR_CREATE=true
```

---

## [bmai-2] - 2026-04-28 — 迁移到真实上游历史

**上游基线**: v0.1.113 → v0.1.119

### 变更
- 从 flat-import 迁移到真实上游 git 历史
- 归档旧 main 到 `archive/main-import-v0.1.113`
- Cherry-pick 本地 4 个提交到 upstream/main 分支（0 冲突）

---

## [bmai-1] - 2026-04-27 — 初始导入 + Ops Dashboard 账户筛选

**上游基线**: v0.1.113

### 新增功能
运维仪表盘按上游 provider 账户筛选。

### 修改文件（7 个，均标记 `[bmai-fork]`）
- `backend/internal/service/ops_dashboard_models.go`
- `backend/internal/handler/admin/ops_dashboard_handler.go`
- `backend/internal/handler/admin/ops_snapshot_v2_handler.go`
- `backend/internal/repository/ops_repo_dashboard.go`
- `frontend/src/api/admin/ops.ts`
- `frontend/src/views/admin/ops/OpsDashboard.vue`
- `frontend/src/views/admin/ops/components/OpsDashboardHeader.vue`
