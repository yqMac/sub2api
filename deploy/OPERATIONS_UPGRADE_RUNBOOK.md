# Sub2API 升级 / 二开 / 发版操作手册

## 1. 目标

本文档用于指导后续对 `yqMac/sub2api` 的二次开发、合并上游、发版、回滚，以及如何在当前部署方式下尽量做到"无感"验证与低风险切换。

适用当前环境：
- 代码仓库：`git@github.com:yqMac/sub2api.git`
- 上游仓库：`https://github.com/Wei-Shaw/sub2api.git`
- 当前正式域名：`https://aiapi.yqmac.com`
- 正式容器：`sub2api`
- 预览容器：`sub2api-test`
- 正式端口：`8180`
- 预览端口：`8181`
- 反代容器：`nginx`（OpenResty）

---

## 2. 当前生产拓扑

### 2.1 当前有效拓扑

```text
浏览器
  -> https://aiapi.yqmac.com
  -> OpenResty/nginx 容器
  -> 宿主机 8180（正式） / 8181（预览，按 cookie 切换）
  -> sub2api 容器 / sub2api-test 容器
  -> deploy_sub2api-network
  -> sub2api-postgres / sub2api-redis
```

### 2.2 预览机制

当前已配置基于 Cookie 的灰度预览：

- 开启预览：`https://aiapi.yqmac.com/__preview_enable`
- 关闭预览：`https://aiapi.yqmac.com/__preview_disable`

行为：
- 带 `preview_aiapi=1` Cookie 的浏览器流量会走 `8181` 预览容器
- 其他用户继续走 `8180` 正式容器
- 这是一种低风险的 UI 验证方式，不影响其他用户

---

## 3. 当前部署方式的关键结论

### 3.1 当前不要直接用 `docker compose up -d`

**结论：当前生产环境不应直接用 `deploy/docker-compose.yml` 整体重建。**

原因：
- 该 compose 文件里把 `postgres` 和 `redis` 都映射到了宿主机固定端口：
  - `5432:5432`
  - `6379:6379`
- 当前机器上这些端口已被别的服务占用
- 因此执行：
  ```bash
  docker compose up -d --force-recreate sub2api
  ```
  会连带触发依赖重建，并在 `redis/postgres` 绑定端口时失败，导致正式应用容器无法启动

### 3.2 当前推荐的发版方式

**当前推荐：仅用 `docker run` 管理正式 `sub2api` / `sub2api-postgres` / `sub2api-redis`。**

原因：
- 现在已经按这种方式恢复成功
- 可避免 compose 的端口冲突问题
- 可明确区分：验证阶段、正式切换阶段、回滚阶段

---

## 4. 代码分支策略（后续二开 / 合并上游）

### 4.1 远程仓库约定

- `origin`：`yqMac/sub2api`（自己的 Fork，只 push 这里）
- `upstream`：`Wei-Shaw/sub2api`（只 fetch，不 push）

检查：
```bash
git remote -v
```

### 4.2 推荐分支模型

- `main`：自己的稳定主线
- `feat/<name>`：功能分支
- `hotfix/<name>`：线上修复分支

### 4.3 日常开发流程

```bash
# 拉上游
git fetch upstream

# 更新自己的 main
git checkout main
git merge upstream/main

# 推送到自己的仓库
git push origin main

# 开功能分支
git checkout -b feat/vendor-dashboard
```

### 4.4 自定义改动标记规则

所有对上游文件的直接修改，统一加：

```go
// [bmai-fork]
```

或前端：

```html
<!-- [bmai-fork] -->
```

作用：
- 方便后续和上游 rebase / merge 时快速定位冲突点
- 方便代码审查时识别"本地增强"和"上游原始逻辑"

---

## 5. 发版前检查清单

每次发版前至少确认：

- [ ] 代码已提交到 `origin` 对应分支
- [ ] 镜像已能本地构建成功
- [ ] 预览容器能正常启动
- [ ] 使用 `__preview_enable` 做过外部 UI 验证
- [ ] 核心筛选/调用/登录功能已验证
- [ ] 已记录当前正式镜像 tag，便于回滚

可选检查：
```bash
git status
git log --oneline -5
sudo docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"
```

---

## 6. 标准发版流程（当前推荐）

## 6.1 构建镜像

在仓库根目录：

```bash
cd /data/cc/sub2api
sudo docker build -t sub2api-bmai:<版本号> .
```

示例：
```bash
sudo docker build -t sub2api-bmai:0.1.113-bmai.2 .
```

建议版本命名：
- `上游版本-bmai.序号`
- 例如：`0.1.113-bmai.3`

---

## 6.2 启动预览容器（8181）

