# Fork Management Standard

适用于所有基于开源项目二开的 fork 仓库（sub2api、skillhub 等）。

## 分支策略

```
upstream/main (官方仓库)
     │
     ▼
   main ← 纯净上游镜像，只允许 fast-forward merge upstream/main
     │
     ▼
  develop ← 所有二开代码的唯一主线，发版从这里出
     │
     ├── feat/*   ← 新功能，完成后合并回 develop
     ├── fix/*    ← 修复，完成后合并回 develop
     └── hotfix/* ← 紧急修复，完成后合并回 develop
```

### 核心规则

1. **main 只跟踪上游** — 零自定义代码，只做 `git fetch upstream && git merge upstream/main`
2. **develop 是二开主线** — 所有自定义功能在此，所有发版镜像从此构建
3. **上游更新单向流动** — upstream → main → develop，永远不反向
4. **冲突只在 develop 解决** — main 永远是 fast-forward，无冲突风险

### 上游同步流程

```bash
# 1. 同步 main（无冲突，fast-forward）
git fetch upstream
git checkout main
git merge upstream/main

# 2. 合并到 develop（冲突在这里解决）
git checkout develop
git merge main

# 3. 验证二开标记完整性
grep -R "\[bmai-fork\]" -n backend frontend

# 4. 构建、测试、发版
```

### 版本号规范

```
<上游版本>-bmai.<迭代号>
例: 0.1.121-bmai.4
```

- 上游版本跟随 upstream tag
- bmai 迭代号在每次二开发版时递增
- Docker 镜像 tag 与版本号一致：`sub2api-bmai:0.1.121-bmai.4`

## 二开标记规范

### 内联标记

对上游文件的修改必须用 `[bmai-fork]` 注释标记：

```go
// [bmai-fork] 添加飞书 OAuth 支持
if config.FeishuConnect.Enabled {
    registerFeishuProvider(...)
}
```

```typescript
// [bmai-fork] ops dashboard account filter
const accountId = ref<number | null>(null)
```

### Commit Message

二开提交在末尾加 `[bmai-fork]` 标签：

```
feat(auth): add feishu oauth login [bmai-fork]
fix(ops): dashboard account filter not applied [bmai-fork]
```

### 新增文件

完全新增的文件（非修改上游文件）在文件头部加标记：

```go
// [bmai-fork] 飞书 OAuth 认证服务
package service
```

## 必备文档

每个二开项目必须维护以下文件：

| 文件 | 位置 | 用途 |
|------|------|------|
| `UPSTREAM.md` | 项目根目录 | 上游信息、当前基线版本、自定义文件索引 |
| `FORK-CHANGELOG.md` | 项目根目录 | 每次二开迭代的变更记录 |
| `OPERATIONS_UPGRADE_RUNBOOK.md` | deploy/ | 发版/升级/回滚操作手册 |

## Docker Compose 规范

### 文件结构（override 模式）

```
deploy/
├── docker-compose.yml          # 基础定义（所有环境共用）
├── docker-compose.override.yml # 生产环境（自动加载）
├── docker-compose.dev.yml      # 开发环境（显式指定）
└── .env                        # 环境变量
```

### 安全原则

1. **数据库不暴露端口** — postgres/redis 默认只在内部网络可达
2. **使用 host bind mount** — 不用 named volume，数据可见可迁移
3. **preview 用 profile 隔离** — `profiles: ["preview"]`，默认不启动
4. **只更新 app 容器** — `docker compose up -d sub2api`，不动数据库
5. **env_file 管理敏感配置** — 不在 compose 文件中硬编码密钥

### 发版命令

```bash
# 生产发版（只更新 app）
VERSION=0.1.121-bmai.4 docker compose up -d sub2api

# 启动 preview
PREVIEW_VERSION=0.1.122-bmai.1 docker compose --profile preview up -d sub2api-test

# 开发环境
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

## 分支清理规则

- 已合并到 develop 的 feat/fix 分支：合并后立即删除
- upgrade 分支：合并完成后删除，只保留 tag
- archive 分支：保留，用于历史追溯
- worktree 残留分支：确认无用后删除
