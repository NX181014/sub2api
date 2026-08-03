<template>
  <div class="account-filter-controls min-w-0 flex-1 space-y-3">
    <div class="flex flex-wrap items-start gap-2">
      <div class="min-w-0 w-full sm:min-w-[240px] sm:flex-1">
        <SearchInput
          :model-value="searchQuery"
          :placeholder="t('admin.accounts.searchAccounts')"
          class="w-full"
          @update:model-value="$emit('update:searchQuery', $event)"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.searchAccountsHint') }}</p>
      </div>
      <Select :model-value="filters.platform" class="min-w-36 flex-1 sm:max-w-40" :options="pOpts" :aria-label="t('admin.accounts.columns.platform')" @update:model-value="updatePlatform" />
      <Select :model-value="filters.status" class="min-w-36 flex-1 sm:max-w-40" :options="sOpts" :aria-label="t('admin.accounts.columns.status')" @update:model-value="updateStatus" />
      <Select :model-value="filters.uploader_user_id" class="min-w-40 flex-1 sm:max-w-48" :options="uploaderOpts" :aria-label="t('admin.sharedPool.columns.uploader')" searchable @update:model-value="updateUploader" />
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :aria-expanded="showAdvanced"
        @click="showAdvanced = !showAdvanced"
      >
        {{ t('admin.accounts.moreFilters') }}
        <span v-if="advancedFilterCount" class="ml-1 rounded-full bg-primary-100 px-1.5 text-xs text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">{{ advancedFilterCount }}</span>
      </button>
      <button v-if="activeFilterCount" type="button" class="btn btn-secondary btn-sm" @click="$emit('clear')">
        {{ t('admin.accounts.clearFilters', { count: activeFilterCount }) }}
      </button>
    </div>

    <div v-if="showAdvanced" class="flex flex-wrap items-center gap-2 border-t border-gray-200 pt-3 dark:border-dark-700">
      <Select :model-value="filters.type" class="w-40" :options="tOpts" :aria-label="t('admin.accounts.columns.type')" @update:model-value="updateType" />
      <Select :model-value="filters.privacy_mode" class="w-44" :options="privacyOpts" :aria-label="t('admin.accounts.privacyLabel')" @update:model-value="updatePrivacyMode" />
      <Select :model-value="filters.group" class="w-40" :options="gOpts" :aria-label="t('admin.accounts.columns.groups')" @update:model-value="updateGroup" />
    </div>

    <div v-if="activeFilterCount || resultAccountCount || resultBatchCount" class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
      <span class="font-medium text-gray-700 dark:text-gray-200">{{ t('admin.accounts.resultSummary', { accounts: resultAccountCount, batches: resultBatchCount }) }}</span>
      <template v-for="chip in activeChips" :key="chip.key">
        <button type="button" class="inline-flex min-h-8 items-center gap-1 rounded-full bg-primary-50 px-2.5 py-1 text-primary-700 hover:bg-primary-100 dark:bg-primary-900/30 dark:text-primary-300" @click="clearChip(chip.key)">
          <Icon name="x" size="xs" class="shrink-0" aria-hidden="true" />
          {{ chip.label }}: {{ chip.value }}
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup } from '@/types'