```bash
sudo docker rm -f sub2api-test 2>/dev/null || true

sudo docker run -d \
  --name sub2api-test \
  --restart unless-stopped \
  --network deploy_sub2api-network \
  -p 8181:8080 \
  -v /data/cc/sub2api/deploy/data:/app/data \
  -e AUTO_SETUP=true \
  -e SERVER_HOST=0.0.0.0 \
  -e SERVER_PORT=8080 \
  -e SERVER_MODE=release \
  -e RUN_MODE=standard \
  -e DATABASE_HOST=postgres \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=sub2api \
  -e DATABASE_PASSWORD=7e8a5db0d468f01455947ad680c29fd0 \
  -e DATABASE_DBNAME=sub2api \
  -e DATABASE_SSLMODE=disable \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD= \
  -e REDIS_DB=0 \
  -e REDIS_ENABLE_TLS=false \
  -e JWT_SECRET=cd0f502c111b88d767e2bc50728946e3f877d5dd1f01d0060c309d0983a54c40 \
  -e TOTP_ENCRYPTION_KEY=8e8c8e2a5bf3ab179b92bed229a3a10f4f918a5d09ae4f16716a8c60dd446060 \
  -e TZ=Asia/Shanghai \
  sub2api-bmai:<版本号>
```

检查：
```bash
sudo docker ps --filter name=sub2api-test
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8181/
```

---

## 6.3 外部预览验证

浏览器访问：

1. `https://aiapi.yqmac.com/__preview_disable`
2. `https://aiapi.yqmac.com/__preview_enable`

然后刷新正式域名页面，当前浏览器就会自动走 `8181` 的预览容器。

建议验证：
- 登录是否正常
- 管理后台是否正常
- 运维仪表盘是否正常
- 本次改动功能是否符合预期
- 普通用户主链路是否正常

验证完成后，可关闭预览：
- `https://aiapi.yqmac.com/__preview_disable`

---

## 6.4 正式切换（当前环境下的推荐方式）

**不要用 compose 重建。**

记录当前正式镜像：
```bash
sudo docker inspect sub2api --format '{{.Config.Image}}'
```

正式切换：
```bash
sudo docker rm -f sub2api

sudo docker run -d \
  --name sub2api \
  --restart unless-stopped \
  --network deploy_sub2api-network \
  -p 8180:8080 \
  -v /data/cc/sub2api/deploy/data:/app/data \
  -e AUTO_SETUP=true \
  -e SERVER_HOST=0.0.0.0 \
  -e SERVER_PORT=8080 \
  -e SERVER_MODE=release \
  -e RUN_MODE=standard \
  -e DATABASE_HOST=postgres \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=sub2api \
  -e DATABASE_PASSWORD=7e8a5db0d468f01455947ad680c29fd0 \
  -e DATABASE_DBNAME=sub2api \
  -e DATABASE_SSLMODE=disable \
  -e DATABASE_MAX_OPEN_CONNS=50 \
  -e DATABASE_MAX_IDLE_CONNS=10 \
  -e DATABASE_CONN_MAX_LIFETIME_MINUTES=30 \
  -e DATABASE_CONN_MAX_IDLE_TIME_MINUTES=5 \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD= \
  -e REDIS_DB=0 \
  -e REDIS_POOL_SIZE=1024 \
  -e REDIS_MIN_IDLE_CONNS=10 \
  -e REDIS_ENABLE_TLS=false \
  -e ADMIN_EMAIL=admin@sub2api.local \
  -e ADMIN_PASSWORD= \
  -e JWT_SECRET=cd0f502c111b88d767e2bc50728946e3f877d5dd1f01d0060c309d0983a54c40 \
  -e JWT_EXPIRE_HOUR=24 \
  -e TOTP_ENCRYPTION_KEY=8e8c8e2a5bf3ab179b92bed229a3a10f4f918a5d09ae4f16716a8c60dd446060 \
  -e TZ=Asia/Shanghai \
  -e SECURITY_URL_ALLOWLIST_ENABLED=false \
  -e SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true \
  -e SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=true \
  -e UPDATE_PROXY_URL= \
  sub2api-bmai:<版本号>
```

检查：
```bash
sudo docker ps --filter name=sub2api
curl -sk -o /dev/null -w "%{http_code}" https://aiapi.yqmac.com/
```

---

## 7. 回滚流程

### 7.1 快速回滚应用镜像

前提：你保留了上一个可用镜像 tag，例如：
- `sub2api-bmai:0.1.113-bmai.1`

回滚命令：
```bash
sudo docker rm -f sub2api

sudo docker run -d \
  --name sub2api \
  --restart unless-stopped \
  --network deploy_sub2api-network \
  -p 8180:8080 \
  -v /data/cc/sub2api/deploy/data:/app/data \
  ...（环境变量与正式切换一致）... \
  sub2api-bmai:<上一个稳定版本>
```

### 7.2 为什么这次回滚风险低

因为本次类型是：
- 代码改动 + 前端改动
- **没有数据库 schema 变更**

所以只要容器镜像能回滚，服务就能快速恢复。

---

## 8. 数据库 / Redis 容器管理规范

### 8.1 当前推荐状态

生产当前建议保留：
- `sub2api-postgres`
- `sub2api-redis`

都连接到：
- `deploy_sub2api-network`

并且加网络别名：
- `postgres`
- `redis`

