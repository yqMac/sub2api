<!-- [bmai-fork] audit logs main view — uses AppLayout + settings-tabs pattern -->
<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <!-- Tab Navigation -->
      <div class="sticky top-0 z-10 overflow-x-auto settings-tabs-scroll">
        <nav class="settings-tabs">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            :class="['settings-tab', activeTab === tab.key && 'settings-tab-active']"
            @click="activeTab = tab.key"
          >
            <span>{{ t(tab.titleKey) }}</span>
          </button>
        </nav>
      </div>

      <!-- Tab Content -->
      <div v-show="activeTab === 'logs'">
        <AuditLogList />
      </div>
      <div v-show="activeTab === 'settings'">
        <AuditSettings />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import AuditLogList from './audit/AuditLogList.vue'
import AuditSettings from './audit/AuditSettings.vue'

const { t } = useI18n()
const activeTab = ref<'logs' | 'settings'>('logs')
const tabs = [
  { key: 'logs' as const, titleKey: 'admin.audit.tabs.logs' },
  { key: 'settings' as const, titleKey: 'admin.audit.tabs.settings' }
]
</script>

<style scoped>
.settings-tabs-scroll {
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
}
.settings-tabs-scroll:hover {
  scrollbar-color: rgb(0 0 0 / 0.15) transparent;
}
:root.dark .settings-tabs-scroll:hover {
  scrollbar-color: rgb(255 255 255 / 0.2) transparent;
}
.settings-tabs {
  @apply inline-flex min-w-full gap-0.5 rounded-2xl
         border border-gray-100 bg-white/80 p-1 backdrop-blur-sm
         dark:border-dark-700/50 dark:bg-dark-800/80;
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.04), 0 1px 2px rgb(0 0 0 / 0.02);
}
@media (min-width: 640px) {
  .settings-tabs { @apply flex; }
}
.settings-tab {
  @apply relative flex flex-1 items-center justify-center gap-1.5
         whitespace-nowrap rounded-xl px-2.5 py-2
         text-sm font-medium text-gray-500 dark:text-dark-400
         transition-all duration-200 ease-out;
}
.settings-tab:hover:not(.settings-tab-active) {
  @apply text-gray-700 dark:text-gray-300;
  background: rgb(0 0 0 / 0.03);
}
:root.dark .settings-tab:hover:not(.settings-tab-active) {
  background: rgb(255 255 255 / 0.04);
}
.settings-tab-active {
  @apply text-primary-600 dark:text-primary-400;
  background: linear-gradient(135deg, rgba(20, 184, 166, 0.08), rgba(20, 184, 166, 0.03));
  box-shadow: 0 1px 2px rgba(20, 184, 166, 0.1);
}
:root.dark .settings-tab-active {
  background: linear-gradient(135deg, rgba(45, 212, 191, 0.12), rgba(45, 212, 191, 0.05));
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.25);
}
</style>
