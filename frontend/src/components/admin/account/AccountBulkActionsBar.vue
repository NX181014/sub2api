<template>
  <div class="mb-4 flex flex-col gap-3 rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20 sm:flex-row sm:items-center sm:justify-between">
    <div class="flex flex-wrap items-center gap-2">
      <span v-if="selectedIds.length > 0" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-else class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.filtered', { count: filteredCount }) }}
      </span>
      <template v-if="selectedIds.length > 0">
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
      <template v-if="selectedIds.length > 0">
        <button :disabled="busy" @click="$emit('delete')" class="btn btn-danger btn-sm disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.delete') }}</button>
        <button :disabled="busy" @click="$emit('reset-status')" class="btn btn-secondary btn-sm disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.resetStatus') }}</button>
        <button :disabled="busy" @click="$emit('refresh-token')" class="btn btn-secondary btn-sm disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.refreshToken') }}</button>
        <button :disabled="busy" @click="$emit('probe-upstream-billing')" class="btn btn-secondary btn-sm disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}</button>
        <button :disabled="busy" @click="$emit('toggle-schedulable', true)" class="btn btn-success btn-sm disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.enableScheduling') }}</button>
        <button :disabled="busy" @click="$emit('toggle-schedulable', false)" class="btn btn-warning btn-sm disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.disableScheduling') }}</button>
        <button :disabled="busy" @click="$emit('edit-selected')" class="btn btn-primary btn-sm disabled:cursor-not-allowed disabled:opacity-50">{{ t('admin.accounts.bulkActions.edit') }}</button>
      </template>
      <button v-else-if="hasActiveFilters" :disabled="busy" @click="$emit('edit-filtered')" class="btn btn-primary btn-sm disabled:cursor-not-allowed disabled:opacity-50">
        {{ t('admin.accounts.bulkEdit.submit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

withDefaults(defineProps<{
  selectedIds: number[]
  filteredCount?: number
  hasActiveFilters?: boolean
  hiddenSelectedCount?: number
  allPageSelected?: boolean
  pageSelectedCount?: number
  busy?: boolean
}>(), {
  filteredCount: 0,
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
</script>
