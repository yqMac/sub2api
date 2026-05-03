# 审计功能 编码与协作规范

> **范围**：sub2api 二开（[bmai-fork]），本规范在审计/告警/拦截功能开发期间生效，长期可作为后续二开通用规范
> **相关文档**：[PLAN.md](./PLAN.md) · [DESIGN-BACKEND.md](./DESIGN-BACKEND.md) · [DESIGN-FRONTEND.md](./DESIGN-FRONTEND.md) · [../FORK-MANAGEMENT-STANDARD.md](../FORK-MANAGEMENT-STANDARD.md)

## 一、二开标记规范

### 1.1 内联标记

所有对上游文件的修改必须用 `[bmai-fork]` 注释标记。

**Go**：
```go
// [bmai-fork] add response preview to ForwardResult for audit capture
type ForwardResult struct {
    // ... 现有字段
    ResponseBodyPreview []byte
}
```

**TypeScript / Vue**：
```typescript
// [bmai-fork] audit content type filter
const contentType = ref<AuditContentType[]>([])
```

```vue
<!-- [bmai-fork] add audit menu item -->
<MenuItem :to="/admin/audit" :icon="ClipboardList">审计日志</MenuItem>
```

**SQL**：
```sql
-- [bmai-fork] audit_logs table
CREATE TABLE audit_logs (...)
```

### 1.2 新增文件标记

新增文件在文件头部加标记：

```go
// [bmai-fork] audit log service
package service
```

```vue
<!-- [bmai-fork] audit log list view -->
<template>...</template>
```

### 1.3 Commit Message 标记

所有审计相关 commit 末尾加 `[bmai-fork]`：

```
feat(audit): add audit_logs table and migration [bmai-fork]
feat(audit-fe): add audit log list page [bmai-fork]
fix(audit): handle nil capture buffer in non-streaming path [bmai-fork]
```

## 二、分支策略

遵循项目根 `docs/FORK-MANAGEMENT-STANDARD.md`：

```
develop ← 审计功能主线
  ├─ feat/audit-p0-schema       ← 数据库 schema
  ├─ feat/audit-p0-backend      ← 后端审计写入
  ├─ feat/audit-p0-frontend     ← 前端列表/详情/设置
  ├─ feat/audit-p0-organization ← 组织架构
  ├─ feat/audit-p1-stats        ← 统计聚合
  ├─ feat/audit-p2-rules        ← 规则引擎
  └─ feat/audit-p3-quota        ← 部门配额
```

每个 feat 分支完成后：
1. PR 标题：`feat(audit): <description> [bmai-fork]`
2. PR 描述包含：变更摘要、测试说明、相关 PLAN 章节
3. 合并方式：`--no-ff` 保留分支历史
4. 合并后立即删除分支

## 三、目录组织规范

### 3.1 后端

| 类型 | 目录 | 命名 |
|------|------|------|
| 迁移 | `backend/migrations/` | `<num>_<verb>_<noun>.sql`，例 `135_create_organizations.sql` |
| 领域模型 | `backend/internal/domain/` | 单一实体一个文件，例 `audit.go`、`organization.go` |
| 服务 | `backend/internal/service/` | `<feature>_<role>.go`，例 `audit_log.go`, `audit_classify.go`, `audit_capture.go` |
| 仓储 | `backend/internal/repository/` | `<entity>_repo.go`，例 `audit_log_repo.go` |
| Handler | `backend/internal/handler/admin/` | `<feature>_handler.go`，例 `audit_handler.go`, `audit_settings_handler.go` |
| 测试 | 与被测文件同目录 | `<file>_test.go` |

### 3.2 前端

| 类型 | 目录 | 命名 |
|------|------|------|
| API 模块 | `frontend/src/api/admin/` | `<feature>.ts`，例 `audit.ts`, `organizations.ts` |
| 视图（页面） | `frontend/src/views/admin/` | `<Feature>View.vue`（顶级），子页面 `<feature>/<Subpage>.vue` |
| 组件 | `frontend/src/views/<page>/components/` | `<Feature><Component>.vue` |
| 通用组件 | `frontend/src/components/<category>/` | 仅当跨页面复用时放这里 |
| Composable | `frontend/src/composables/` | `use<Feature>.ts`，例 `useAuditFilters.ts` |
| Pinia store | `frontend/src/stores/` | `<feature>.ts`，例 `audit.ts` |
| i18n | `frontend/src/i18n/locales/` | `zh.ts` / `en.ts` 同步更新 |

## 四、命名规范

### 4.1 Go

- 包名：小写、单词、不缩写
- 类型：`PascalCase`，例 `AuditLog`, `OrganizationService`
- 接口：以 `-er` 或职责命名，例 `AuditLogRepository`, `Classifier`
- 常量：`PascalCase`，例 `ContentTypeCode`, `RiskLevelHigh`
- 私有函数/字段：`camelCase`
- 错误：`Err` 前缀，例 `ErrAuditDisabled`

### 4.2 TypeScript / Vue