这样应用容器始终用固定 host：
- `DATABASE_HOST=postgres`
- `REDIS_HOST=redis`

### 8.2 当前不要对 DB/Redis 做的事

不要执行：
```bash
cd /data/cc/sub2api/deploy
sudo docker compose up -d --force-recreate sub2api
```

因为这会连带依赖重建，触发端口冲突。

---

## 9. 后续若要把 compose 修正为可用，需要改什么

要让 compose 恢复可安全使用，至少要做以下修正：

### 9.1 删除 postgres / redis 的宿主机端口映射

当前有：
```yaml
postgres:
  ports:
    - 5432:5432

redis:
  ports:
    - 6379:6379
```

应改为：
- **删除这两段 `ports`**
- 只保留容器内网络访问

因为应用和 DB/Redis 都在同一 docker network 内，不需要暴露到宿主机。

### 9.2 修正后才能考虑恢复 compose 管理

修正完成后，未来可考虑使用：
```bash
sudo docker compose up -d --no-deps sub2api
```

注意是 `--no-deps`，避免无意重建数据库和缓存。

在未修复前，不要用 compose 做生产切换。

---

## 10. 推荐的无感 / 低风险策略

### 10.1 当前可做到的最佳方式

当前最稳妥策略是：
1. 构建新镜像
2. 8181 启动预览容器
3. 用 `__preview_enable` 验证 UI/功能
4. 正式容器短暂重启切换
5. 如有问题立即用上一个镜像回滚

### 10.2 关于"无感"

严格意义上的"完全无感、零连接中断"，当前架构下**做不到**，因为正式流量最终都指向宿主机 `8180` 上的单个应用容器。

当前能做到的是：
- **对绝大多数用户低风险**
- **预览验证无影响**
- **正式切换时间极短**
- **回滚快**

### 10.3 真正零感知的未来方案

如果未来要做到更接近零中断，建议演进为：
- nginx 级蓝绿切换（8180 / 8182 双正式位）
- 或容器编排层滚动更新
- 或前置负载均衡 + 多副本应用

当前阶段先不必做，成本较高。

---

## 11. 这次二开的关键文件

重点关注：
- `backend/internal/service/ops_dashboard_models.go`
- `backend/internal/handler/admin/ops_dashboard_handler.go`
- `backend/internal/handler/admin/ops_snapshot_v2_handler.go`
- `backend/internal/repository/ops_repo_dashboard.go`
- `frontend/src/api/admin/ops.ts`
- `frontend/src/views/admin/ops/OpsDashboard.vue`
- `frontend/src/views/admin/ops/components/OpsDashboardHeader.vue`
- `UPSTREAM.md`

上游合并时，优先 grep：
```bash
grep -R "\[bmai-fork\]" -n backend frontend
```

---

## 12. 上游合并建议流程

```bash
# 1. 拉上游
git fetch upstream

# 2. 更新 main
git checkout main
git merge upstream/main
git push origin main

# 3. 回到自己的功能主线
git checkout feat/vendor-dashboard
git rebase main

# 4. 查找本地定制点
grep -R "\[bmai-fork\]" -n backend frontend

# 5. 重新构建镜像
sudo docker build -t sub2api-bmai:<新版本号> .

# 6. 走预览 -> 切正式 -> 回滚预案
```

---

## 13. 本次问题复盘（一定要记住）

### 13.1 问题一：前端筛选看起来无效

原因：
- `snapshot-v2` 聚合接口漏了 `account_id`
- 页面虽然有下拉框，但核心聚合接口没带上筛选

修复：
- `ops_snapshot_v2_handler.go` 改为复用 `parseOpsDashboardFilter()`
- `opsDashboardSnapshotV2CacheKey` 加 `AccountID`
- 前端 `getDashboardSnapshotV2` 加 `account_id`

### 13.2 问题二：compose 切正式失败

原因：
- `postgres` / `redis` 暴露宿主机端口
- 与现有服务端口冲突
- compose 连带依赖重建，导致正式应用没起来

修复：
- 临时改为手工 `docker run` 管理正式容器
- 后续再修 compose

---

## 14. 发版后检查命令

```bash
sudo docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | grep -E "sub2api|postgres|redis"

curl -sk -o /dev/null -w "%{http_code}" https://aiapi.yqmac.com/

curl -sk -o /dev/null -w "%{http_code}" -b "preview_aiapi=1" https://aiapi.yqmac.com/

sudo docker logs --tail=100 sub2api
sudo docker logs --tail=100 sub2api-test
```

---

## 15. 当前建议的后续动作

按优先级：
1. **保留当前预览 cookie 机制**，作为后续所有 UI 二开的标准验收入口
2. **下次发版继续走 `docker run` 方式**，不要直接 compose 切正式
3. **找一个窗口修 compose**：移除 postgres/redis 端口映射
4. **补一条 snapshot-v2 + account_id 的回归测试**，避免同类问题再出现
5. **长期**再考虑蓝绿/双正式位方案
