<!-- [bmai-fork] organizations management page with Feishu sync -->
<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.organizations.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.organizations.description') }}</p>
        </div>
        <button @click="showCreateOrg = true"
          class="rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-primary-600">
          {{ t('admin.organizations.create') }}
        </button>
      </div>

      <div v-if="loading" class="card flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="!orgs.length" class="card py-12 text-center text-gray-500 dark:text-gray-400">
        {{ t('admin.organizations.noOrganizations') }}
      </div>

      <div v-else class="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <!-- Org list (left) -->
        <div class="space-y-2 lg:col-span-1">
          <button
            v-for="org in orgs" :key="org.ID"
            @click="selectOrg(org.ID)"
            class="card block w-full cursor-pointer p-4 text-left transition hover:shadow-md"
            :class="selectedOrgId === org.ID ? 'ring-2 ring-primary-500' : ''">
            <div class="flex items-center justify-between">
              <div class="font-medium text-gray-900 dark:text-white">{{ org.Name }}</div>
              <span class="rounded-full px-2 py-0.5 text-xs font-medium"
                :class="org.Type === 'feishu' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'">
                {{ org.Type }}
              </span>
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ org.TenantKey }}</div>
          </button>
        </div>

        <!-- Right panel with tabs -->
        <div class="card lg:col-span-2" v-if="selectedOrgId">
          <!-- Tab bar -->
          <div class="border-b border-gray-100 dark:border-dark-700">
            <nav class="flex gap-0 px-4">
              <button v-for="tab in rightTabs" :key="tab.key"
                @click="rightTab = tab.key"
                class="border-b-2 px-4 py-3 text-sm font-medium transition-colors"
                :class="rightTab === tab.key
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'">
                {{ t(tab.labelKey) }}
              </button>
            </nav>
          </div>

          <!-- Tab: Departments -->
          <div v-show="rightTab === 'departments'">
            <div class="flex items-center justify-between px-6 py-4">
              <span class="text-sm text-gray-500 dark:text-gray-400">{{ departments.length }} {{ t('admin.organizations.department.tree') }}</span>
              <button @click="addDept(null)"
                class="rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-600">
                {{ t('admin.organizations.department.addRoot') }}
              </button>
            </div>
            <div v-if="!departments.length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              No departments yet
            </div>
            <div v-else class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
                <thead>
                  <tr class="text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                    <th class="px-6 py-3">{{ t('admin.organizations.department.name') }}</th>
                    <th class="px-6 py-3">{{ t('admin.organizations.department.path') }}</th>
                    <th class="px-6 py-3">{{ t('admin.organizations.department.externalId') }}</th>
                    <th class="px-6 py-3"></th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-50 dark:divide-dark-700">
                  <tr v-for="d in sortedDepts" :key="d.ID" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                    <td class="px-6 py-3" :style="{ paddingLeft: `${(d.Level - 1) * 1.5 + 1.5}rem` }">
                      <span class="font-medium text-gray-900 dark:text-white">{{ d.Name }}</span>
                    </td>
                    <td class="px-6 py-3 text-xs text-gray-500">{{ d.FullPath }}</td>
                    <td class="px-6 py-3 font-mono text-xs text-gray-500">{{ d.ExternalID }}</td>
                    <td class="px-6 py-3 text-right">
                      <button @click="addDept(d.ID)" class="mr-3 text-xs text-primary-600 hover:underline">+子部门</button>
                      <button @click="deleteDept(d)" class="text-xs text-red-500 hover:underline">{{ t('admin.organizations.department.delete') }}</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Tab: Feishu Config (only for feishu type) -->
          <div v-show="rightTab === 'feishu'" class="space-y-6 p-6">
            <!-- Credentials -->
            <div>
              <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">飞书应用凭证</h3>
              <div class="space-y-3">
                <div>
                  <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">App ID</label>
                  <input v-model="feishuForm.app_id" type="text" placeholder="cli_xxxxxxxxxx"
                    class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono dark:border-dark-600 dark:bg-dark-700" />
                </div>
                <div>
                  <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">App Secret</label>
                  <input v-model="feishuForm.app_secret" :type="showSecret ? 'text' : 'password'" placeholder="••••••••••••••••"
                    class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono dark:border-dark-600 dark:bg-dark-700" />
                  <button @click="showSecret = !showSecret" class="mt-1 text-xs text-gray-500 hover:text-gray-700">
                    {{ showSecret ? '隐藏' : '显示' }} Secret
                  </button>
                </div>
              </div>
              <div class="mt-4 flex gap-3">
                <button @click="saveFeishuConfig" :disabled="!feishuForm.app_id || !feishuForm.app_secret || savingConfig"
                  class="rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 hover:bg-primary-600">
                  {{ savingConfig ? '保存中...' : '保存凭证' }}
                </button>
                <button @click="testFeishu" :disabled="!feishuForm.app_id || !feishuForm.app_secret || testing"
                  class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 disabled:opacity-50 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700">
                  {{ testing ? '测试中...' : '测试连接' }}
                </button>
              </div>
              <!-- Test result -->
              <div v-if="testResult" class="mt-3 rounded-lg p-3 text-sm"
                :class="testResult.success ? 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-300' : 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'">
                <span v-if="testResult.success">✓ 连接成功，共 {{ testResult.department_count }} 个部门</span>
                <span v-else>✗ 连接失败：{{ testResult.error }}</span>
              </div>
            </div>

            <!-- Sync -->
            <div class="border-t border-gray-100 pt-6 dark:border-dark-700">
              <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">同步组织架构</h3>
              <p class="mb-4 text-xs text-gray-500 dark:text-gray-400">
                从飞书拉取所有部门和用户，按企业邮箱匹配系统用户，自动建立部门归属关系。
              </p>
              <button @click="syncFeishu" :disabled="syncing || !feishuConfigured"
                class="rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 hover:bg-primary-600">
                {{ syncing ? '同步中...' : '立即同步' }}
              </button>
              <p v-if="!feishuConfigured" class="mt-2 text-xs text-amber-600 dark:text-amber-400">请先保存飞书凭证</p>

              <!-- Sync result -->
              <div v-if="syncResult" class="mt-4 rounded-lg border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/50">
                <div class="mb-2 text-sm font-medium text-gray-900 dark:text-white">同步结果</div>
                <div class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
                  <div class="text-center">
                    <div class="text-2xl font-bold text-primary-600">{{ syncResult.departments_synced }}</div>
                    <div class="text-xs text-gray-500">部门同步</div>
                  </div>
                  <div class="text-center">
                    <div class="text-2xl font-bold text-primary-600">{{ syncResult.users_synced }}</div>
                    <div class="text-xs text-gray-500">飞书用户</div>
                  </div>
                  <div class="text-center">
                    <div class="text-2xl font-bold text-green-600">{{ syncResult.users_matched }}</div>
                    <div class="text-xs text-gray-500">已匹配</div>
                  </div>
                  <div class="text-center">
                    <div class="text-2xl font-bold text-gray-400">{{ syncResult.users_unmatched }}</div>
                    <div class="text-xs text-gray-500">未匹配</div>
                  </div>
                </div>
                <div v-if="syncResult.errors?.length" class="mt-3 rounded border border-red-200 bg-red-50 p-2 text-xs text-red-600 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
                  <div class="font-medium mb-1">{{ syncResult.errors.length }} 个错误：</div>
                  <div v-for="(e, i) in syncResult.errors.slice(0, 5)" :key="i">{{ e }}</div>
                  <div v-if="syncResult.errors.length > 5" class="text-gray-500">...还有 {{ syncResult.errors.length - 5 }} 个</div>
                </div>
              </div>

              <!-- Last sync status -->
              <div v-if="syncStatus?.last_sync_at" class="mt-3 text-xs text-gray-500 dark:text-gray-400">
                上次同步：{{ formatTime(syncStatus.last_sync_at) }}
                <span v-if="syncStatus.last_sync_error" class="ml-2 text-red-500">失败：{{ syncStatus.last_sync_error }}</span>
              </div>
            </div>
          </div>

          <!-- Tab: Users -->
          <div v-show="rightTab === 'users'">
            <div class="flex items-center justify-between px-6 py-4">
              <span class="text-sm text-gray-500 dark:text-gray-400">该组织下的用户</span>
            </div>
            <div v-if="!orgUsers.length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              暂无用户，请先同步飞书或手动分配
            </div>
            <div v-else class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
                <thead>
                  <tr class="text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                    <th class="px-6 py-3">用户 ID</th>
                    <th class="px-6 py-3">部门</th>
                    <th class="px-6 py-3">职位</th>
                    <th class="px-6 py-3">工号</th>
                    <th class="px-6 py-3">主部门</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-50 dark:divide-dark-700">
                  <tr v-for="u in orgUsers" :key="u.UserID" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                    <td class="px-6 py-3 font-mono text-xs">{{ u.UserID }}</td>
                    <td class="px-6 py-3 text-xs text-gray-500">{{ deptName(u.DepartmentID) }}</td>
                    <td class="px-6 py-3 text-xs text-gray-500">{{ u.Role || '-' }}</td>
                    <td class="px-6 py-3 text-xs text-gray-500">{{ u.EmployeeID || '-' }}</td>
                    <td class="px-6 py-3">
                      <span v-if="u.IsPrimary" class="rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-700 dark:bg-green-900/30 dark:text-green-300">主</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- No org selected -->
        <div v-else class="card flex items-center justify-center py-12 text-sm text-gray-500 dark:text-gray-400 lg:col-span-2">
          选择左侧组织查看详情
        </div>
      </div>
    </div>

    <!-- Create Org Dialog -->
    <Teleport to="body">
      <div v-if="showCreateOrg" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="showCreateOrg = false">
        <div class="card w-full max-w-md p-6">
          <h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.organizations.create') }}</h3>
          <div class="space-y-4">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.organizations.tenantKey') }}</label>
              <input v-model="newOrg.tenant_key" type="text" placeholder="例：tenant_xxx"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.organizations.name') }}</label>
              <input v-model="newOrg.name" type="text" placeholder="组织名称"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.organizations.type') }}</label>
              <select v-model="newOrg.type"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700">
                <option value="feishu">飞书 (feishu)</option>
                <option value="manual">手动 (manual)</option>
                <option value="oidc">OIDC</option>
              </select>
            </div>
          </div>
          <div class="mt-6 flex justify-end gap-3">
            <button @click="showCreateOrg = false"
              class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 dark:border-dark-600 dark:text-gray-300">
              取消
            </button>
            <button @click="createOrg" :disabled="!newOrg.tenant_key || !newOrg.name"
              class="rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white disabled:opacity-50">
              创建
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { organizationAPI, type Organization, type Department, type UserDepartment } from '@/api/admin/organizations'