- 类型/接口：`PascalCase`，例 `AuditLogListItem`, `AuditSettings`
- 组件文件名：`PascalCase.vue`，例 `AuditDetailDialog.vue`
- Composable：`use` 前缀，例 `useAuditFilters`
- 常量：`SCREAMING_SNAKE_CASE`，例 `MAX_PREVIEW_BYTES`
- API 方法：`camelCase`，例 `auditAPI.list`, `auditAPI.getSettings`
- Pinia store：`use<Name>Store`，例 `useAuditStore`

### 4.3 数据库

- 表名：`snake_case`、复数，例 `audit_logs`, `departments`
- 列名：`snake_case`，例 `created_at`, `request_preview`
- 索引：`idx_<table>_<columns>`，唯一索引 `uk_<table>_<columns>`
- 外键：列名为 `<entity>_id`，例 `user_id`, `department_id`
- 时间戳：所有表必须有 `created_at TIMESTAMPTZ`，可变实体加 `updated_at`

### 4.4 API

- 路径：`kebab-case`，例 `/api/admin/audit/logs`
- Query 参数：`snake_case`，例 `start_time`, `page_size`
- JSON 字段：`snake_case`，与数据库一致
- 资源命名复数：`/audit/logs`、`/organizations/:id/departments`

## 五、代码风格

### 5.1 Go

- `gofmt` + `goimports` 自动格式化
- 错误处理：永不忽略，返回上层或 log + 降级
- 上下文：所有 service/repo 方法第一个参数必须是 `context.Context`
- 不要使用全局变量，依赖通过 wire 注入
- 接口定义在使用方包，实现在被使用方包（依赖倒置）

### 5.2 TypeScript

- ESLint + Prettier 项目现有配置
- 严格模式（`strict: true`）
- 不使用 `any`，必要时用 `unknown` 加 type guard
- 异步：用 `async/await`，不混用 `.then()`
- 组件 props 必须有类型声明
- 使用 `<script setup>` 而非 Options API

### 5.3 SQL

- 关键字大写：`SELECT`, `FROM`, `WHERE`
- 列名/表名小写
- 复杂查询用 CTE 而非嵌套子查询
- 必须显式指定列，不用 `SELECT *`
- 索引必须有 `idx_` 前缀

## 六、测试规范

### 6.1 后端

| 类型 | 要求 |
|------|------|
| 单元测试 | 服务层、分类器、buffer 等纯函数必须有覆盖 |
| 集成测试 | Handler + Repo 配合用 testdb 验证 |
| Repo 测试 | 用 testcontainers 或 docker-compose 起真实 PG，不用 mock |
| 命名 | `Test<Func>_<Scenario>`，例 `TestClassifyContentType_Image` |
| 表驱动 | 多场景用 `tt := []struct{...}` 表驱动 |

### 6.2 前端

| 类型 | 要求 |
|------|------|
| 组件测试 | 复杂组件（详情弹窗、规则编辑器）必须有 vitest 测试 |
| Composable 测试 | URL 同步、防抖等逻辑必须测 |
| 命名 | `<file>.spec.ts` |
| Mock | API 用 `vi.mock` 替换，不发真实请求 |

### 6.3 验证策略

每个 PR 合并前：
1. 单元测试通过
2. 后端集成测试通过
3. 前端 typecheck + lint 通过
4. 手动验证清单（见 PLAN.md 的"验证清单"）

## 七、Commit 规范

遵循 Conventional Commits + bmai-fork 标记。

### 7.1 格式

```
<type>(<scope>): <description> [bmai-fork]

[optional body]

[optional footer]
```

### 7.2 Type

| Type | 用途 |
|------|------|
| feat | 新功能 |
| fix | bug 修复 |
| refactor | 重构（不改行为） |
| perf | 性能优化 |
| test | 仅测试 |
| docs | 仅文档 |
| chore | 构建、配置、依赖等 |
| db | 数据库迁移 |

### 7.3 Scope

| Scope | 含义 |
|-------|------|
| audit | 审计后端 |
| audit-fe | 审计前端 |
| audit-rule | 规则引擎 |
| audit-stats | 统计聚合 |
| organization | 组织/部门后端 |
| organization-fe | 组织/部门前端 |
| feishu | 飞书集成 |

### 7.4 示例

```
feat(audit): add audit_logs table and migration [bmai-fork]
feat(audit): wire audit capture into streaming response [bmai-fork]
feat(audit-fe): add audit log list page [bmai-fork]
feat(audit-fe): add detail dialog with 2x2 comparison [bmai-fork]
feat(organization): add department tree query [bmai-fork]
feat(feishu): sync user departments after OAuth login [bmai-fork]
fix(audit): handle nil capture buffer for non-streaming path [bmai-fork]
db(audit): add initial monthly partition for audit_logs [bmai-fork]
test(audit): add classify content type unit tests [bmai-fork]
docs(audit): update DESIGN-BACKEND with hook integration [bmai-fork]
```

## 八、配置与开关

