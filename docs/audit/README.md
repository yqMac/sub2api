# 审计 / 告警 / 拦截 功能设计文档

本目录包含 sub2api 二开（[bmai-fork]）的审计功能完整设计。

## 文档导航

| 文档 | 内容 | 受众 |
|------|------|------|
| [PLAN.md](./PLAN.md) | 总体方案、可行性结论、实施优先级 | 所有人，先看这个 |
| [DESIGN-BACKEND.md](./DESIGN-BACKEND.md) | 后端架构、数据模型、API 规范、Hook 集成 | 后端开发者 |
| [DESIGN-FRONTEND.md](./DESIGN-FRONTEND.md) | 前端组件、路由、状态管理、UI 规范 | 前端开发者 |
| [CONVENTIONS.md](./CONVENTIONS.md) | 编码规范、命名、Commit 规范、测试要求 | 所有开发者 |

## 阅读顺序

1. **第一次看**：先 `PLAN.md` 了解整体方案
2. **开始编码**：再看对应的 `DESIGN-*.md`
3. **提交代码**：先看 `CONVENTIONS.md`

## 实施进度

| 阶段 | 状态 | 版本 |
|------|------|------|
| P0：核心审计 + 组织架构 | 🔜 待开始 | 0.1.121-bmai.5 |
| P1：统计聚合 | ⏳ 计划中 | 0.1.121-bmai.6 |
| P2：规则引擎 | ⏳ 计划中 | 0.1.121-bmai.7 |
| P3：配额/导出 | ⏳ 计划中 | 0.1.121-bmai.8 |

## 关键决策

- 新建 `audit_logs` 独立表，不扩展 `usage_logs`
- 复用现有 `OpsAlertEvaluatorService` 告警引擎
- 飞书 OAuth 同步部门信息，建立组织架构层级
- 部门维度冗余写入 audit_logs，避免 JOIN
- 三个 Hook 点：Pre-Audit / 审计写入 / Post-Audit
- 所有写入异步，关闭时零开销

详见 PLAN.md。