const { t } = useI18n()

// State
const orgs = ref<Organization[]>([])
const selectedOrgId = ref<number | null>(null)
const selectedOrg = computed(() => orgs.value.find(o => o.ID === selectedOrgId.value) ?? null)
const departments = ref<Department[]>([])
const orgUsers = ref<UserDepartment[]>([])
const loading = ref(true)
const showCreateOrg = ref(false)
const newOrg = ref({ tenant_key: '', name: '', type: 'feishu' })

// Right panel tabs
const rightTab = ref<'departments' | 'feishu' | 'users'>('departments')
const rightTabs = computed(() => {
  const tabs: Array<{ key: 'departments' | 'feishu' | 'users'; labelKey: string }> = [
    { key: 'departments', labelKey: 'admin.organizations.department.tree' },
    { key: 'users', labelKey: 'admin.organizations.department.users' },
  ]
  if (selectedOrg.value?.Type === 'feishu') {
    tabs.splice(1, 0, { key: 'feishu', labelKey: 'admin.organizations.feishu.tab' })
  }
  return tabs
})

// Feishu config
const feishuForm = ref({ app_id: '', app_secret: '' })
const showSecret = ref(false)
const savingConfig = ref(false)
const testing = ref(false)
const syncing = ref(false)
const testResult = ref<{ success: boolean; department_count?: number; error?: string } | null>(null)
const syncResult = ref<any>(null)
const syncStatus = ref<any>(null)
const feishuConfigured = ref(false)

