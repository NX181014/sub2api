<template>
  <div class="flex w-full flex-wrap items-center justify-between gap-2 border-b border-gray-200 bg-gray-50/70 px-3 py-2 dark:border-dark-700 dark:bg-dark-800/60">
    <div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
      <label
        class="inline-flex min-h-11 items-center gap-2 px-1 text-sm font-medium text-gray-800 dark:text-gray-100"
        :class="busy || filteredCount === 0 ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'"
      >
        <input
          data-test="bulk-page-checkbox"
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="allPageSelected"
          :indeterminate="pageSelectedCount > 0 && !allPageSelected"
          :disabled="busy || filteredCount === 0"
          :aria-label="allPageSelected ? t('admin.accounts.bulkActions.clearCurrentPage') : t('admin.accounts.bulkActions.selectCurrentPage')"
          @change="$emit('toggle-page')"
        />
        <span>
          {{ allPageSelected ? t('admin.accounts.bulkActions.clearCurrentPage') : t('admin.accounts.bulkActions.selectCurrentPage') }}
          <span v-if="pageSelectedCount > 0 && !allPageSelected">({{ pageSelectedCount }})</span>
        </span>
      </label>

      <span class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('admin.accounts.bulkActions.filtered', { count: filteredCount }) }}
      </span>
      <span class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('admin.accounts.bulkActions.currentPage', { count: currentPageCount }) }}
      </span>
      <span v-if="selectedIds.length" class="text-sm font-medium text-primary-700 dark:text-primary-300">
        {{ selectedBatchCount > 0
          ? t('admin.accounts.bulkActions.selectedScope', { count: selectedIds.length, batches: selectedBatchCount })
          : t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-if="hiddenSelectedCount > 0" class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.bulkActions.hiddenSelected', { count: hiddenSelectedCount }) }}
      </span>
      <button
        v-if="selectedIds.length"
        :disabled="busy"
        class="min-h-11 px-2 text-sm font-medium text-primary-700 hover:text-primary-800 disabled:cursor-not-allowed disabled:opacity-50 dark:text-primary-300 dark:hover:text-primary-200"
        @click="$emit('clear')"
      >
        {{ t('admin.accounts.bulkActions.clear') }}
      </button>
    </div>

    <div v-if="selectedIds.length" class="flex w-full flex-wrap gap-2 sm:w-auto sm:justify-end">
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
  hiddenSelectedCount?: number
  allPageSelected?: boolean
  pageSelectedCount?: number
  currentPageCount?: number
  busy?: boolean
}>(), {
  filteredCount: 0,
  selectedBatchCount: 0,
  hiddenSelectedCount: 0,
  allPageSelected: false,
  pageSelectedCount: 0,
  currentPageCount: 0,
  busy: false
})
defineEmits([
  'delete',
  'edit-selected',
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