### 8.1 配置存储

- 运行时配置存数据库 `system_settings` 表，key 前缀 `audit_`
- 启动时读入内存（AuditConfig struct）
- 修改后通过 setting_service 的 broadcast 机制热刷新
- 默认值在 settings 初始化迁移中插入

### 8.2 开关原则

- 总开关：默认 `false`（关闭），通过 UI 开启
- 子开关：默认按"最小影响"原则
  - capture_upstream: false（不捕获上游）
  - classify_response: true（轻量响应分类）
  - pre_hook_enabled: false（不启用拦截）
  - post_hook_enabled: false（不启用异步告警）
- 所有开关都可热切换，不需要重启

### 8.3 灰度策略

P2 规则上线时：
1. 先用 `warn` 模式（不阻断，只记录）
2. 观察一周命中数据
3. 切换为 `block` 模式

## 九、错误处理

### 9.1 后端

- 审计写入失败：log + 不影响请求
- 部门同步失败：log + 标记下次重试
- 规则匹配失败：log + 默认放行（fail-open）
- 配置读取失败：log + 用默认值（关闭）

### 9.2 前端

- API 调用失败：toast 错误提示 + 不阻塞页面
- 长列表加载失败：显示重试按钮
- 详情加载失败：弹窗内显示错误

## 十、性能要求

| 场景 | 要求 |
|------|------|
| 审计开启时单请求延迟增加 | < 1ms |
| 列表查询（24h，分页 50） | < 200ms |
| 详情查询 | < 100ms |
| 统计查询（30 天预聚合） | < 500ms |
| 规则匹配（5 条规则） | < 1ms（内存） |
| 部门同步（单用户） | < 5s（异步，不阻塞登录） |

测试方法：
- 后端：用 `go test -bench` 验证
- 前端：用 Chrome DevTools Performance 验证

## 十一、安全要求

### 11.1 数据脱敏

- 不记录请求/响应中的明文密码、API key（如检测到 `password=`、`Authorization: Bearer xxx` 模式时打码）
- TOTP 密钥永不入审计
- 审计日志的访问需要管理员权限
- 详情接口返回时应用脱敏规则（P2 增强）

### 11.2 访问控制

- 所有 `/api/admin/audit/*` 路由必须经过 admin auth middleware
- 普通用户不可访问审计数据
- 部门级查看权限（P3）：管理员可看全部，部门负责人只能看本部门

### 11.3 SQL 注入

- 所有查询必须用参数化绑定（`?` 或 `$1`）
- LIKE 模糊匹配必须 escape `%` 和 `_`
- 多值筛选用 `ANY()` 或 `IN ($1, $2, ...)`，不拼接字符串

## 十二、文档要求

### 12.1 必须维护的文档

| 文档 | 触发更新 |
|------|---------|
| `docs/audit/PLAN.md` | 整体方案变更时 |
| `docs/audit/DESIGN-BACKEND.md` | 后端架构变更时 |
| `docs/audit/DESIGN-FRONTEND.md` | 前端架构变更时 |
| `docs/audit/CONVENTIONS.md` | 规范变更时 |
| `FORK-CHANGELOG.md` | 每个 PR 合并后 |
| `UPSTREAM.md` | 自定义文件清单变更时 |

### 12.2 代码注释

- 复杂算法必须有注释说明思路
- 公开函数/类型有 godoc 注释
- 不要写"这段代码做什么"，要写"为什么这样做"
- TODO/FIXME 必须有 issue 编号或说明

## 十三、联调与集成

### 13.1 后端先行

P0 阶段建议顺序：
1. 后端：建表 + 仓储层 + handler 框架
2. 后端：写入逻辑（先做非流式，再做流式）
3. 后端：测试 + 集成测试
4. 前端：API 联调（用 mock 数据先过 UI）
5. 前端 ↔ 后端：联调
6. 端到端验证

### 13.2 接口冻结

后端 API 设计完成后（在 DESIGN-BACKEND.md 中），前端开始时不允许后端单方面改路径或字段名。如需变更，先改文档 → 双方同步。

### 13.3 Mock 数据

前端开发期可用 `frontend/src/mocks/audit.ts` 提供 mock 数据，部署到生产前删除。

## 十四、版本与发布

每个 P 阶段（P0/P1/P2/P3）作为一个发布版本：

| 阶段 | 版本号 | 说明 |
|------|--------|------|
| P0 完成 | `0.1.121-bmai.5` | 审计基础 + 组织架构 |
| P1 完成 | `0.1.121-bmai.6` | 统计聚合 |
| P2 完成 | `0.1.121-bmai.7` | 规则引擎 |
| P3 完成 | `0.1.121-bmai.8` | 配额/导出 |

每次发布：
1. 更新 `VERSION` 文件
2. 更新 `FORK-CHANGELOG.md`（添加 bmai-N 条目）
3. 构建镜像 `sub2api-bmai:<version>`
4. 按 `OPERATIONS_UPGRADE_RUNBOOK.md` 流程灰度上线
