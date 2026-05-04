<!-- [bmai-fork] organizations management page — AppLayout wrapped -->
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
            <div class="font-medium text-gray-900 dark:text-white">{{ org.Name }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ org.TenantKey }} · {{ org.Type }}</div>
          </button>
        </div>

        <!-- Department panel (right) -->
        <div class="card lg:col-span-2">
          <div v-if="!selectedOrgId" class="py-12 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.organizations.department.tree') }}
          </div>
          <template v-else>
            <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.organizations.department.tree') }}</h3>
              <button @click="addDept(null)"
                class="rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-600">
                {{ t('admin.organizations.department.addRoot') }}
              </button>
            </div>
            <div v-if="!departments.length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              No departments
            </div>
            <div v-else class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
                <thead>
                  <tr class="text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                    <th class="px-6 py-3">{{ t('admin.organizations.department.name') }}</th>
                    <th class="px-6 py-3">{{ t('admin.organizations.department.path') }}</th>
                    <th class="px-6 py-3">{{ t('admin.organizations.department.externalId') }}</th>
                    <th class="px-6 py-3">{{ t('admin.organizations.department.users') }}</th>
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
                    <td class="px-6 py-3">
                      <button @click="manageUsers(d)" class="text-xs text-primary-600 hover:underline">Manage</button>
                    </td>
                    <td class="px-6 py-3 text-right">
                      <button @click="addDept(d.ID)" class="mr-2 text-xs text-primary-600 hover:underline">+</button>
                      <button @click="deleteDept(d)" class="text-xs text-red-500 hover:underline">{{ t('admin.organizations.department.delete') }}</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </template>
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
              <input v-model="newOrg.tenant_key" type="text"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.organizations.name') }}</label>
              <input v-model="newOrg.name" type="text"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.organizations.type') }}</label>
              <select v-model="newOrg.type"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700">
                <option value="manual">manual</option>
                <option value="feishu">feishu</option>
                <option value="oidc">oidc</option>
              </select>
            </div>
          </div>
          <div class="mt-6 flex justify-end gap-3">
            <button @click="showCreateOrg = false"
              class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 dark:border-dark-600 dark:text-gray-300">
              Cancel
            </button>
            <button @click="createOrg" :disabled="!newOrg.tenant_key || !newOrg.name"
              class="rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white disabled:opacity-50">
              Create
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { organizationAPI, type Organization, type Department } from '@/api/admin/organizations'

const { t } = useI18n()
const orgs = ref<Organization[]>([])
const selectedOrgId = ref<number | null>(null)
const departments = ref<Department[]>([])
const loading = ref(true)
const showCreateOrg = ref(false)
const newOrg = ref({ tenant_key: '', name: '', type: 'manual' })

const sortedDepts = computed(() =>
  [...departments.value].sort((a, b) => a.FullPath.localeCompare(b.FullPath))
)

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
  try { departments.value = await organizationAPI.departmentTree(id) }
  catch (e) { console.error(e) }
}

async function createOrg() {
  try {
    await organizationAPI.create(newOrg.value)
    showCreateOrg.value = false
    newOrg.value = { tenant_key: '', name: '', type: 'manual' }
    await loadOrgs()
  } catch (e: any) { alert(e.message || 'Create failed') }
}

async function addDept(parentId: number | null) {
  if (!selectedOrgId.value) return
  const name = prompt('Department name:')
  if (!name) return
  const externalId = prompt('External ID:', `dept-${Date.now()}`)
  if (!externalId) return
  try {
    const input: any = { name, external_id: externalId }
    if (parentId !== null) input.parent_id = parentId
    await organizationAPI.createDepartment(selectedOrgId.value, input)
    await selectOrg(selectedOrgId.value)
  } catch (e) { console.error(e) }
}

async function deleteDept(d: Department) {
  if (!confirm(`Delete department "${d.Name}"?`)) return
  try {
    await organizationAPI.deleteDepartment(d.ID)
    if (selectedOrgId.value) await selectOrg(selectedOrgId.value)
  } catch (e) { console.error(e) }
}

async function manageUsers(d: Department) {
  const userIdStr = prompt(`Assign user to "${d.Name}". Enter user ID:`)
  if (!userIdStr) return
  const userId = parseInt(userIdStr, 10)
  if (!userId) return
  const isPrimary = confirm('Set as primary department?')
  try {
    await organizationAPI.assignUser(d.ID, { user_id: userId, is_primary: isPrimary })
    alert('User assigned')
  } catch (e: any) { alert(e.message || 'Failed') }
}

onMounted(loadOrgs)
</script>
