<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.actionDialog.title')"
    width="wide"
    :close-on-click-outside="true"
    @close="emit('close')"
  >
    <div v-if="account" class="space-y-6">
      <header class="flex min-w-0 flex-wrap items-center justify-between gap-3 border-b border-gray-200 pb-4 dark:border-dark-700">
        <div class="min-w-0">
          <p class="truncate text-base font-semibold text-gray-900 dark:text-white" :title="account.name">{{ account.name }}</p>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">#{{ account.id }} · {{ account.platform }} · {{ account.type }}</p>
        </div>
        <span :class="['rounded-md px-2.5 py-1 text-xs font-medium', statusClass]">
          {{ statusLabel }}
        </span>
      </header>

      <section>
        <h4 class="action-section-title">{{ t('admin.accounts.actionDialog.common') }}</h4>
        <div class="action-grid">
          <button class="action-button" autofocus @click="$emit('edit', account); $emit('close')">
            <Icon name="edit" size="md" class="text-primary-600 dark:text-primary-400" />
            <span>{{ t('common.edit') }}</span>
          </button>
          <button class="action-button" @click="$emit('test', account); $emit('close')">
            <Icon name="play" size="md" class="text-emerald-600 dark:text-emerald-400" />
            <span>{{ t('admin.accounts.testConnection') }}</span>
          </button>
          <button class="action-button" @click="$emit('stats', account); $emit('close')">
            <Icon name="chart" size="md" class="text-indigo-600 dark:text-indigo-400" />
            <span>{{ t('admin.accounts.viewStats') }}</span>
          </button>
          <button class="action-button" @click="$emit('schedule', account); $emit('close')">
            <Icon name="clock" size="md" class="text-amber-600 dark:text-amber-400" />
            <span>{{ t('admin.scheduledTests.schedule') }}</span>
          </button>
        </div>
      </section>

      <section>
        <h4 class="action-section-title">{{ t('admin.accounts.actionDialog.maintenance') }}</h4>
        <div class="action-grid">
          <button v-if="canDuplicate" class="action-button" @click="$emit('duplicate', account); $emit('close')">
            <Icon name="copy" size="md" class="text-sky-600 dark:text-sky-400" />
            <span>{{ t('admin.accounts.duplicateAccount') }}</span>
          </button>
          <template v-if="(account.type === 'oauth' || account.type === 'setup-token') && !isShadow">
            <button class="action-button" @click="$emit('reauth', account); $emit('close')">
              <Icon name="link" size="md" class="text-blue-600 dark:text-blue-400" />
              <span>{{ t('admin.accounts.reAuthorize') }}</span>
            </button>
            <button class="action-button" @click="$emit('refresh-token', account); $emit('close')">
              <Icon name="refresh" size="md" class="text-violet-600 dark:text-violet-400" />
              <span>{{ t('admin.accounts.refreshToken') }}</span>
            </button>
          </template>
          <button v-if="hasRecoverableState" class="action-button" @click="$emit('recover-state', account); $emit('close')">
            <Icon name="sync" size="md" class="text-emerald-600 dark:text-emerald-400" />
            <span>{{ t('admin.accounts.recoverState') }}</span>
          </button>
          <button v-if="hasQuotaLimit" class="action-button" @click="$emit('reset-quota', account); $emit('close')">
            <Icon name="refresh" size="md" class="text-teal-600 dark:text-teal-400" />
            <span>{{ t('admin.accounts.resetQuota') }}</span>
          </button>
        </div>
      </section>

      <section v-if="isOpenAIOAuthParent || supportsPrivacy">
        <h4 class="action-section-title">{{ t('admin.accounts.actionDialog.platform') }}</h4>
        <div class="action-grid">
          <button v-if="isOpenAIOAuthParent" class="action-button" @click="$emit('create-spark-shadow', account); $emit('close')">
            <Icon name="sparkles" size="md" class="text-amber-600 dark:text-amber-400" />
            <span>{{ t('admin.accounts.createSparkShadow') }}</span>
          </button>
          <button v-if="supportsPrivacy" class="action-button" @click="$emit('set-privacy', account); $emit('close')">
            <Icon name="shield" size="md" class="text-emerald-600 dark:text-emerald-400" />
            <span>{{ t('admin.accounts.setPrivacy') }}</span>
          </button>
        </div>
      </section>

      <section>
        <h4 class="action-section-title">{{ t('admin.accounts.actionDialog.data') }}</h4>
        <div class="action-grid">
          <button class="action-button" @click="$emit('pool-record', account); $emit('close')">
            <Icon name="chart" size="md" class="text-emerald-600 dark:text-emerald-400" />
            <span>{{ t('admin.sharedPool.actions.poolRecord') }}</span>
          </button>
          <button class="action-button" @click="$emit('credential', account); $emit('close')">
            <Icon name="lock" size="md" class="text-amber-600 dark:text-amber-400" />
            <span>{{ t('admin.sharedPool.approval.viewCredential') }}</span>
          </button>
        </div>
      </section>

      <section class="rounded-lg border border-red-200 bg-red-50/70 p-4 dark:border-red-900/60 dark:bg-red-950/20">
        <h4 class="text-sm font-semibold text-red-800 dark:text-red-300">{{ t('admin.accounts.actionDialog.danger') }}</h4>
        <p class="mt-1 text-xs text-red-700/80 dark:text-red-300/80">{{ t('admin.accounts.actionDialog.deleteHint') }}</p>
        <button class="btn btn-danger mt-3 min-h-11" @click="$emit('delete', account); $emit('close')">
          <Icon name="trash" size="sm" />
          {{ t('common.delete') }}
        </button>
      </section>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { Icon } from '@/components/icons'
