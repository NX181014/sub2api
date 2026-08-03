<template>
  <Teleport to="body">
    <div v-if="show" class="fixed inset-0 z-50" @keydown="handleKeydown">
      <button
        type="button"
        class="absolute inset-0 h-full w-full bg-black/45"
        :aria-label="t('common.close')"
        @click="emit('close')"
      ></button>
      <section
        ref="panelRef"
        class="absolute inset-0 flex min-w-0 flex-col bg-white shadow-xl dark:bg-dark-900 sm:inset-4 sm:rounded-lg sm:border sm:border-gray-200 sm:dark:border-dark-700 lg:inset-x-[max(2rem,calc((100vw-80rem)/2))] lg:inset-y-8"
        role="dialog"
        aria-modal="true"
        aria-labelledby="account-trace-title"
        tabindex="-1"
      >
        <header class="shrink-0 border-b border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900 sm:rounded-t-lg sm:px-5">
          <div class="flex min-w-0 items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.actions.poolRecord') }}</p>
              <h2 id="account-trace-title" class="mt-0.5 max-w-[min(70vw,720px)] truncate text-lg font-semibold text-gray-900 dark:text-white" :title="account?.name || `#${accountId}`">
                {{ account?.name || `#${accountId}` }}
              </h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">#{{ accountId }}</p>
            </div>
            <button
              type="button"
              class="inline-flex h-11 w-11 shrink-0 items-center justify-center text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
              :title="t('common.close')"
              :aria-label="t('common.close')"
              @click="emit('close')"
            >
              <Icon name="x" size="md" />
            </button>
          </div>

          <div v-if="account" class="mt-3 grid min-w-0 gap-3 border-t border-gray-100 pt-3 text-sm dark:border-dark-700 lg:grid-cols-[minmax(0,1.25fr)_minmax(280px,.75fr)] lg:items-center">
            <dl class="grid min-w-0 grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-4">
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.platformType') }}</dt><dd class="mt-1"><PlatformTypeBadge :platform="account.platform" :type="account.type" /></dd></div>
              <div class="min-w-0"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploader') }}</dt><dd class="mt-1 truncate font-medium" :title="uploaderName">{{ uploaderName }}</dd></div>
              <div class="min-w-0"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.page.importBatch') }}</dt><dd class="mt-1 truncate font-medium" :title="importBatch">{{ importBatch }}</dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploadedAt') }}</dt><dd class="mt-1 font-medium">{{ formatDate(account.created_at) }}</dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.status') }}</dt><dd class="mt-1"><AccountStatusIndicator :account="account" /></dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.schedulable') }}</dt><dd class="mt-1"><StatusBadge :status="effectiveSchedulable ? 'success' : 'warning'" :label="t(effectiveSchedulable ? 'admin.accounts.schedulableEnabled' : 'admin.accounts.schedulableDisabled')" /></dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.impacts.lifecycle_events') }}</dt><dd class="mt-1"><StatusBadge :status="lifecycleState.badge" :label="lifecycleState.label" /></dd></div>
              <p v-if="runtimeReason" class="col-span-2 break-words text-xs text-amber-700 dark:text-amber-300 sm:col-span-4">{{ runtimeReason }}</p>
            </dl>
            <div class="min-w-0 border-t border-gray-100 pt-3 dark:border-dark-700 lg:border-l lg:border-t-0 lg:pl-4 lg:pt-0">
              <div class="flex items-center justify-between gap-2"><p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accounts.columns.usageWindows') }}</p><AccountCapacityCell class="shrink-0" :account="account" /></div>
              <div class="mt-2 min-w-0"><AccountUsageCell :account="account" /></div>
            </div>
          </div>
        </header>

        <nav class="grid shrink-0 grid-cols-3 border-b border-gray-200 px-2 dark:border-dark-700 sm:grid-cols-5 sm:px-4" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="-mb-px min-h-11 min-w-0 border-b-2 px-2 text-sm font-medium sm:px-3"
            :class="activeTab === tab.key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400'"
            role="tab"
            :aria-selected="activeTab === tab.key"
            @click="activeTab = tab.key"
          >
            {{ tab.label }}
          </button>
        </nav>

        <div class="min-h-0 min-w-0 flex-1 overflow-y-auto p-3 pb-[max(1rem,env(safe-area-inset-bottom))] sm:p-5">
          <div v-if="loading" class="flex min-h-48 items-center justify-center"><LoadingSpinner /></div>
          <div
            v-else-if="dataQualityNotices.length"
            class="mb-4 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-300"
          >
            <p class="font-medium">{{ t('admin.sharedPool.intake.pending') }}</p>
            <p v-for="notice in dataQualityNotices" :key="notice" class="mt-1">{{ notice }}</p>
          </div>

          <template v-if="!loading && activeTab === 'costs'">
            <div v-if="entries.length" class="divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
              <article v-for="row in entries" :key="row.id" class="grid min-w-0 gap-3 p-4 sm:grid-cols-2 lg:grid-cols-[minmax(140px,1fr)_minmax(140px,1fr)_repeat(3,minmax(110px,.75fr))] lg:items-center">
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.costType') }}</span><span class="mt-1 block font-semibold">{{ t(`admin.sharedPool.entryTypes.${row.entry_type}`) }}</span></div>
                <div class="min-w-0"><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.source') }}</span><span class="mt-1 block truncate font-medium" :title="row.purchase_source || undefined">{{ row.purchase_source || '-' }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.payer') }}</span><span class="mt-1 block truncate" :title="row.payer_email">{{ row.payer_email || '-' }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.originalAmount') }}</span><span class="mt-1 block font-semibold tabular-nums">{{ formatMoney(Number(row.original_amount), row.currency) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.servicePeriod') }}</span><span class="mt-1 block text-xs">{{ dateOnly(row.service_start) }} - {{ dateOnly(row.service_end) }}</span><span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ formatDate(row.paid_at) }}</span></div>
              </article>
            </div>
            <EmptyState v-if="!entries.length" :title="t('admin.sharedPool.ledger.emptyEntries')" />
            <Pagination
              v-if="entriesTotal > 20"
              class="mt-4"
              :page="entriesPage"
              :page-size="20"
              :total="entriesTotal"
              @update:page="emit('entry-page', $event)"
            />
          </template>

          <template v-else-if="!loading && activeTab === 'settlement'">
            <div v-if="settlement" class="mb-4 flex flex-wrap items-center justify-between gap-2 border-b border-gray-100 pb-3 text-sm dark:border-dark-700">
              <span>{{ dateOnly(settlement.period_start) }} - {{ dateOnly(settlement.period_end) }}</span>
              <StatusBadge :status="settlementState.badge" :label="t(`admin.sharedPool.status.${settlementState.key}`)" />
            </div>
            <div v-if="accountLines.length" class="divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
              <article v-for="row in accountLines" :key="row.id" class="grid min-w-0 gap-3 p-4 sm:grid-cols-2 lg:grid-cols-[minmax(140px,1fr)_repeat(4,minmax(100px,.75fr))] lg:items-center">
                <div class="min-w-0"><span class="block truncate font-semibold" :title="row.user_name">{{ row.user_name }}</span><StatusBadge class="mt-1" :status="row.trace_quality === 'exact' ? 'success' : 'warning'" :label="row.trace_quality" /></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.usageWeight') }}</span><span class="mt-1 block tabular-nums">{{ formatMoney(row.account_usage_weight) }}</span><span class="text-xs text-gray-500 dark:text-gray-400">{{ formatPercent(row.usage_share) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.allocated') }}</span><span class="mt-1 block tabular-nums">{{ formatMoney(row.allocated_cost) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.credit') }}</span><span class="mt-1 block tabular-nums">{{ formatMoney(row.contribution_credit) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.net') }}</span><span class="mt-1 block font-semibold tabular-nums">{{ formatMoney(row.net_amount) }}</span></div>
              </article>
            </div>
            <div v-if="accountCosts.length" class="mt-5 border-t border-gray-100 pt-4 dark:border-dark-700">
              <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.columns.cost') }}</h3>
              <div class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
                <div v-for="cost in accountCosts" :key="cost.id" class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 py-2">
                  <span class="min-w-0 truncate">#{{ cost.cost_entry_id }} · {{ cost.kind }}</span>
                  <span class="font-medium tabular-nums">{{ formatMoney(cost.amount) }}</span>
                </div>
              </div>
            </div>
            <EmptyState v-if="!accountLines.length && !accountCosts.length" :title="t('admin.sharedPool.empty.settlement')" />
            <Pagination
              v-if="settlementTotal > 1"
              class="mt-4"
              :page="settlementPage"
              :page-size="1"
              :show-page-size-selector="false"
              :total="settlementTotal"
              @update:page="emit('settlement-page', $event)"
            />
          </template>

          <template v-else-if="!loading && activeTab === 'payback'">
            <dl v-if="recovery" class="grid grid-cols-2 gap-4 text-sm sm:grid-cols-3">
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.cost') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ formatMoney(recovery.purchase_cost, recovery.currency) }}</dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.usageValue') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ formatMoney(recovery.usage_value, recovery.currency) }}</dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.roi') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ formatPercent(recovery.roi_rate) }}</dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.remaining') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ formatMoney(recovery.remaining_cost, recovery.currency) }}</dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.netProfit') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ formatMoney(recovery.net_profit || 0, recovery.currency) }}</dd></div>
              <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.recoveredAt') }}</dt><dd class="mt-1 font-semibold">{{ formatDate(recovery.recovered_at) }}</dd></div>
            </dl>
            <EmptyState v-else :title="t('admin.sharedPool.empty.overview')" />
          </template>

          <template v-else-if="!loading && activeTab === 'lifecycle'">
            <ol v-if="lifecycle.length" class="divide-y divide-gray-100 dark:divide-dark-700">
              <li v-for="event in lifecycle" :key="event.id" class="py-3 first:pt-0">
                <div class="flex items-start justify-between gap-3">
                  <span class="font-medium">{{ lifecycleLabel(event.event_type) }}</span>
                  <time class="shrink-0 text-xs text-gray-500 dark:text-gray-400">{{ formatDate(event.occurred_at) }}</time>
                </div>
                <p v-if="event.reason" class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ event.reason }}</p>
              </li>
            </ol>
            <EmptyState v-else :title="t('admin.sharedPool.approval.impacts.lifecycle_events')" />
          </template>

          <template v-else-if="!loading">
            <ol v-if="approvals.length" class="divide-y divide-gray-100 dark:divide-dark-700">
              <li v-for="approval in approvals" :key="approval.id" class="py-3 first:pt-0">
                <div class="flex items-start justify-between gap-3">
                  <span class="font-medium">{{ approvalLabel(approval.action_type) }}</span>
                  <StatusBadge :status="approvalState(approval.status)" :label="t(`admin.sharedPool.approval.${approval.status}`)" />
                </div>
                <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ approval.reason || '-' }}</p>
                <time class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ formatDate(approval.requested_at) }}</time>
              </li>
            </ol>
            <EmptyState v-else :title="t('admin.sharedPool.approval.empty')" />
            <Pagination
              v-if="approvalsTotal > 20"
              class="mt-4"
              :page="approvalsPage"
              :page-size="20"
              :total="approvalsTotal"
              @update:page="emit('approval-page', $event)"
            />
          </template>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
