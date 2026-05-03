<!-- [bmai-fork] audit log list with filters and detail dialog -->
<template>
  <div class="audit-log-list">
    <!-- Filters -->
    <div class="border-b border-gray-200 p-4 dark:border-gray-700">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-3 lg:grid-cols-6">
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.filters.timeRange') }}</label>
          <select v-model="filter.timeRange" class="w-full rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800">
            <option value="1h">1h</option>
            <option value="6h">6h</option>
            <option value="24h">24h</option>
            <option value="7d">7d</option>
            <option value="30d">30d</option>
          </select>
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.filters.userId') }}</label>
          <input v-model="filter.user_id" type="text" placeholder="1,2,3" class="w-full rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.filters.contentType') }}</label>
          <select v-model="filter.content_type" class="w-full rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800">
            <option value="">All</option>
            <option v-for="ct in contentTypes" :key="ct" :value="ct">{{ t(`admin.audit.contentType.${ct}`) }}</option>
          </select>
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.filters.model') }}</label>
          <input v-model="filter.model" type="text" placeholder="claude-3.5-sonnet" class="w-full rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.filters.platform') }}</label>
          <select v-model="filter.platform" class="w-full rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800">
            <option value="">All</option>
            <option value="anthropic">Anthropic</option>
            <option value="openai">OpenAI</option>
            <option value="gemini">Gemini</option>
            <option value="bedrock">Bedrock</option>
          </select>
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.filters.keyword') }}</label>
          <input v-model="filter.keyword" type="text" placeholder="search request/response" class="w-full rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" />
        </div>
      </div>
      <div class="mt-3 flex justify-end gap-2">
        <button @click="reset" class="rounded border border-gray-300 px-3 py-1 text-sm dark:border-gray-600">
          {{ t('admin.audit.filters.clear') }}
        </button>
        <button @click="load" class="rounded bg-primary-500 px-4 py-1 text-sm text-white hover:bg-primary-600">
          Search
        </button>
      </div>
    </div>

    <!-- Table -->
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead class="bg-gray-50 dark:bg-gray-900">
          <tr>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.time') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.user') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.department') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.model') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.contentType') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.request') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.response') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.tokens') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.duration') }}</th>
            <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.audit.columns.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-800">
          <tr v-if="loading">
            <td colspan="10" class="py-8 text-center text-gray-500">Loading...</td>
          </tr>
          <tr v-else-if="!items.length">
            <td colspan="10" class="py-8 text-center text-gray-500">No audit logs in this time range. Make sure audit is enabled in Settings tab.</td>
          </tr>
          <tr v-for="row in items" :key="row.id" class="hover:bg-gray-50 dark:hover:bg-gray-900">
            <td class="whitespace-nowrap px-3 py-2 text-xs text-gray-700 dark:text-gray-300">{{ formatTime(row.created_at) }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-sm text-gray-900 dark:text-gray-100">{{ row.user_id }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-sm text-gray-700 dark:text-gray-300">{{ row.department_path || '-' }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-sm text-gray-900 dark:text-gray-100">{{ row.model }}</td>
            <td class="whitespace-nowrap px-3 py-2">
              <span v-if="row.content_type" :class="badgeClass(row.content_type)" class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium">
                {{ t(`admin.audit.contentType.${row.content_type}`) }}
              </span>
            </td>
            <td class="max-w-xs truncate px-3 py-2 text-xs text-gray-600 dark:text-gray-400">{{ row.request_summary }}</td>
            <td class="max-w-xs truncate px-3 py-2 text-xs text-gray-600 dark:text-gray-400">{{ row.response_summary }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-xs text-gray-700 dark:text-gray-300">{{ row.input_tokens }}/{{ row.output_tokens }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-xs text-gray-700 dark:text-gray-300">{{ row.duration_ms }}ms</td>
            <td class="whitespace-nowrap px-3 py-2">
              <button @click="openDetail(row.id)" class="text-sm text-primary-600 hover:underline">View</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between border-t border-gray-200 px-4 py-3 dark:border-gray-700">
      <div class="text-sm text-gray-500">Total: {{ total }}</div>
      <div class="flex gap-2">
        <button @click="prevPage" :disabled="filter.page <= 1" class="rounded border border-gray-300 px-3 py-1 text-sm disabled:opacity-50 dark:border-gray-600">Prev</button>
        <span class="px-3 py-1 text-sm">{{ filter.page }} / {{ totalPages }}</span>
        <button @click="nextPage" :disabled="filter.page >= totalPages" class="rounded border border-gray-300 px-3 py-1 text-sm disabled:opacity-50 dark:border-gray-600">Next</button>
      </div>
    </div>

    <!-- Detail Dialog -->
    <AuditDetailDialog v-if="detailId" :id="detailId" @close="detailId = null" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { auditAPI, type AuditLogListItem, type AuditContentType } from '@/api/admin/audit'
import AuditDetailDialog from './components/AuditDetailDialog.vue'

const { t } = useI18n()

const filter = reactive({
  timeRange: '24h' as '1h' | '6h' | '24h' | '7d' | '30d',
  user_id: '',
  content_type: '',
  model: '',
  platform: '',
  keyword: '',
  page: 1,
  page_size: 50
})

const items = ref<AuditLogListItem[]>([])
const total = ref(0)
const loading = ref(false)
const detailId = ref<number | null>(null)

const contentTypes: AuditContentType[] = ['conversation', 'code', 'image', 'tool_use', 'reasoning', 'plan', 'script']

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filter.page_size)))

function rangeToTimes(): { start_time: string; end_time: string } {
  const end = new Date()
  const start = new Date()
  switch (filter.timeRange) {
    case '1h': start.setHours(end.getHours() - 1); break
    case '6h': start.setHours(end.getHours() - 6); break
    case '7d': start.setDate(end.getDate() - 7); break
    case '30d': start.setDate(end.getDate() - 30); break
    default: start.setHours(end.getHours() - 24)
  }
  return { start_time: start.toISOString(), end_time: end.toISOString() }
}

async function load() {
  loading.value = true
  try {
    const { start_time, end_time } = rangeToTimes()
    const res = await auditAPI.list({
      start_time,
      end_time,
      user_id: filter.user_id || undefined,
      content_type: filter.content_type || undefined,
      model: filter.model || undefined,
      platform: filter.platform || undefined,
      keyword: filter.keyword || undefined,
      page: filter.page,
      page_size: filter.page_size
    })
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e) {
    console.error('failed to load audit logs', e)
  } finally {
    loading.value = false
  }
}

function reset() {
  filter.user_id = ''
  filter.content_type = ''
  filter.model = ''
  filter.platform = ''
  filter.keyword = ''
  filter.page = 1
  load()
}

function prevPage() {
  if (filter.page > 1) {
    filter.page--
    load()
  }
}

function nextPage() {
  if (filter.page < totalPages.value) {
    filter.page++
    load()
  }
}

function openDetail(id: number) {
  detailId.value = id
}

function formatTime(s: string): string {
  return new Date(s).toLocaleString()
}

function badgeClass(ct: AuditContentType): string {
  const map: Record<AuditContentType, string> = {
    conversation: 'bg-blue-100 text-blue-800',
    code: 'bg-green-100 text-green-800',
    image: 'bg-purple-100 text-purple-800',
    tool_use: 'bg-orange-100 text-orange-800',
    reasoning: 'bg-yellow-100 text-yellow-800',
    plan: 'bg-indigo-100 text-indigo-800',
    script: 'bg-teal-100 text-teal-800'
  }
  return map[ct] || 'bg-gray-100 text-gray-800'
}

onMounted(load)
</script>