const props = withDefaults(defineProps<{
  searchQuery: string
  filters: Record<string, any>
  groups?: AdminGroup[]
  uploaders?: Array<{ value: number; label: string }>
  resultAccountCount?: number
  resultBatchCount?: number
}>(), {
  resultAccountCount: 0,
  resultBatchCount: 0
})
const emit = defineEmits(['update:searchQuery', 'update:filters', 'clear'])
const { t } = useI18n()
const showAdvanced = ref(false)
const activeFilterCount = computed(() => [
  props.searchQuery.trim(),
  props.filters.platform,
  props.filters.type,
  props.filters.status,
  props.filters.privacy_mode,
  props.filters.group,
  props.filters.uploader_user_id
].filter(Boolean).length)
const advancedFilterCount = computed(() => [props.filters.type, props.filters.privacy_mode, props.filters.group].filter(Boolean).length)
const activeChips = computed(() => {
  const chips: Array<{ key: string; label: string; value: string }> = []
  if (props.searchQuery.trim()) chips.push({ key: 'search', label: t('admin.accounts.searchLabel'), value: props.searchQuery.trim() })
  const values: Array<[string, string, unknown]> = [
    ['platform', t('admin.accounts.columns.platform'), props.filters.platform],
    ['status', t('admin.accounts.columns.status'), props.filters.status],
    ['uploader_user_id', t('admin.sharedPool.columns.uploader'), props.filters.uploader_user_id],
    ['type', t('admin.accounts.columns.type'), props.filters.type],
    ['privacy_mode', t('admin.accounts.privacyLabel'), props.filters.privacy_mode],
    ['group', t('admin.accounts.columns.groups'), props.filters.group]
  ]
  for (const [key, label, rawValue] of values) {
    if (!rawValue) continue
    const option = key === 'platform' ? pOpts.value.find(item => item.value === rawValue)
      : key === 'status' ? sOpts.value.find(item => item.value === rawValue)
        : key === 'type' ? tOpts.value.find(item => item.value === rawValue)
          : key === 'privacy_mode' ? privacyOpts.value.find(item => item.value === rawValue)
            : key === 'group' ? gOpts.value.find(item => item.value === rawValue)
              : uploaderOpts.value.find(item => item.value === rawValue)
    chips.push({ key, label, value: option?.label || String(rawValue) })
  }
  return chips
})
const clearChip = (key: string) => {
  if (key === 'search') {
    emit('update:searchQuery', '')
    return
  }
  emit('update:filters', { ...props.filters, [key]: '' })
}
const updatePlatform = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, platform: value }) }
const updateType = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, type: value }) }
const updateStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, status: value }) }
const updatePrivacyMode = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, privacy_mode: value }) }
const updateGroup = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, group: value }) }
const updateUploader = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, uploader_user_id: value }) }
const pOpts = computed(() => [{ value: '', label: t('admin.accounts.allPlatforms') }, { value: 'anthropic', label: 'Anthropic' }, { value: 'openai', label: 'OpenAI' }, { value: 'gemini', label: 'Gemini' }, { value: 'antigravity', label: 'Antigravity' }, { value: 'grok', label: 'Grok' }])
const tOpts = computed(() => [{ value: '', label: t('admin.accounts.allTypes') }, { value: 'oauth', label: t('admin.accounts.oauthType') }, { value: 'setup-token', label: t('admin.accounts.setupToken') }, { value: 'apikey', label: t('admin.accounts.apiKey') }, { value: 'bedrock', label: 'AWS Bedrock' }])
const sOpts = computed(() => [{ value: '', label: t('admin.accounts.allStatus') }, { value: 'active', label: t('admin.accounts.status.active') }, { value: 'inactive', label: t('admin.accounts.status.inactive') }, { value: 'error', label: t('admin.accounts.status.error') }, { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') }, { value: 'overloaded', label: t('admin.accounts.status.overloaded') }, { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') }, { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') }])
const privacyOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPrivacyModes') },
  { value: '__unset__', label: t('admin.accounts.privacyUnset') },
  { value: 'training_off', label: t('admin.accounts.privacyTrainingOff') },
  { value: 'training_set_cf_blocked', label: t('admin.accounts.privacyCfBlocked') },
  { value: 'training_set_failed', label: t('admin.accounts.privacyFailed') },
  { value: 'privacy_set', label: t('admin.accounts.privacyAntigravitySet') },
  { value: 'privacy_set_failed', label: t('admin.accounts.privacyAntigravityFailed') }
])
const gOpts = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...(props.groups || []).map(g => ({ value: String(g.id), label: g.name }))
])
const uploaderOpts = computed(() => [
  { value: '', label: t('admin.sharedPool.ledger.allUploaders') },
  { value: 'unassigned', label: t('admin.accounts.unassignedUploader') },
  ...(props.uploaders || [])
])
</script>

<style scoped>
.account-filter-controls :deep(.select-trigger),
.account-filter-controls :deep(.input),
.account-filter-controls .btn {
  min-height: 2.75rem;
}

</style>