import type {
  PoolApproval,
  PoolApprovalAction,
  PoolApprovalStatus,
  SharedPoolAccountCost,
  SharedPoolLedgerEntry,
  SharedPoolLifecycleEvent,
  SharedPoolSettlementPreview
} from '@/api/admin/sharedPool'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import { EmptyState, LoadingSpinner, Pagination } from '@/components/common'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTimeToMinute } from '@/utils/format'
import { formatPoolMoney, settlementStatusPresentation } from '@/utils/sharedPool'

type TraceTab = 'costs' | 'settlement' | 'payback' | 'lifecycle' | 'approvals'
const props = defineProps<{
  show: boolean
  loading: boolean
  accountId: number
  account: Account | null
  entries: SharedPoolLedgerEntry[]
  entriesPage: number
  entriesTotal: number
  settlement: SharedPoolSettlementPreview | null
  settlementPage: number
  settlementTotal: number
  recovery: SharedPoolAccountCost | null
  lifecycle: SharedPoolLifecycleEvent[]
  approvals: PoolApproval[]
  approvalsPage: number
  approvalsTotal: number
}>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'entry-page', page: number): void
  (event: 'settlement-page', page: number): void
  (event: 'approval-page', page: number): void
}>()
const { t, locale } = useI18n()
const activeTab = ref<TraceTab>('costs')
const panelRef = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null
let bodyLocked = false

