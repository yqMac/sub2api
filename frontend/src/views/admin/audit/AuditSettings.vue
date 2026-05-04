<!-- [bmai-fork] audit settings page — card-based layout matching SettingsView -->
<template>
  <div class="space-y-6">
    <div v-if="loading" class="card flex items-center justify-center py-12">
      <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
    </div>
    <template v-else-if="settings">
      <!-- Master Switch -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.audit.settings.master.title') }}</h2>
        </div>
        <div class="space-y-4 p-6">
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.audit.settings.master.enabled') }}</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.audit.settings.master.enabledDesc') }}</div>
            </div>
            <button @click="settings.enabled = !settings.enabled"
              class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors"
              :class="settings.enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-gray-600'">
              <span class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200"
                :class="settings.enabled ? 'translate-x-5' : 'translate-x-0'" />
            </button>
          </div>
        </div>
      </div>

      <!-- Capture Settings -->
      <div class="card" :class="{ 'opacity-50 pointer-events-none': !settings.enabled }">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.audit.settings.capture.title') }}</h2>
        </div>
        <div class="space-y-4 p-6">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.settings.capture.maxRequest') }}</span>
            <div class="flex items-center gap-2">
              <input type="number" v-model.number="settings.max_request_bytes" :min="1024" :step="1024"
                class="w-32 rounded-lg border border-gray-200 px-3 py-1.5 text-sm dark:border-dark-600 dark:bg-dark-700" />
              <span class="text-sm text-gray-500">bytes</span>
            </div>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.settings.capture.maxResponse') }}</span>
            <div class="flex items-center gap-2">
              <input type="number" v-model.number="settings.max_response_bytes" :min="1024" :step="1024"
                class="w-32 rounded-lg border border-gray-200 px-3 py-1.5 text-sm dark:border-dark-600 dark:bg-dark-700" />
              <span class="text-sm text-gray-500">bytes</span>
            </div>
          </div>
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.settings.capture.captureUpstream') }}</div>
              <div class="text-xs text-gray-500">{{ t('admin.audit.settings.capture.captureUpstreamDesc') }}</div>
            </div>
            <button @click="settings.capture_upstream = !settings.capture_upstream"
              class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full transition-colors"
              :class="settings.capture_upstream ? 'bg-primary-500' : 'bg-gray-300 dark:bg-gray-600'">
              <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200"
                :class="settings.capture_upstream ? 'translate-x-4' : 'translate-x-0.5'" />
            </button>
          </div>
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.settings.capture.classifyResponse') }}</div>
              <div class="text-xs text-gray-500">{{ t('admin.audit.settings.capture.classifyResponseDesc') }}</div>
            </div>
            <button @click="settings.classify_response = !settings.classify_response"
              class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full transition-colors"
              :class="settings.classify_response ? 'bg-primary-500' : 'bg-gray-300 dark:bg-gray-600'">
              <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200"
                :class="settings.classify_response ? 'translate-x-4' : 'translate-x-0.5'" />
            </button>
          </div>
        </div>
      </div>

      <!-- Storage -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.audit.settings.storage.title') }}</h2>
        </div>
        <div class="space-y-4 p-6">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.audit.settings.storage.retention') }}</span>
            <div class="flex items-center gap-2">
              <input type="number" v-model.number="settings.retention_days" :min="1" :max="365"
                class="w-24 rounded-lg border border-gray-200 px-3 py-1.5 text-sm dark:border-dark-600 dark:bg-dark-700" />
              <span class="text-sm text-gray-500">days</span>
            </div>
          </div>
          <div v-if="storage" class="grid grid-cols-3 gap-4 rounded-lg bg-gray-50 p-4 dark:bg-dark-700/50">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.audit.settings.storage.totalRows') }}</div>
              <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ formatNumber(storage.TotalRows) }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.audit.settings.storage.totalSize') }}</div>
              <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ formatBytes(storage.TotalBytes) }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.audit.settings.storage.earliest') }}</div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ storage.EarliestRecord ? formatDate(storage.EarliestRecord) : '-' }}</div>
            </div>
          </div>
          <button @click="manualCleanup"
            class="rounded-lg border border-red-200 bg-red-50 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-100 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
            {{ t('admin.audit.settings.storage.manualCleanup') }}
          </button>
        </div>
      </div>

      <!-- Save -->
      <div class="flex justify-end">
        <button @click="save" :disabled="saving"
          class="rounded-lg bg-primary-500 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-primary-600 disabled:opacity-50">
          {{ saving ? '...' : t('admin.audit.settings.save') }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { auditAPI, type AuditSettings, type AuditStorageInfo } from '@/api/admin/audit'

const { t } = useI18n()
const settings = ref<AuditSettings | null>(null)
const storage = ref<AuditStorageInfo | null>(null)
const loading = ref(true)
const saving = ref(false)

async function load() {
  try {
    settings.value = await auditAPI.getSettings()
    storage.value = await auditAPI.getStorage()
  } catch (e) {
    console.error('failed to load audit settings', e)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!settings.value) return
  saving.value = true
  try {
    settings.value = await auditAPI.updateSettings(settings.value)
    alert(t('admin.audit.settings.saved'))
  } catch (e) {
    console.error(e)
    alert('Save failed')
  } finally {
    saving.value = false
  }
}

async function manualCleanup() {
  const days = prompt(t('admin.audit.settings.storage.cleanupConfirm', { days: settings.value?.retention_days || 30 }), String(settings.value?.retention_days || 30))
  if (!days) return
  const n = parseInt(days, 10)
  if (!n || n < 1) return
  try {
    const res = await auditAPI.cleanup(n)
    alert(`Deleted ${res.deleted} rows`)
    storage.value = await auditAPI.getStorage()
  } catch (e) { console.error(e) }
}

function formatNumber(n: number): string { return n.toLocaleString() }
function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`
}
function formatDate(s: string): string { return new Date(s).toLocaleDateString() }

onMounted(load)
</script>
