<!-- [bmai-fork] audit logs main view with tab navigation -->
<template>
  <div class="audit-view min-h-screen bg-gray-50 dark:bg-gray-900">
    <div class="container mx-auto px-4 py-6">
      <div class="mb-4">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">{{ t('admin.audit.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.audit.description') }}</p>
      </div>

      <div class="mb-4 border-b border-gray-200 dark:border-gray-700">
        <nav class="-mb-px flex space-x-8" aria-label="Tabs">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            @click="activeTab = tab.key"
            class="whitespace-nowrap py-3 px-1 border-b-2 font-medium text-sm transition-colors"
            :class="
              activeTab === tab.key
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-200'
            "
          >
            {{ t(tab.titleKey) }}
          </button>
        </nav>
      </div>

      <div class="rounded-lg bg-white shadow dark:bg-gray-800">
        <AuditLogList v-if="activeTab === 'logs'" />
        <AuditSettings v-else-if="activeTab === 'settings'" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AuditLogList from './audit/AuditLogList.vue'
import AuditSettings from './audit/AuditSettings.vue'

const { t } = useI18n()
const activeTab = ref<'logs' | 'settings'>('logs')
const tabs = [
  { key: 'logs' as const, titleKey: 'admin.audit.tabs.logs' },
  { key: 'settings' as const, titleKey: 'admin.audit.tabs.settings' }
]
</script>