const focusableElements = () => Array.from(panelRef.value?.querySelectorAll<HTMLElement>(
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
) || [])
const releaseBodyLock = () => {
  if (!bodyLocked) return
  const count = Math.max(0, Number(document.body.dataset.modalOpenCount || 1) - 1)
  if (count) document.body.dataset.modalOpenCount = String(count)
  else {
    delete document.body.dataset.modalOpenCount
    document.body.classList.remove('modal-open')
  }
  bodyLocked = false
}
const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    emit('close')
    return
  }
  if (event.key !== 'Tab') return
  const focusable = focusableElements()
  if (!focusable.length) {
    event.preventDefault()
    panelRef.value?.focus()
  } else if (event.shiftKey && document.activeElement === focusable[0]) {
    event.preventDefault()
    focusable[focusable.length - 1].focus()
  } else if (!event.shiftKey && document.activeElement === focusable[focusable.length - 1]) {
    event.preventDefault()
    focusable[0].focus()
  }
}

watch(() => props.show, async (show) => {
  if (show) {
    previousActiveElement = document.activeElement as HTMLElement
    const count = Number(document.body.dataset.modalOpenCount || 0) + 1
    document.body.dataset.modalOpenCount = String(count)
    document.body.classList.add('modal-open')
    bodyLocked = true
    await nextTick()
    ;(focusableElements()[0] || panelRef.value)?.focus()
  } else {
    releaseBodyLock()
    previousActiveElement?.focus()
    previousActiveElement = null
  }
})
onUnmounted(releaseBodyLock)

