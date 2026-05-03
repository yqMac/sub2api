# 审计功能 前端详细设计

> **范围**：sub2api 前端（Vue 3 + TypeScript + Tailwind），二开标记 `[bmai-fork]`
> **相关文档**：[PLAN.md](./PLAN.md) · [DESIGN-BACKEND.md](./DESIGN-BACKEND.md) · [CONVENTIONS.md](./CONVENTIONS.md)

## 目录

- [一、目录结构](#一目录结构)
- [二、路由与导航](#二路由与导航)
- [三、API 模块](#三api-模块)
- [四、AuditView 主页面](#四auditview-主页面)
- [五、Tab 1：日志列表](#五tab-1日志列表)
- [六、详情弹窗](#六详情弹窗)
- [七、Tab 2：统计概览](#七tab-2统计概览-p1)
- [八、Tab 3：规则管理](#八tab-3规则管理-p2)
- [九、Tab 4：审计设置](#九tab-4审计设置)
- [十、组织架构页](#十组织架构页)
- [十一、组件复用](#十一组件复用)
- [十二、i18n 规范](#十二i18n-规范)
- [十三、状态管理](#十三状态管理)

---

## 一、目录结构

```
frontend/src/
├── api/admin/
│   ├── audit.ts                              [bmai-fork]
│   └── organizations.ts                      [bmai-fork]
├── views/admin/
│   ├── AuditView.vue                         [bmai-fork] 主页 Tab 容器
│   ├── OrganizationsView.vue                 [bmai-fork] 组织架构页
│   ├── audit/
│   │   ├── AuditLogList.vue                  [bmai-fork] Tab 1
│   │   ├── AuditStats.vue                    [bmai-fork] Tab 2 (P1)
│   │   ├── AuditRules.vue                    [bmai-fork] Tab 3 (P2)
│   │   ├── AuditSettings.vue                 [bmai-fork] Tab 4
│   │   └── components/
│   │       ├── AuditDetailDialog.vue         [bmai-fork] 详情弹窗
│   │       ├── AuditFilters.vue              [bmai-fork] 筛选栏
│   │       ├── AuditContentTypeBadge.vue     [bmai-fork] 类型 badge
│   │       ├── AuditPreviewPanel.vue         [bmai-fork] 单栏 preview 显示
│   │       ├── AuditRuleEditor.vue           [bmai-fork] 规则编辑器（P2）
│   │       ├── DimensionPicker.vue           [bmai-fork] 维度选择器（P1）
│   │       ├── DepartmentDrillDown.vue       [bmai-fork] 部门钻取（P1）
│   │       └── AuditTrendChart.vue           [bmai-fork] 趋势图（P1）
│   └── organizations/
│       ├── DepartmentTree.vue                [bmai-fork]
│       ├── DepartmentDetail.vue              [bmai-fork]
│       └── UserDepartmentAssign.vue          [bmai-fork]
├── composables/
│   ├── useAuditFilters.ts                    [bmai-fork] 筛选状态 + URL 同步
│   └── useDepartmentTree.ts                  [bmai-fork] 部门树加载
└── stores/
    └── audit.ts                              [bmai-fork] Pinia store（设置缓存）
```

---

## 二、路由与导航

### 2.1 路由注册

`frontend/src/router/index.ts` 增加：

```typescript
// [bmai-fork] audit & organization routes
{
  path: '/admin/audit',
  name: 'AdminAudit',
  component: () => import('@/views/admin/AuditView.vue'),
  meta: {
    requiresAuth: true,
    requiresAdmin: true,
    titleKey: 'admin.audit.title'
  },
  children: [
    {
      path: '',
      redirect: { name: 'AdminAuditLogs' }
    },
    {
      path: 'logs',
      name: 'AdminAuditLogs',
      component: () => import('@/views/admin/audit/AuditLogList.vue'),
      meta: { titleKey: 'admin.audit.tabs.logs' }
    },
    {
      path: 'stats',
      name: 'AdminAuditStats',
      component: () => import('@/views/admin/audit/AuditStats.vue'),
      meta: { titleKey: 'admin.audit.tabs.stats' }
    },
    {
      path: 'rules',
      name: 'AdminAuditRules',
      component: () => import('@/views/admin/audit/AuditRules.vue'),
      meta: { titleKey: 'admin.audit.tabs.rules' }
    },
    {
      path: 'settings',
      name: 'AdminAuditSettings',
      component: () => import('@/views/admin/audit/AuditSettings.vue'),
      meta: { titleKey: 'admin.audit.tabs.settings' }
    }
  ]
},
{
  path: '/admin/organizations',
  name: 'AdminOrganizations',
  component: () => import('@/views/admin/OrganizationsView.vue'),
  meta: {
    requiresAuth: true,
    requiresAdmin: true,
    titleKey: 'admin.organizations.title'
  }
}
```

### 2.2 侧边栏菜单

在现有 admin 侧边栏（`AppLayout.vue` 或对应菜单配置）添加：

```
- Dashboard
- Accounts
- Users
- Groups
- ...
- 审计日志 (/admin/audit)        [bmai-fork]
- 组织架构 (/admin/organizations) [bmai-fork]
- ...
- Ops Dashboard
- Settings
```

图标参考：审计 → `📋`，组织 → `🏢`（实际用 lucide-vue-next 中的 `ClipboardList` 和 `Building2`）。

---

## 三、API 模块

### 3.1 `api/admin/audit.ts`

```typescript
// [bmai-fork] audit API module
import http from '@/api/http'

export interface AuditLogListItem {
  id: number
  request_id: string
  user_id: number
  user_email: string
  organization_id: number | null
  department_id: number | null
  department_path: string | null
  content_type: AuditContentType | null
  model: string
  platform: string
  endpoint: string
  request_summary: string
  response_summary: string
  request_bytes: number
  response_bytes: number
  input_tokens: number
  output_tokens: number
  duration_ms: number
  stream: boolean
  status_code: number
  risk_level: 'low' | 'medium' | 'high' | null
  intercepted: boolean
  created_at: string
}

export interface AuditLogDetail extends AuditLogListItem {
  request_preview: string
  request_truncated: boolean
  response_preview: string
  response_truncated: boolean
  upstream_request_preview: string | null
  upstream_request_bytes: number | null
  upstream_request_truncated: boolean
  upstream_response_preview: string | null
  upstream_response_bytes: number | null
  upstream_response_truncated: boolean
  intercept_reason: string | null
  risk_tags: string[] | null
}

export type AuditContentType =
  | 'conversation' | 'code' | 'image'
  | 'tool_use' | 'reasoning' | 'plan' | 'script'

export interface AuditLogFilter {
  start_time?: string
  end_time?: string
  user_id?: number[]
  organization_id?: number
  department_id?: number
  department_path?: string
  content_type?: AuditContentType[]
  model?: string[]
  platform?: string[]
  keyword?: string
  risk_level?: 'low' | 'medium' | 'high'
  intercepted?: boolean
  page?: number
  page_size?: number
  sort_by?: 'created_at' | 'duration_ms' | 'input_tokens' | 'output_tokens'
  sort_order?: 'asc' | 'desc'
}

export interface AuditSettings {
  enabled: boolean
  max_request_bytes: number
  max_response_bytes: number
  capture_upstream: boolean
  retention_days: number
  classify_response: boolean
}

export interface AuditStorageInfo {
  total_rows: number
  total_bytes: number
  earliest_record: string | null
  partitions: Array<{ name: string; rows: number; bytes: number }>
}

export const auditAPI = {
  list(filter: AuditLogFilter, signal?: AbortSignal) {
    return http.get<{ items: AuditLogListItem[]; total: number; page: number; page_size: number }>(
      '/api/admin/audit/logs',
      { params: filter, signal }
    )
  },
  get(id: number) {
    return http.get<AuditLogDetail>(`/api/admin/audit/logs/${id}`)
  },
  bulkDelete(filter: { start_time?: string; end_time?: string }) {
    return http.delete<{ deleted: number }>('/api/admin/audit/logs', { data: filter })
  },
  getSettings() {
    return http.get<AuditSettings>('/api/admin/audit/settings')
  },
  updateSettings(settings: Partial<AuditSettings>) {
    return http.put<AuditSettings>('/api/admin/audit/settings', settings)
  },
  getStorage() {
    return http.get<AuditStorageInfo>('/api/admin/audit/storage')
  },
  cleanup(beforeDate: string) {
    return http.post<{ deleted: number }>('/api/admin/audit/cleanup', { before_date: beforeDate })
  },
  // P1
  statsOverview(params: { start_time: string; end_time: string }) { /* ... */ },
  statsTrend(params: TrendParams) { /* ... */ },
  statsDistribution(params: DistributionParams) { /* ... */ },
  // P2
  rulesList() { /* ... */ },
  ruleCreate(rule: AuditRuleInput) { /* ... */ },
  ruleUpdate(id: number, rule: AuditRuleInput) { /* ... */ },
  ruleDelete(id: number) { /* ... */ },
}
```

### 3.2 `api/admin/organizations.ts`

```typescript
// [bmai-fork] organization API module
import http from '@/api/http'

export interface Organization {
  id: number
  tenant_key: string
  name: string
  type: 'feishu' | 'manual' | 'oidc'
  user_count: number
  created_at: string
}

export interface Department {
  id: number
  organization_id: number
  parent_id: number | null
  external_id: string
  name: string
  full_path: string
  level: number
  user_count: number
  children?: Department[]
}

export interface UserDepartment {
  user_id: number
  user_email: string
  department_id: number
  is_primary: boolean
  role: string | null
  employee_id: string | null
}

export const organizationAPI = {
  list() {
    return http.get<Organization[]>('/api/admin/organizations')
  },
  departmentTree(orgId: number) {
    return http.get<Department[]>(`/api/admin/organizations/${orgId}/departments`)
  },
  departmentUsers(deptId: number, page = 1, pageSize = 50) {
    return http.get<{ items: UserDepartment[]; total: number }>(
      `/api/admin/organizations/departments/${deptId}/users`,
      { params: { page, page_size: pageSize } }
    )
  },
  syncFromFeishu(orgId: number) {
    return http.post<{ synced_users: number; synced_departments: number }>(
      `/api/admin/organizations/${orgId}/sync-feishu`
    )
  }
}
```

---

## 四、AuditView 主页面

`views/admin/AuditView.vue`：

```vue
<!-- [bmai-fork] audit main view with tab navigation -->
<template>
  <AppLayout>
    <div class="audit-view">
      <!-- Tab Header -->
      <div class="border-b border-gray-200 dark:border-gray-700">
        <nav class="-mb-px flex space-x-8 px-6" aria-label="Tabs">
          <RouterLink
            v-for="tab in tabs"
            :key="tab.name"
            :to="{ name: tab.name }"
            class="whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm"
            :class="[
              isActive(tab.name)
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            ]"
          >
            {{ t(tab.titleKey) }}
            <span v-if="tab.badge" class="ml-2 rounded-full bg-red-100 px-2 py-0.5 text-xs text-red-800">
              {{ tab.badge }}
            </span>
          </RouterLink>
        </nav>
      </div>

      <!-- Tab Content -->
      <RouterView />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, RouterLink, RouterView } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'

const { t } = useI18n()
const route = useRoute()

const tabs = [
  { name: 'AdminAuditLogs', titleKey: 'admin.audit.tabs.logs' },
  { name: 'AdminAuditStats', titleKey: 'admin.audit.tabs.stats' },
  { name: 'AdminAuditRules', titleKey: 'admin.audit.tabs.rules' },
  { name: 'AdminAuditSettings', titleKey: 'admin.audit.tabs.settings' },
]

const isActive = (name: string) => route.name === name
</script>
```

---

## 五、Tab 1：日志列表

### 5.1 布局结构

```vue
<!-- [bmai-fork] audit log list -->
<template>
  <TablePageLayout>
    <template #filters>
      <AuditFilters v-model:filter="filter" :loading="loading" />
    </template>

    <template #table>
      <DataTable
        :columns="columns"
        :data="items"
        :loading="loading"
        :sort-by="filter.sort_by"
        :sort-order="filter.sort_order"
        @sort-change="handleSortChange"
        @row-click="openDetail"
      />
    </template>

    <template #pagination>
      <Pagination
        v-model:page="filter.page"
        v-model:page-size="filter.page_size"
        :total="total"
        :persisted-key="'audit-page-size'"
        @change="reload"
      />
    </template>
  </TablePageLayout>

  <AuditDetailDialog
    v-model:show="detailVisible"
    :log-id="selectedId"
    @close="detailVisible = false"
  />
</template>
```

### 5.2 筛选栏 `AuditFilters.vue`

| 控件 | 类型 | 状态绑定 |
|------|------|----------|
| 时间范围 | 复用 OpsDashboardHeader 时间选择器 | `start_time`, `end_time` |
| 用户 | 搜索下拉（按 email 搜索 users API） | `user_id[]` |
| 组织 | 下拉 | `organization_id` |
| 部门 | 树形选择器（点开展开） | `department_id` |
| 内容类型 | 多选 chip | `content_type[]` |
| 模型 | 下拉（来自 settings API 的可用模型列表） | `model[]` |
| 平台 | 下拉 | `platform[]` |
| 风险等级 | 下拉（low/medium/high） | `risk_level` |
| 已拦截 | toggle | `intercepted` |
| 关键词 | 搜索框 + 防抖 300ms | `keyword` |

URL 同步：通过 `useAuditFilters` composable 把 filter 状态同步到 query string。

### 5.3 表格列定义

```typescript
const columns: TableColumn[] = [
  { key: 'created_at', titleKey: 'admin.audit.columns.time', width: 160, sortable: true,
    render: row => formatDateTime(row.created_at) },
  { key: 'user_email', titleKey: 'admin.audit.columns.user', width: 180,
    render: row => row.user_email },
  { key: 'department_path', titleKey: 'admin.audit.columns.department', width: 180,
    render: row => row.department_path || '-' },
  { key: 'model', titleKey: 'admin.audit.columns.model', width: 160,
    render: row => `<PlatformTypeBadge platform="${row.platform}" /> ${row.model}` },
  { key: 'content_type', titleKey: 'admin.audit.columns.contentType', width: 100,
    render: row => `<AuditContentTypeBadge :type="${row.content_type}" />` },
  { key: 'request_summary', titleKey: 'admin.audit.columns.request', flex: 1,
    render: row => truncate(row.request_summary, 100) },
  { key: 'response_summary', titleKey: 'admin.audit.columns.response', flex: 1,
    render: row => truncate(row.response_summary, 100) },
  { key: 'tokens', titleKey: 'admin.audit.columns.tokens', width: 100,
    render: row => `${row.input_tokens} / ${row.output_tokens}` },
  { key: 'duration_ms', titleKey: 'admin.audit.columns.duration', width: 80, sortable: true,
    render: row => `${row.duration_ms}ms` },
  { key: 'risk', titleKey: 'admin.audit.columns.risk', width: 80,
    render: row => row.intercepted
      ? '<Badge color="red">已拦截</Badge>'
      : row.risk_level
        ? `<Badge color="${row.risk_level === 'high' ? 'red' : 'yellow'}">${row.risk_level}</Badge>`
        : '-' },
  { key: 'actions', titleKey: 'admin.audit.columns.actions', width: 80,
    render: row => `<button @click="openDetail(${row.id})">详情</button>` },
]
```

### 5.4 内容类型 Badge `AuditContentTypeBadge.vue`

颜色规范：

| Type | 颜色 | 图标 |
|------|------|------|
| conversation | blue | 💬 MessageCircle |
| code | green | 💻 Code |
| image | purple | 🖼️ Image |
| tool_use | orange | 🔧 Wrench |
| reasoning | yellow | 🧠 Brain |
| plan | indigo | 📋 ClipboardList |
| script | teal | 📜 ScrollText |

```vue
<template>
  <span
    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
    :class="badgeClass"
  >
    <component :is="icon" class="h-3 w-3" />
    {{ t(`admin.audit.contentType.${type}`) }}
  </span>
</template>
```

### 5.5 Composable：`useAuditFilters.ts`

```typescript
// [bmai-fork] sync filter state with URL query
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { AuditLogFilter } from '@/api/admin/audit'

export function useAuditFilters() {
  const route = useRoute()
  const router = useRouter()

  const filter = ref<AuditLogFilter>(parseFromQuery(route.query))

  watch(filter, (val) => {
    debouncedUpdateQuery(val)
  }, { deep: true })

  function parseFromQuery(q: any): AuditLogFilter { /* ... */ }
  function debouncedUpdateQuery(f: AuditLogFilter) { /* router.replace with new query */ }

  return { filter, reset: () => { filter.value = defaultFilter() } }
}
```

---

## 六、详情弹窗

`AuditDetailDialog.vue`：

```vue
<!-- [bmai-fork] audit detail dialog with 2x2 comparison view -->
<template>
  <BaseDialog
    :show="show"
    :title="t('admin.audit.detail.title', { id: log?.request_id })"
    width="extra-wide"
    @close="$emit('update:show', false)"
  >
    <div v-if="loading" class="py-12 text-center">
      <Spinner />
    </div>

    <div v-else-if="log" class="space-y-4">
      <!-- 元信息栏 -->
      <div class="grid grid-cols-2 gap-4 rounded-lg bg-gray-50 p-4 text-sm dark:bg-gray-800 md:grid-cols-4">
        <DetailField :label="t('admin.audit.detail.user')" :value="log.user_email" />
        <DetailField :label="t('admin.audit.detail.department')" :value="log.department_path" />
        <DetailField :label="t('admin.audit.detail.model')" :value="`${log.platform} / ${log.model}`" />
        <DetailField :label="t('admin.audit.detail.contentType')">
          <AuditContentTypeBadge :type="log.content_type" />
        </DetailField>
        <DetailField :label="t('admin.audit.detail.duration')" :value="`${log.duration_ms}ms`" />
        <DetailField :label="t('admin.audit.detail.tokens')" :value="`${log.input_tokens} in / ${log.output_tokens} out`" />
        <DetailField :label="t('admin.audit.detail.endpoint')" :value="log.endpoint" />
        <DetailField :label="t('admin.audit.detail.time')" :value="formatDateTime(log.created_at)" />
      </div>

      <!-- 风险信息（如果有） -->
      <div v-if="log.intercepted || log.risk_level" class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-700 dark:bg-red-900/20">
        <h4 class="mb-2 font-medium text-red-800 dark:text-red-200">
          {{ t('admin.audit.detail.riskInfo') }}
        </h4>
        <DetailField v-if="log.intercepted" :label="t('admin.audit.detail.interceptReason')" :value="log.intercept_reason" />
        <DetailField v-if="log.risk_level" :label="t('admin.audit.detail.riskLevel')" :value="log.risk_level" />
        <DetailField v-if="log.risk_tags" :label="t('admin.audit.detail.riskTags')" :value="log.risk_tags.join(', ')" />
      </div>

      <!-- 2x2 对比视图 -->
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <AuditPreviewPanel
          :title="t('admin.audit.detail.userRequest')"
          :preview="log.request_preview"
          :bytes="log.request_bytes"
          :truncated="log.request_truncated"
          format="json"
          default-expanded
        />
        <AuditPreviewPanel
          :title="t('admin.audit.detail.userResponse')"
          :preview="log.response_preview"
          :bytes="log.response_bytes"
          :truncated="log.response_truncated"
          :format="log.stream ? 'sse' : 'json'"
          default-expanded
        />
        <AuditPreviewPanel
          :title="t('admin.audit.detail.upstreamRequest')"
          :preview="log.upstream_request_preview"
          :bytes="log.upstream_request_bytes"
          :truncated="log.upstream_request_truncated"
          format="json"
          :empty-text="t('admin.audit.detail.upstreamCaptureDisabled')"
        />
        <AuditPreviewPanel
          :title="t('admin.audit.detail.upstreamResponse')"
          :preview="log.upstream_response_preview"
          :bytes="log.upstream_response_bytes"
          :truncated="log.upstream_response_truncated"
          :format="log.stream ? 'sse' : 'json'"
          :empty-text="t('admin.audit.detail.upstreamCaptureDisabled')"
        />
      </div>
    </div>

    <template #footer>
      <button @click="copyAll" class="btn-secondary">{{ t('common.copy') }}</button>
      <button @click="$emit('update:show', false)" class="btn-primary">{{ t('common.close') }}</button>
    </template>
  </BaseDialog>
</template>
```

`AuditPreviewPanel.vue` 设计：
- 标题栏 + 字节数 + 截断提示 + 复制按钮 + 折叠按钮
- 内容区根据 `format` 选择渲染：
  - `json`：缩进 + 语法高亮（轻量实现，不引入新库）
  - `sse`：按 `data:` 分块显示
  - `text`：纯文本
- 截断时显示 `"显示前 32KB / 共 120KB"`

---

## 七、Tab 2：统计概览 (P1)

### 7.1 顶部控制栏

```
┌────────────────────────────────────────────────────────────┐
│  时间范围 [24h ▾]  粒度 [小时 ▾]  维度 [部门 ▾]  组织 [全部 ▾]│
└────────────────────────────────────────────────────────────┘
```

`DimensionPicker.vue` 提供：用户/部门/组织/内容类型/模型/平台/无（仅时间）

### 7.2 KPI 卡片行（4 个）

```
┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
│ 总请求数   │ │ 总 Tokens │ │ 平均耗时   │ │ 风险事件   │
│ 12,345    │ │ 2.5M in   │ │ 1,234ms   │ │ 25        │
│ +15%      │ │ 8.1M out  │ │ p95: 3200 │ │ 已拦截 5  │
└───────────┘ └───────────┘ └───────────┘ └───────────┘
```

### 7.3 图表区（2x2 网格）

| 位置 | 图表 | 组件 |
|------|------|------|
| 左上 | 请求量趋势（堆叠面积，按内容类型/部门分组） | AuditTrendChart.vue |
| 右上 | 内容类型分布（环形图） | DonutChart.vue（复用） |
| 左下 | 模型/部门 Top 10（横向柱状） | BarChart.vue（复用） |
| 右下 | Token 消耗趋势（双轴折线） | DualLineChart.vue |

### 7.4 部门钻取卡片 `DepartmentDrillDown.vue`

```
全公司 (12,345)
├─ 技术部 (8,000)              ▶ 钻取
│  ├─ 平台组 (3,500)
│  ├─ 算法组 (2,500)
│  └─ 基础架构组 (2,000)
└─ 产品部 (3,000)
```

每行显示：请求数 + 内容类型迷你饼图 + Token 总量 + 风险事件数。

---

## 八、Tab 3：规则管理 (P2)

### 8.1 列表页

复用 `TablePageLayout` + `DataTable`，列：

| 列 | 内容 |
|----|------|
| 名称 | rule.name |
| 类型 | Badge: 拦截 / 告警 / 标记 |
| 状态 | toggle 开关 |
| 优先级 | 数字 |
| 作用范围 | 摘要文本（"3 个部门, 全用户"） |
| 命中次数(7d) | rule.hit_count_7d |
| 操作 | 编辑 / 删除 |

### 8.2 编辑器 `AuditRuleEditor.vue`

复用 BaseDialog（width=wide），分四个区块：

1. **基本信息**：名称、描述、类型、启用、优先级
2. **作用范围**：组织、部门（树形多选）、用户（搜索多选）、模型、内容类型
3. **匹配条件**（动态可视化构建器）：
   - 关键词：字段（请求/响应/上游请求/上游响应）+ 模式（多个，正则可选）
   - 频次：窗口（秒）+ 最大次数
   - 大小：字段 + 最大字节
4. **动作**：根据规则类型显示不同选项

---

## 九、Tab 4：审计设置

`AuditSettings.vue`：

```vue
<template>
  <div class="max-w-3xl space-y-6 p-6">
    <!-- 总开关 -->
    <SettingSection :title="t('admin.audit.settings.master.title')">
      <SettingRow
        :label="t('admin.audit.settings.master.enabled')"
        :description="t('admin.audit.settings.master.enabledDesc')"
      >
        <ToggleSwitch v-model="settings.enabled" />
      </SettingRow>
    </SettingSection>

    <!-- 捕获设置 -->
    <SettingSection :title="t('admin.audit.settings.capture.title')" :disabled="!settings.enabled">
      <SettingRow :label="t('admin.audit.settings.capture.maxRequest')">
        <NumberInput v-model="settings.max_request_bytes" :min="1024" :max="1048576" suffix="bytes" />
      </SettingRow>
      <SettingRow :label="t('admin.audit.settings.capture.maxResponse')">
        <NumberInput v-model="settings.max_response_bytes" :min="1024" :max="1048576" suffix="bytes" />
      </SettingRow>
      <SettingRow
        :label="t('admin.audit.settings.capture.upstream')"
        :description="t('admin.audit.settings.capture.upstreamDesc')"
      >
        <ToggleSwitch v-model="settings.capture_upstream" />
      </SettingRow>
      <SettingRow
        :label="t('admin.audit.settings.capture.classifyResponse')"
        :description="t('admin.audit.settings.capture.classifyResponseDesc')"
      >
        <ToggleSwitch v-model="settings.classify_response" />
      </SettingRow>
    </SettingSection>

    <!-- 存储管理 -->
    <SettingSection :title="t('admin.audit.settings.storage.title')">
      <SettingRow :label="t('admin.audit.settings.storage.retention')">
        <NumberInput v-model="settings.retention_days" :min="1" :max="365" suffix="days" />
      </SettingRow>
      <div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-800">
        <div class="grid grid-cols-3 gap-4 text-sm">
          <div>
            <div class="text-gray-500">{{ t('admin.audit.settings.storage.totalRows') }}</div>
            <div class="text-lg font-semibold">{{ formatNumber(storage.total_rows) }}</div>
          </div>
          <div>
            <div class="text-gray-500">{{ t('admin.audit.settings.storage.totalSize') }}</div>
            <div class="text-lg font-semibold">{{ formatBytes(storage.total_bytes) }}</div>
          </div>
          <div>
            <div class="text-gray-500">{{ t('admin.audit.settings.storage.earliest') }}</div>
            <div class="text-lg font-semibold">{{ formatDate(storage.earliest_record) }}</div>
          </div>
        </div>
        <div class="mt-4 flex gap-2">
          <button @click="manualCleanup" class="btn-secondary">
            {{ t('admin.audit.settings.storage.manualCleanup') }}
          </button>
        </div>
      </div>
    </SettingSection>

    <!-- 保存按钮 -->
    <div class="flex justify-end gap-3 border-t pt-4">
      <button @click="reset" class="btn-secondary">{{ t('common.reset') }}</button>
      <button @click="save" :disabled="!dirty || saving" class="btn-primary">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </div>
  </div>
</template>
```

---

## 十、组织架构页

`OrganizationsView.vue` 双栏布局：

```
┌──────────────────────────┐  ┌──────────────────────────────┐
│ 组织列表                  │  │ 部门树 / 详情                 │
│                          │  │                              │
│ ▸ YQ Mac (tenant_xxx)   │  │  YQ Mac                      │
│   25 用户  最后同步 2h 前 │  │  ├─ 技术部 (15)               │
│   [飞书同步] [详情]       │  │  │  ├─ 平台组 (8)              │
│                          │  │  │  ├─ 算法组 (5)              │
│ ▸ Other Co              │  │  │  └─ 基础架构组 (2)          │
│   3 用户                 │  │  ├─ 产品部 (5)                 │
│                          │  │  └─ 运营部 (5)                 │
│ [+ 手动添加]              │  │                              │
└──────────────────────────┘  └──────────────────────────────┘
```

部门点击 → 右侧切换到部门详情（用户列表 + 配额信息 + 关联规则）。

---

## 十一、组件复用

| 现有组件 | 用途 |
|---------|------|
| `AppLayout` | 主布局 |
| `TablePageLayout` | 表格页面布局 |
| `DataTable` | 表格 + 虚拟滚动 |
| `BaseDialog` | 弹窗 |
| `Pagination` | 分页 |
| `PlatformTypeBadge` | 平台标识 |
| `ToggleSwitch` | 开关（参考 SettingsView 中的实现） |
| `Spinner` / `LoadingState` | 加载状态 |
| `useTableLoader` | 表格加载逻辑 |
| `usePersistedPageSize` | 分页大小持久化 |

| 新增组件（基础） | 用途 |
|-----------------|------|
| `SettingSection` / `SettingRow` | 设置项布局（如果不存在） |
| `NumberInput` | 数字输入 |
| `DonutChart` / `DualLineChart` | 图表（基于 chart.js） |

---

## 十二、i18n 规范

所有文案必须走 i18n。新增 `admin.audit.*` 和 `admin.organizations.*` 命名空间。

`frontend/src/i18n/locales/zh.ts` 增加：

```typescript
admin: {
  // 现有...
  audit: {
    title: '审计日志',
    tabs: {
      logs: '日志列表',
      stats: '统计概览',
      rules: '规则管理',
      settings: '审计设置',
    },
    columns: {
      time: '时间',
      user: '用户',
      department: '部门',
      model: '模型',
      contentType: '类型',
      request: '请求摘要',
      response: '响应摘要',
      tokens: 'Tokens',
      duration: '耗时',
      risk: '风险',
      actions: '操作',
    },
    contentType: {
      conversation: '对话',
      code: '代码',
      image: '图片',
      tool_use: '工具调用',
      reasoning: '推理',
      plan: '方案',
      script: '脚本',
    },
    detail: {
      title: '审计详情 - {id}',
      user: '用户',
      department: '部门',
      model: '模型',
      contentType: '类型',
      duration: '耗时',
      tokens: 'Tokens',
      endpoint: '端点',
      time: '时间',
      riskInfo: '风险信息',
      interceptReason: '拦截原因',
      riskLevel: '风险等级',
      riskTags: '风险标签',
      userRequest: '用户请求',
      userResponse: '用户响应',
      upstreamRequest: '上游请求',
      upstreamResponse: '上游响应',
      upstreamCaptureDisabled: '上游捕获未启用',
    },
    settings: {
      master: { title: '总开关', enabled: '启用审计', enabledDesc: '...' },
      capture: { title: '捕获设置', maxRequest: '请求最大字节', /* ... */ },
      storage: { title: '存储管理', retention: '保留天数', /* ... */ },
    },
  },
  organizations: {
    title: '组织架构',
    syncFromFeishu: '从飞书同步',
    // ...
  },
},
```

`en.ts` 同步翻译。

---

## 十三、状态管理

### 13.1 Pinia store `stores/audit.ts`

```typescript
// [bmai-fork] audit settings cache
import { defineStore } from 'pinia'
import { auditAPI, type AuditSettings } from '@/api/admin/audit'

export const useAuditStore = defineStore('audit', () => {
  const settings = ref<AuditSettings | null>(null)
  const loaded = ref(false)

  async function load() {
    if (loaded.value) return
    settings.value = (await auditAPI.getSettings()).data
    loaded.value = true
  }

  async function update(patch: Partial<AuditSettings>) {
    settings.value = (await auditAPI.updateSettings(patch)).data
  }

  return { settings, loaded, load, update }
})
```

### 13.2 局部状态

筛选状态、表格状态等使用 `ref` + composable 管理，不放 store。

### 13.3 缓存策略

- 设置：Pinia 缓存，启用/禁用切换时刷新
- 部门树：组件级 ref + composable，进入页面时拉取
- 列表数据：不缓存，每次请求都拉新数据

---

## 附录：开发顺序建议

P0 实现顺序（前端）：
1. API 模块 (`api/admin/audit.ts`, `api/admin/organizations.ts`)
2. 路由注册 + 主页面 (`AuditView.vue`)
3. 设置页 (`AuditSettings.vue`) — 简单，可以先验证 API 联调
4. 列表页（不含详情弹窗） — 验证表格 + 筛选
5. 详情弹窗 — 验证 4 栏对比视图
6. 组织架构页 — 验证部门树
7. i18n 文案

P1 实现顺序：
1. 统计 API 联调
2. KPI 卡片
3. 图表（先用占位数据验证布局）
4. 部门钻取
5. 维度切换

P2 实现顺序：
1. 规则列表页
2. 规则编辑器（先做基础字段，再做匹配条件构建器）
3. 命中事件查看
