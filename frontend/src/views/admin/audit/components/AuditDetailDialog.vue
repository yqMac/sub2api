<!-- [bmai-fork] audit detail dialog with 2x2 comparison view -->
<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="$emit('close')">
      <div class="max-h-[90vh] w-full max-w-6xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-gray-800">
        <!-- Header -->
        <div class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {{ t('admin.audit.detail.title') }}
            <span v-if="log" class="ml-2 font-mono text-sm text-gray-500">{{ log.request_id }}</span>
          </h2>
          <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Body -->
        <div class="max-h-[calc(90vh-7rem)] overflow-y-auto p-6">
          <div v-if="loading" class="py-12 text-center text-gray-500">Loading...</div>
          <div v-else-if="log" class="space-y-4">
            <!-- Meta info -->
            <div class="grid grid-cols-2 gap-4 rounded-lg bg-gray-50 p-4 text-sm dark:bg-gray-900 md:grid-cols-4">
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.userId') || '用户 ID' }}</div>
                <div class="font-medium">{{ log.user_id }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.department') || '部门' }}</div>
                <div class="font-medium">{{ log.department_path || '-' }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.model') || '模型' }}</div>
                <div class="font-medium">
                  <span class="rounded bg-gray-200 px-1.5 py-0.5 text-xs dark:bg-gray-700">{{ platformLabel(log.platform) }}</span>
                  <span class="ml-1">{{ log.model }}</span>
                </div>
              </div>
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.contentType') || '类型' }}</div>
                <div class="font-medium">{{ log.content_type ? t(`admin.audit.contentType.${log.content_type}`) : '-' }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.duration') || '耗时' }}</div>
                <div class="font-medium">{{ log.duration_ms }}ms</div>
              </div>
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.tokens') || 'Tokens' }}</div>
                <div class="font-medium">{{ log.input_tokens }} {{ t('admin.audit.detail.tokensIn') || '输入' }} / {{ log.output_tokens }} {{ t('admin.audit.detail.tokensOut') || '输出' }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.streamMode') || '请求模式' }}</div>
                <div class="font-medium">{{ log.stream ? t('admin.audit.stream.yes') : t('admin.audit.stream.no') }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.statusCode') || '状态码' }}</div>
                <div class="font-medium">
                  <span :class="statusClass(log.status_code)" class="rounded px-1.5 py-0.5 text-xs">{{ log.status_code }} · {{ statusLabel(log.status_code) }}</span>
                </div>
              </div>
              <div class="col-span-2">
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.endpoint') || '端点' }}</div>
                <div class="font-mono text-xs">{{ log.endpoint }}</div>
              </div>
              <div class="col-span-2">
                <div class="text-xs text-gray-500">{{ t('admin.audit.detail.time') || '时间' }}</div>
                <div class="text-xs">{{ formatTime(log.created_at) }}</div>
              </div>
            </div>

            <!-- 2x2 grid -->
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <PreviewPanel
                :title="t('admin.audit.detail.userRequest')"
                :preview="log.request_preview"
                :bytes="log.request_bytes"
                :truncated="log.request_truncated"
              />
              <PreviewPanel
                :title="t('admin.audit.detail.userResponse')"
                :preview="log.response_preview"
                :bytes="log.response_bytes"
                :truncated="log.response_truncated"
              />
              <PreviewPanel
                :title="t('admin.audit.detail.upstreamRequest')"
                :preview="log.upstream_request_preview"
                :bytes="log.upstream_request_bytes"
                :truncated="log.upstream_request_truncated"
                :empty-text="t('admin.audit.detail.upstreamCaptureDisabled')"
              />
              <PreviewPanel
                :title="t('admin.audit.detail.upstreamResponse')"
                :preview="log.upstream_response_preview"
                :bytes="log.upstream_response_bytes"
                :truncated="log.upstream_response_truncated"
                :empty-text="t('admin.audit.detail.upstreamCaptureDisabled')"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { auditAPI, type AuditLogDetail } from '@/api/admin/audit'

const props = defineProps<{ id: number }>()
defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()

const log = ref<AuditLogDetail | null>(null)
const loading = ref(true)

async function load() {
  try {
    log.value = await auditAPI.get(props.id)
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function formatTime(s: string): string {
  return new Date(s).toLocaleString()
}

function platformLabel(p: string): string {
  if (!p) return '-'
  const key = `admin.audit.platform.${p}`
  const v = t(key)
  return v === key ? p : v
}

function statusLabel(code: number): string {
  if (code >= 200 && code < 300) return t('admin.audit.status.success')
  if (code === 429) return t('admin.audit.status.rateLimited')
  if (code >= 400 && code < 500) return t('admin.audit.status.clientError')
  if (code >= 500) return t('admin.audit.status.serverError')
  return String(code)
}

function statusClass(code: number): string {
  if (code >= 200 && code < 300) return 'bg-green-100 text-green-800'
  if (code === 429) return 'bg-yellow-100 text-yellow-800'
  if (code >= 400 && code < 500) return 'bg-orange-100 text-orange-800'
  if (code >= 500) return 'bg-red-100 text-red-800'
  return 'bg-gray-100 text-gray-700'
}

function formatBytes(n: number): string {
  if (!n) return '0 B'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {}
}

const PreviewPanel = (props: any) => {
  const empty = !props.preview || !props.preview.length
  return h('div', { class: 'rounded-lg border border-gray-200 dark:border-gray-700' }, [
    h('div', { class: 'flex items-center justify-between border-b border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900' }, [
      h('div', { class: 'text-sm font-medium' }, props.title),
      h('div', { class: 'flex items-center gap-2 text-xs text-gray-500' }, [
        !empty && h('span', null, formatBytes(props.bytes || 0)),
        props.truncated && h('span', { class: 'rounded bg-yellow-100 px-1.5 py-0.5 text-yellow-700' }, 'truncated'),
        !empty && h('button', { onClick: () => copy(props.preview), class: 'text-primary-600 hover:underline' }, 'copy')
      ])
    ]),
    empty
      ? h('div', { class: 'p-4 text-center text-sm text-gray-400' }, props.emptyText || '-')
      : h('pre', { class: 'max-h-64 overflow-auto p-3 text-xs whitespace-pre-wrap break-all bg-white dark:bg-gray-800' }, props.preview)
  ])
}

onMounted(load)
</script>