watch(() => props.accountId, () => { activeTab.value = 'costs' })

const tabs = computed(() => [
  { key: 'costs' as const, label: t('admin.sharedPool.tabs.ledger') },
  { key: 'settlement' as const, label: t('admin.sharedPool.tabs.settlement') },
  { key: 'payback' as const, label: t('admin.sharedPool.tabs.overview') },
  { key: 'lifecycle' as const, label: t('admin.sharedPool.approval.impacts.lifecycle_events') },
  { key: 'approvals' as const, label: t('admin.sharedPool.approval.title') }
])
const uploaderName = computed(() => props.account?.uploader_username || props.account?.uploader_email || '-')
const importBatch = computed(() => String(props.account?.extra?.import_batch_id || t('admin.sharedPool.page.singleImport')))
const latestLifecycle = computed(() => props.lifecycle[0])
const lifecycleState = computed(() => {
  const event = latestLifecycle.value?.event_type
  if (!event) return { badge: 'inactive', label: '-' }
  const badge = event === 'banned_confirmed' ? 'danger' : event === 'refund' ? 'warning' : event === 'recovered' ? 'success' : 'inactive'
  return { badge, label: lifecycleLabel(event) }
})
const runtimeReason = computed(() => {
  const account = props.account
  if (!account) return ''
  if (account.error_message) return account.error_message
  if (account.temp_unschedulable_reason) return account.temp_unschedulable_reason
  const until = account.rate_limit_reset_at || account.overload_until || account.temp_unschedulable_until
  return until ? formatDate(until) : ''
})
const effectiveSchedulable = computed(() => {
  const account = props.account
  if (!account || account.status !== 'active' || !account.schedulable) return false
  const now = Date.now()
  if ([account.rate_limit_reset_at, account.overload_until, account.temp_unschedulable_until]
    .some((value) => value ? new Date(value).getTime() > now : false)) return false
  if (account.auto_pause_on_expired && account.expires_at && account.expires_at * 1000 <= now) return false
  const quotaExceeded = (used?: number | null, limit?: number | null) =>
    typeof limit === 'number' && limit > 0 && typeof used === 'number' && used >= limit
  const quotaWindowActive = (kind: 'daily' | 'weekly') => {
    const extra = account.extra as Record<string, unknown> | undefined
    const mode = String(extra?.[`quota_${kind}_reset_mode`] || 'rolling')
    const rawTime = mode === 'fixed'
      ? extra?.[`quota_${kind}_reset_at`]
      : extra?.[`quota_${kind}_start`]
    const parsedTime = rawTime ? new Date(String(rawTime)).getTime() : Number.NaN
    const resetTime = mode === 'fixed' || !Number.isFinite(parsedTime)
      ? parsedTime
      : parsedTime + (kind === 'daily' ? 86_400_000 : 604_800_000)
    return Number.isFinite(resetTime) && resetTime > now
  }
  return !(['apikey', 'bedrock'].includes(account.type) && (
    quotaExceeded(account.quota_used, account.quota_limit) ||
    (quotaWindowActive('daily') && quotaExceeded(account.quota_daily_used, account.quota_daily_limit)) ||
    (quotaWindowActive('weekly') && quotaExceeded(account.quota_weekly_used, account.quota_weekly_limit))
  ))
})
const settlementState = computed(() => settlementStatusPresentation(props.settlement?.status || 'draft'))
const accountLines = computed(() => (props.settlement?.account_lines || []).filter((line) => line.account_id === props.accountId))
const accountCosts = computed(() => (props.settlement?.account_costs || []).filter((cost) => cost.account_id === props.accountId))
const dataQualityNotices = computed(() => {
  const notices: string[] = []
  if (!props.entries.length) notices.push(t('admin.sharedPool.intake.pendingNotice'))
  const nonExactLines = accountLines.value.filter((line) => line.trace_quality !== 'exact').length
  if (nonExactLines) notices.push(t('admin.sharedPool.settlement.unpricedWarning', { count: nonExactLines }))
  const pending = props.approvals.filter((item) => item.status === 'pending')
  if (pending.length) notices.push(t('admin.sharedPool.approval.queueSummary', {
    pending: pending.length,
    highRisk: pending.filter((item) => item.changes?.business?.high_risk).length
  }))
  return notices
})
const formatMoney = (amount: number, currency = props.settlement?.currency || 'CNY') => formatPoolMoney(Number.isFinite(amount) ? amount : 0, currency, locale.value)
const formatPercent = (value: number) => `${(Number.isFinite(value) ? value : 0).toFixed(1)}%`
const formatDate = (value?: string | null) => value ? formatDateTimeToMinute(value, locale.value) : '-'
const dateOnly = (value?: string | null) => value ? value.slice(0, 10) : '-'
const lifecycleLabel = (event: SharedPoolLifecycleEvent['event_type']) => t(`admin.sharedPool.event.${event === 'banned_confirmed' ? 'banned' : event}`)
const approvalLabel = (action: PoolApprovalAction) => t(`admin.sharedPool.approval.${action === 'UPDATE_ACCOUNT' ? 'updateAccount' : action === 'VIEW_CREDENTIAL' ? 'viewCredential' : 'deleteAccount'}`)
const approvalState = (status: PoolApprovalStatus) => status === 'approved' || status === 'consumed' ? 'success' : status === 'pending' ? 'warning' : status === 'rejected' ? 'danger' : 'inactive'
</script>