const sortedDepts = computed(() =>
  [...departments.value].sort((a, b) => a.FullPath.localeCompare(b.FullPath))
)

function deptName(deptId: number): string {
  return departments.value.find(d => d.ID === deptId)?.Name ?? String(deptId)
}

// Load
async function loadOrgs() {
  loading.value = true
  try {
    orgs.value = await organizationAPI.list()
    if (orgs.value.length > 0 && !selectedOrgId.value) await selectOrg(orgs.value[0].ID)
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

async function selectOrg(id: number) {
  selectedOrgId.value = id
  rightTab.value = 'departments'
  testResult.value = null
  syncResult.value = null
  try {
    departments.value = await organizationAPI.departmentTree(id)
    await loadFeishuStatus(id)
  } catch (e) { console.error(e) }
}

async function loadFeishuStatus(orgId: number) {
  try {
    const status = await organizationAPI.feishuStatus(orgId)
    syncStatus.value = status
    feishuConfigured.value = !!status?.feishu_configured
    if (status?.feishu_app_id) feishuForm.value.app_id = status.feishu_app_id
  } catch (e) { /* ignore */ }
}

async function loadOrgUsers() {
  if (!selectedOrgId.value) return
  try {
    // Load users from all departments
    const allUsers: UserDepartment[] = []
    for (const dept of departments.value) {
      const res = await organizationAPI.departmentUsers(dept.ID, 1, 200)
      allUsers.push(...(res.items || []))
    }
    // Deduplicate by UserID
    const seen = new Set<number>()
    orgUsers.value = allUsers.filter(u => {
      if (seen.has(u.UserID)) return false
      seen.add(u.UserID)
      return true
    })
  } catch (e) { console.error(e) }
}

// Watch tab change to load users lazily
watch(rightTab, async (tab) => {
  if (tab === 'users' && orgUsers.value.length === 0) await loadOrgUsers()
})

// Org CRUD
async function createOrg() {
  try {
    await organizationAPI.create(newOrg.value)
    showCreateOrg.value = false
    newOrg.value = { tenant_key: '', name: '', type: 'feishu' }
    await loadOrgs()
  } catch (e: any) { alert(e.message || 'Create failed') }
}

// Department CRUD
async function addDept(parentId: number | null) {
  if (!selectedOrgId.value) return
  const name = prompt('部门名称：')
  if (!name) return
  const externalId = prompt('外部 ID（飞书 department_id 或自定义）：', `dept-${Date.now()}`)
  if (!externalId) return
  try {
    const input: any = { name, external_id: externalId }
    if (parentId !== null) input.parent_id = parentId
    await organizationAPI.createDepartment(selectedOrgId.value, input)
    departments.value = await organizationAPI.departmentTree(selectedOrgId.value)
  } catch (e) { console.error(e) }
}

async function deleteDept(d: Department) {
  if (!confirm(`确认删除部门「${d.Name}」？`)) return
  try {
    await organizationAPI.deleteDepartment(d.ID)
    if (selectedOrgId.value) departments.value = await organizationAPI.departmentTree(selectedOrgId.value)
  } catch (e) { console.error(e) }
}

// Feishu
async function saveFeishuConfig() {
  if (!selectedOrgId.value) return
  savingConfig.value = true
  try {
    await organizationAPI.feishuConfig(selectedOrgId.value, feishuForm.value)
    feishuConfigured.value = true
    alert('凭证已保存')
  } catch (e: any) { alert(e.message || '保存失败') }
  finally { savingConfig.value = false }
}

async function testFeishu() {
  if (!selectedOrgId.value) return
  testing.value = true
  testResult.value = null
  try {
    // Save first if not yet saved
    await organizationAPI.feishuConfig(selectedOrgId.value, feishuForm.value)
    testResult.value = await organizationAPI.feishuTest(selectedOrgId.value)
    feishuConfigured.value = true
  } catch (e: any) {
    testResult.value = { success: false, error: e.message || '请求失败' }
  } finally { testing.value = false }
}

async function syncFeishu() {
  if (!selectedOrgId.value) return
  syncing.value = true
  syncResult.value = null
  try {
    syncResult.value = await organizationAPI.feishuSync(selectedOrgId.value)
    // Reload departments after sync
    departments.value = await organizationAPI.departmentTree(selectedOrgId.value)
    orgUsers.value = []  // Reset so next tab open reloads
    await loadFeishuStatus(selectedOrgId.value)
  } catch (e: any) { alert(e.message || '同步失败') }
  finally { syncing.value = false }
}

function formatTime(s: string): string {
  return new Date(s).toLocaleString()
}

onMounted(loadOrgs)
</script>
