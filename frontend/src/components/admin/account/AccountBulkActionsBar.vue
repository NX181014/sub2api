<template>
  <div v-if="selectedIds.length" class="sticky bottom-3 z-20 mx-3 mb-4 flex flex-col gap-3 rounded-lg border border-primary-200 bg-white/95 p-3 shadow-lg backdrop-blur dark:border-primary-800 dark:bg-dark-900/95 sm:flex-row sm:items-center sm:justify-between">
    <div class="flex flex-wrap items-center gap-2">
      <span v-if="selectedIds.length > 0" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selectedScope', { count: selectedIds.length, batches: selectedBatchCount }) }}
      </span>
      <template>
        <span v-if="hiddenSelectedCount > 0" class="text-xs text-primary-700 dark:text-primary-300">
          {{ t('admin.accounts.bulkActions.hiddenSelected', { count: hiddenSelectedCount }) }}
        </span>
        <button
          :disabled="busy"
          @click="$emit('toggle-page')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 disabled:cursor-not-allowed disabled:opacity-50 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ allPageSelected ? t('admin.accounts.bulkActions.clearCurrentPage') : t('admin.accounts.bulkActions.selectCurrentPage') }}
          <span v-if="pageSelectedCount > 0 && !allPageSelected">({{ pageSelectedCount }})</span>
        </button>
        <span class="text-gray-300 dark:text-primary-800">•</span>
        <button
          :disabled="busy"
          @click="$emit('clear')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 disabled:cursor-not-allowed disabled:opacity-50 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </template>
    </div>
    <div class="flex w-full flex-wrap gap-2 sm:w-auto sm:justify-end">
      <button :disabled="busy" @click="$emit('edit-selected')" class="btn btn-primary min-h-11 disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.edit') }}</button>
      <button :disabled="busy" @click="showMore = true" class="btn btn-secondary min-h-11 disabled:cursor-not-allowed disabled:opacity-50">{{ t('common.more') }}</button>
    </div>
    <BaseDialog :show="showMore" :title="t('admin.accounts.bulkActions.title')" width="normal" :close-on-click-outside="true" @close="showMore = false">
      <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <button class="bulk-dialog-action" :disabled="busy" @click="$emit('reset-status'); showMore = false">{{ t('admin.accounts.bulkActions.resetStatus') }}</button>
        <button class="bulk-dialog-action" :disabled="busy" @click="$emit('refresh-token'); showMore = false">{{ t('admin.accounts.bulkActions.refreshToken') }}</button>
        <button class="bulk-dialog-action" :disabled="busy" @click="$emit('probe-upstream-billing'); showMore = false">{{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}</button>
        <button class="bulk-dialog-action" :disabled="busy" @click="$emit('toggle-schedulable', true); showMore = false">{{ t('admin.accounts.bulkActions.enableScheduling') }}</button>
        <button class="bulk-dialog-action" :disabled="busy" @click="$emit('toggle-schedulable', false); showMore = false">{{ t('admin.accounts.bulkActions.disableScheduling') }}</button>
      </div>
      <div class="mt-5 border-t border-red-200 pt-4 dark:border-red-900/60">
        <button :disabled="busy" @click="$emit('delete'); showMore = false" class="btn btn-danger min-h-11 disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.delete') }}</button>
      </div>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'

withDefaults(defineProps<{
  selectedIds: number[]
  selectedBatchCount?: number
  filteredCount?: number
  hasActiveFilters?: boolean
  hiddenSelectedCount?: number
  allPageSelected?: boolean
  pageSelectedCount?: number
  busy?: boolean
}>(), {
  filteredCount: 0,
  selectedBatchCount: 0,
  hasActiveFilters: false,
  hiddenSelectedCount: 0,
  allPageSelected: false,
  pageSelectedCount: 0,
  busy: false
})
defineEmits([
  'delete',
  'edit-selected',
  'edit-filtered',
  'clear',
  'toggle-page',
  'toggle-schedulable',
  'reset-status',
  'refresh-token',
  'probe-upstream-billing'
])

const { t } = useI18n()
const showMore = ref(false)
</script>

<style scoped>
.bulk-dialog-action {
  @apply min-h-11 rounded-lg border border-gray-200 px-3 py-2 text-left text-sm font-medium text-gray-800 hover:border-primary-300 hover:bg-primary-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-700 dark:text-gray-100 dark:hover:border-primary-700 dark:hover:bg-primary-900/20;
}
</style>
