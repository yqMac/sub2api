<!-- [bmai-fork] organizations management page (P0: flat department list) -->
<template>
  <div class="organizations-view min-h-screen bg-gray-50 dark:bg-gray-900">
    <div class="container mx-auto px-4 py-6">
      <div class="mb-4 flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">{{ t('admin.organizations.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500">{{ t('admin.organizations.description') }}</p>
        </div>
        <button @click="showCreateOrg = true" class="rounded bg-primary-500 px-4 py-2 text-sm text-white hover:bg-primary-600">
          {{ t('admin.organizations.create') }}
        </button>
      </div>

      <div v-if="loading" class="rounded-lg bg-white p-12 text-center text-gray-500 shadow dark:bg-gray-800">Loading...</div>

      <div v-else-if="!orgs.length" class="rounded-lg bg-white p-12 text-center text-gray-500 shadow dark:bg-gray-800">
        {{ t('admin.organizations.noOrganizations') }}
      </div>

      <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <!-- Org list (left) -->
        <div class="space-y-2 md:col-span-1">
          <button
            v-for="org in orgs"
            :key="org.ID"
            @click="selectOrg(org.ID)"
            class="block w-full rounded-lg border bg-white p-4 text-left shadow-sm transition hover:shadow dark:bg-gray-800"
            :class="selectedOrgId === org.ID ? 'border-primary-500' : 'border-gray-200 dark:border-gray-700'"
          >
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ org.Name }}</div>
            <div class="text-xs text-gray-500">{{ org.TenantKey }} · {{ org.Type }}</div>
          </button>
        </div>

        <!-- Department list (right) -->
        <div class="rounded-lg bg-white p-4 shadow dark:bg-gray-800 md:col-span-2">
          <div v-if="!selectedOrgId" class="py-8 text-center text-sm text-gray-500">
            Select an organization to view departments
          </div>
          <div v-else>
            <div class="mb-3 flex items-center justify-between">
              <h3 class="font-medium">{{ t('admin.organizations.department.tree') }}</h3>
              <button @click="addDept(null)" class="rounded bg-primary-500 px-3 py-1 text-sm text-white hover:bg-primary-600">
                {{ t('admin.organizations.department.addRoot') }}
              </button>
            </div>
            <div v-if="!departments.length" class="py-6 text-center text-sm text-gray-500">No departments</div>
            <table v-else class="min-w-full divide-y divide-gray-200 text-sm dark:divide-gray-700">
              <thead class="bg-gray-50 dark:bg-gray-900">
                <tr>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.organizations.department.name') }}</th>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.organizations.department.path') }}</th>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.organizations.department.externalId') }}</th>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.organizations.department.users') }}</th>
                  <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.organizations.department.edit') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                <tr v-for="d in sortedDepts" :key="d.ID">
                  <td class="px-3 py-2" :style="{ paddingLeft: `${(d.Level - 1) * 1.5 + 0.75}rem` }">
                    <span class="font-medium">{{ d.Name }}</span>
                  </td>
                  <td class="px-3 py-2 text-xs text-gray-500">{{ d.FullPath }}</td>
                  <td class="px-3 py-2 font-mono text-xs text-gray-500">{{ d.ExternalID }}</td>
                  <td class="px-3 py-2">
                    <button @click="manageUsers(d)" class="text-xs text-primary-600 hover:underline">Manage</button>
                  </td>
                  <td class="px-3 py-2">
                    <button @click="addDept(d.ID)" class="mr-2 text-xs text-primary-600 hover:underline">+</button>
                    <button @click="deleteDept(d)" class="text-xs text-red-600 hover:underline">{{ t('admin.organizations.department.delete') }}</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Org Dialog -->
    <Teleport to="body">
      <div v-if="showCreateOrg" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="showCreateOrg = false">
        <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-gray-800">
          <h3 class="mb-4 text-lg font-medium">{{ t('admin.organizations.create') }}</h3>
          <div class="space-y-3">
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('admin.organizations.tenantKey') }}</label>
              <input v-model="newOrg.tenant_key" type="text" class="w-full rounded border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-900" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('admin.organizations.name') }}</label>
              <input v-model="newOrg.name" type="text" class="w-full rounded border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-900" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('admin.organizations.type') }}</label>
              <select v-model="newOrg.type" class="w-full rounded border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-900">
                <option value="manual">manual</option>
                <option value="feishu">feishu</option>
                <option value="oidc">oidc</option>
              </select>
            </div>
          </div>
          <div class="mt-4 flex justify-end gap-2">
            <button @click="showCreateOrg = false" class="rounded border border-gray-300 px-4 py-2 text-sm dark:border-gray-600">Cancel</button>
            <button @click="createOrg" :disabled="!newOrg.tenant_key || !newOrg.name" class="rounded bg-primary-500 px-4 py-2 text-sm text-white disabled:opacity-50">Create</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { organizationAPI, type Organization, type Department } from '@/api/admin/organizations'

const { t } = useI18n()

const orgs = ref<Organization[]>([])
const selectedOrgId = ref<number | null>(null)
const departments = ref<Department[]>([])
const loading = ref(true)

const showCreateOrg = ref(false)
const newOrg = ref({ tenant_key: '', name: '', type: 'manual' })

// Sort by full_path so children appear under parents
const sortedDepts = computed(() =>
  [...departments.value].sort((a, b) => a.FullPath.localeCompare(b.FullPath))
)

async function loadOrgs() {
  loading.value = true
  try {
    orgs.value = await organizationAPI.list()
    if (orgs.value.length > 0 && !selectedOrgId.value) {
      await selectOrg(orgs.value[0].ID)
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function selectOrg(id: number) {
  selectedOrgId.value = id
  try {
    departments.value = await organizationAPI.departmentTree(id)
  } catch (e) {
    console.error(e)
  }
}

async function createOrg() {
  try {
    await organizationAPI.create(newOrg.value)
    showCreateOrg.value = false
    newOrg.value = { tenant_key: '', name: '', type: 'manual' }
    await loadOrgs()
  } catch (e: any) {
    alert(e.message || 'Create failed')
  }
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
  } catch (e) {
    console.error(e)
  }
}

async function deleteDept(d: Department) {
  if (!confirm(`Delete department "${d.Name}"?`)) return
  try {
    await organizationAPI.deleteDepartment(d.ID)
    if (selectedOrgId.value) await selectOrg(selectedOrgId.value)
  } catch (e) {
    console.error(e)
  }
}

async function manageUsers(d: Department) {
  const userIdStr = prompt(`Assign user to "${d.Name}". Enter user ID:`)
  if (!userIdStr) return
  const userId = parseInt(userIdStr, 10)
  if (!userId) return
  const isPrimary = confirm('Set as primary department for this user?')
  try {
    await organizationAPI.assignUser(d.ID, { user_id: userId, is_primary: isPrimary })
    alert('User assigned')
  } catch (e: any) {
    alert(e.message || 'Failed')
  }
}

onMounted(loadOrgs)
</script>