import type { Account } from '@/types'

const props = defineProps<{
  show: boolean
  account: Account | null
  position?: { top: number; left: number } | null
}>()
const emit = defineEmits([
  'close', 'edit', 'test', 'stats', 'schedule', 'duplicate', 'reauth', 'refresh-token',
  'recover-state', 'reset-quota', 'set-privacy', 'create-spark-shadow', 'credential',
  'pool-record', 'delete'
])
const { t } = useI18n()

const canDuplicate = computed(() => {
  if (!props.account || props.account.parent_account_id != null) return false
  return ['apikey', 'upstream', 'bedrock', 'service_account'].includes(props.account.type)
})
const isRateLimited = computed(() => {
  if (props.account?.rate_limit_reset_at && new Date(props.account.rate_limit_reset_at) > new Date()) return true
  const limits = (props.account?.extra as Record<string, unknown> | undefined)?.model_rate_limits as
    | Record<string, { rate_limit_reset_at: string }>
    | undefined
  return limits ? Object.values(limits).some(info => new Date(info.rate_limit_reset_at) > new Date()) : false
})
const isOverloaded = computed(() => props.account?.overload_until && new Date(props.account.overload_until) > new Date())
const isTempUnschedulable = computed(() => props.account?.temp_unschedulable_until && new Date(props.account.temp_unschedulable_until) > new Date())
const hasRecoverableState = computed(() => props.account?.status === 'error' || Boolean(isRateLimited.value) || Boolean(isOverloaded.value) || Boolean(isTempUnschedulable.value))
const isAntigravityOAuth = computed(() => props.account?.platform === 'antigravity' && props.account?.type === 'oauth')
const isOpenAIOAuth = computed(() => props.account?.platform === 'openai' && props.account?.type === 'oauth')
const isShadow = computed(() => props.account?.parent_account_id != null)
const isOpenAIOAuthParent = computed(() => isOpenAIOAuth.value && !isShadow.value)
const supportsPrivacy = computed(() => (isAntigravityOAuth.value || isOpenAIOAuth.value) && !isShadow.value)
const hasQuotaLimit = computed(() => (props.account?.type === 'apikey' || props.account?.type === 'bedrock') && (
  (props.account?.quota_limit ?? 0) > 0 ||
  (props.account?.quota_daily_limit ?? 0) > 0 ||
  (props.account?.quota_weekly_limit ?? 0) > 0
))
const statusClass = computed(() => props.account?.status === 'active'
  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  : props.account?.status === 'error'
    ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300')
const statusLabel = computed(() => {
  const statusKeys: Record<string, string> = {
    rate_limited: 'rateLimited',
    temp_unschedulable: 'tempUnschedulable',
    quota_exceeded: 'quotaExceeded'
  }
  const status = props.account?.status || 'inactive'
  return t(`admin.accounts.status.${statusKeys[status] || status}`)
})
</script>

<style scoped>
.action-section-title {
  @apply mb-2 text-xs font-semibold uppercase text-gray-500 dark:text-gray-400;
  letter-spacing: 0;
}

.action-grid {
  @apply grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3;
}

.action-button {
  @apply flex min-h-11 min-w-0 items-center gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-left text-sm font-medium text-gray-800 transition-colors hover:border-primary-300 hover:bg-primary-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-700 dark:hover:bg-primary-900/20;
}
</style>
