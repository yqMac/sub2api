<!-- [bmai-fork] audit settings page -->
<template>
  <div class="p-6">
    <div v-if="loading" class="py-12 text-center text-gray-500">Loading...</div>
    <div v-else-if="settings" class="space-y-8">
      <!-- Master Switch -->
      <section>
        <h3 class="mb-4 text-lg font-medium text-gray-900 dark:text-gray-100">
          {{ t('admin.audit.settings.master.title') }}
        </h3>
        <div class="flex items-center justify-between rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
          <div>
            <div class="font-medium text-gray-900 dark:text-gray-100">
              {{ t('admin.audit.settings.master.enabled') }}
            </div>
            <div class="text-sm text-gray-500">
              {{ t('admin.audit.settings.master.enabledDesc') }}
            </div>
          </div>
          <button
            @click="settings.enabled = !settings.enabled"
            class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors"
            :class="settings.enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-gray-600'"
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200"
              :class="settings.enabled ? 'translate-x-5' : 'translate-x-0'"
            />
          </button>
        </div>
      </section>

      <!-- Capture Settings -->
      <section :class="{ 'opacity-50 pointer-events-none': !settings.enabled }">
        <h3 class="mb-4 text-lg font-medium text-gray-900 dark:text-gray-100">
          {{ t('admin.audit.settings.capture.title') }}
        </h3>
        <div class="space-y-3 rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
          <SettingNumberRow :label="t('admin.audit.settings.capture.maxRequest')" v-model="settings.max_request_bytes" :min="1024" :step="1024" suffix="bytes" />
          <SettingNumberRow :label="t('admin.audit.settings.capture.maxResponse')" v-model="settings.max_response_bytes" :min="1024" :step="1024" suffix="bytes" />
          <SettingToggleRow
            :label="t('admin.audit.settings.capture.captureUpstream')"
            :description="t('admin.audit.settings.capture.captureUpstreamDesc')"
            v-model="settings.capture_upstream"
          />
          <SettingToggleRow
            :label="t('admin.audit.settings.capture.classifyResponse')"
            :description="t('admin.audit.settings.capture.classifyResponseDesc')"
            v-model="settings.classify_response"
          />
        </div>
      </section>

      <!-- Storage -->
      <section>
        <h3 class="mb-4 text-lg font-medium text-gray-900 dark:text-gray-100">
          {{ t('admin.audit.settings.storage.title') }}
        </h3>
        <div class="space-y-3 rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
          <SettingNumberRow :label="t('admin.audit.settings.storage.retention')" v-model="settings.retention_days" :min="1" :max="365" suffix="days" />
          <div v-if="storage" class="grid grid-cols-3 gap-4 border-t border-gray-200 pt-3 dark:border-gray-700">
            <div>
              <div class="text-sm text-gray-500">{{ t('admin.audit.settings.storage.totalRows') }}</div>
              <div class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ formatNumber(storage.TotalRows) }}</div>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('admin.audit.settings.storage.totalSize') }}</div>
              <div class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ formatBytes(storage.TotalBytes) }}</div>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('admin.audit.settings.storage.earliest') }}</div>
              <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ storage.EarliestRecord ? formatDate(storage.EarliestRecord) : '-' }}
              </div>
            </div>
          </div>
          <div class="border-t border-gray-200 pt-3 dark:border-gray-700">
            <button
              @click="manualCleanup"
              class="rounded-md border border-red-300 bg-red-50 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-100"
            >
              {{ t('admin.audit.settings.storage.manualCleanup') }}
            </button>
          </div>
        </div>
      </section>

      <!-- Save -->
      <div class="flex justify-end border-t border-gray-200 pt-4 dark:border-gray-700">
        <button
          @click="save"
          :disabled="saving"
          class="rounded-md bg-primary-500 px-6 py-2 text-sm font-medium text-white hover:bg-primary-600 disabled:opacity-50"
        >
          {{ saving ? '...' : t('admin.audit.settings.save') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
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
  } catch (e) {
    console.error(e)
  }
}

function formatNumber(n: number): string {
  return n.toLocaleString()
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function formatDate(s: string): string {
  return new Date(s).toLocaleDateString()
}

// Inline tiny components to avoid extra files
const SettingNumberRow = (props: any) => h('div', { class: 'flex items-center justify-between' }, [
  h('div', { class: 'text-sm font-medium text-gray-700 dark:text-gray-300' }, props.label),
  h('div', { class: 'flex items-center gap-2' }, [
    h('input', {
      type: 'number',
      value: props.modelValue,
      min: props.min,
      max: props.max,
      step: props.step,
      onInput: (e: any) => props['onUpdate:modelValue']?.(Number(e.target.value)),
      class: 'w-32 rounded border border-gray-300 px-3 py-1 text-sm dark:border-gray-600 dark:bg-gray-800'
    }),
    props.suffix && h('span', { class: 'text-sm text-gray-500' }, props.suffix)
  ])
])

const SettingToggleRow = (props: any) => h('div', { class: 'flex items-center justify-between' }, [
  h('div', null, [
    h('div', { class: 'text-sm font-medium text-gray-700 dark:text-gray-300' }, props.label),
    props.description && h('div', { class: 'text-xs text-gray-500' }, props.description)
  ]),
  h('button', {
    type: 'button',
    onClick: () => props['onUpdate:modelValue']?.(!props.modelValue),
    class: ['relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full transition-colors',
      props.modelValue ? 'bg-primary-500' : 'bg-gray-300 dark:bg-gray-600']
  }, h('span', {
    class: ['pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200',
      props.modelValue ? 'translate-x-5' : 'translate-x-0.5']
  }))
])

onMounted(load)
</script>
