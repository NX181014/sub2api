<template>
  <Teleport to="body">
    <div v-if="show" class="fixed inset-0 z-50" @keydown="handleKeydown">
      <button
        type="button"
        class="absolute inset-0 h-full w-full bg-black/45"
        :aria-label="t('common.close')"
        @click="emit('close')"
      ></button>
      <aside
        ref="panelRef"
        class="absolute inset-y-0 right-0 flex w-full flex-col bg-white shadow-xl dark:bg-dark-900 md:max-w-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="account-trace-title"
        tabindex="-1"
      >
        <header class="shrink-0 border-b border-gray-200 px-4 py-3 dark:border-dark-700 sm:px-5">
          <div class="flex min-w-0 items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.actions.poolRecord') }}</p>
              <h2 id="account-trace-title" class="mt-0.5 truncate text-base font-semibold text-gray-900 dark:text-white">
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

          <div v-if="account" class="mt-3 grid grid-cols-2 gap-x-4 gap-y-3 text-sm sm:grid-cols-3">
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.platformType') }}</p>
              <PlatformTypeBadge class="mt-1" :platform="account.platform" :type="account.type" />
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploader') }}</p>
              <p class="mt-1 truncate font-medium" :title="uploaderName">{{ uploaderName }}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.page.importBatch') }}</p>
              <p class="mt-1 truncate font-medium">{{ importBatch }}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploadedAt') }}</p>
              <p class="mt-1 whitespace-nowrap font-medium">{{ formatDate(account.created_at) }}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.status') }}</p>
              <StatusBadge class="mt-1" :status="accountState.badge" :label="t(`admin.sharedPool.status.${accountState.key}`)" />
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.schedulable') }}</p>
              <StatusBadge
                class="mt-1"
                :status="account.schedulable ? 'success' : 'warning'"
                :label="t(account.schedulable ? 'admin.accounts.schedulableEnabled' : 'admin.accounts.schedulableDisabled')"
              />
            </div>
          </div>
        </header>

        <nav class="scrollbar-hide flex shrink-0 overflow-x-auto border-b border-gray-200 px-2 dark:border-dark-700 sm:px-4" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="-mb-px min-h-11 shrink-0 border-b-2 px-3 text-sm font-medium"
            :class="activeTab === tab.key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400'"
            role="tab"
            :aria-selected="activeTab === tab.key"
            @click="activeTab = tab.key"
          >
            {{ tab.label }}
          </button>
        </nav>

        <div class="min-h-0 flex-1 overflow-y-auto p-3 sm:p-5">
          <div v-if="loading" class="flex min-h-48 items-center justify-center"><LoadingSpinner /></div>

          <template v-else-if="activeTab === 'costs'">
            <DataTable
              :columns="costColumns"
              :data="entries"
              row-key="id"
              :mobile-column-keys="['entry_type', 'purchase_source', 'original_amount', 'paid_at']"
            >
              <template #cell-entry_type="{ row }">{{ t(`admin.sharedPool.entryTypes.${row.entry_type}`) }}</template>
              <template #cell-original_amount="{ row }"><span class="tabular-nums">{{ formatMoney(Number(row.original_amount), row.currency) }}</span></template>
              <template #cell-service_start="{ row }"><span class="whitespace-nowrap">{{ dateOnly(row.service_start) }} - {{ dateOnly(row.service_end) }}</span></template>
              <template #cell-paid_at="{ row }"><span class="whitespace-nowrap">{{ formatDate(row.paid_at) }}</span></template>
            </DataTable>
            <EmptyState v-if="!entries.length" :title="t('admin.sharedPool.ledger.emptyEntries')" />
          </template>

          <template v-else-if="activeTab === 'settlement'">
            <div v-if="settlement" class="mb-4 flex flex-wrap items-center justify-between gap-2 border-b border-gray-100 pb-3 text-sm dark:border-dark-700">
              <span>{{ dateOnly(settlement.period_start) }} - {{ dateOnly(settlement.period_end) }}</span>
              <StatusBadge :status="settlementState.badge" :label="t(`admin.sharedPool.status.${settlementState.key}`)" />
            </div>
            <DataTable
              :columns="settlementColumns"
              :data="accountLines"
              row-key="id"
              :mobile-column-keys="['user_name', 'account_usage_weight', 'allocated_cost', 'net_amount']"
            >
              <template #cell-account_usage_weight="{ row }"><span class="tabular-nums">{{ formatMoney(row.account_usage_weight) }}</span></template>
              <template #cell-usage_share="{ row }"><span class="tabular-nums">{{ formatPercent(row.usage_share) }}</span></template>
              <template #cell-allocated_cost="{ row }"><span class="tabular-nums">{{ formatMoney(row.allocated_cost) }}</span></template>
              <template #cell-contribution_credit="{ row }"><span class="tabular-nums">{{ formatMoney(row.contribution_credit) }}</span></template>
              <template #cell-net_amount="{ row }"><span class="font-medium tabular-nums">{{ formatMoney(row.net_amount) }}</span></template>
            </DataTable>
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
          </template>

          <template v-else-if="activeTab === 'payback'">
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

          <template v-else>
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
        </div>
      </aside>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
import type {
  SharedPoolAccountCost,
  SharedPoolLedgerEntry,
  SharedPoolLifecycleEvent,
  SharedPoolSettlementPreview
} from '@/api/admin/sharedPool'
import { DataTable, EmptyState, LoadingSpinner } from '@/components/common'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTimeToMinute } from '@/utils/format'
import { accountStatusPresentation, formatPoolMoney, settlementStatusPresentation } from '@/utils/sharedPool'

type TraceTab = 'costs' | 'settlement' | 'payback' | 'lifecycle'
const props = defineProps<{
  show: boolean
  loading: boolean
  accountId: number
  account: Account | null
  entries: SharedPoolLedgerEntry[]
  settlement: SharedPoolSettlementPreview | null
  recovery: SharedPoolAccountCost | null
  lifecycle: SharedPoolLifecycleEvent[]
}>()
const emit = defineEmits<{ (event: 'close'): void }>()
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
  { key: 'lifecycle' as const, label: t('admin.sharedPool.approval.impacts.lifecycle_events') }
])
const costColumns = computed<Column[]>(() => [
  { key: 'entry_type', label: t('admin.sharedPool.columns.costType') },
  { key: 'purchase_source', label: t('admin.sharedPool.columns.source') },
  { key: 'payer_email', label: t('admin.sharedPool.ledger.payer') },
  { key: 'original_amount', label: t('admin.sharedPool.ledger.originalAmount') },
  { key: 'service_start', label: t('admin.sharedPool.columns.servicePeriod') },
  { key: 'paid_at', label: t('admin.sharedPool.ledger.paidAt') }
])
const settlementColumns = computed<Column[]>(() => [
  { key: 'user_name', label: t('admin.sharedPool.columns.member') },
  { key: 'account_usage_weight', label: t('admin.sharedPool.columns.usageWeight') },
  { key: 'usage_share', label: t('admin.sharedPool.columns.share') },
  { key: 'allocated_cost', label: t('admin.sharedPool.columns.allocated') },
  { key: 'contribution_credit', label: t('admin.sharedPool.columns.credit') },
  { key: 'net_amount', label: t('admin.sharedPool.columns.net') }
])
const uploaderName = computed(() => props.account?.uploader_username || props.account?.uploader_email || '-')
const importBatch = computed(() => String(props.account?.extra?.import_batch_id || t('admin.sharedPool.page.singleImport')))
const accountState = computed(() => accountStatusPresentation(props.account?.status === 'error' ? 'warning' : props.account?.status || 'inactive'))
const settlementState = computed(() => settlementStatusPresentation(props.settlement?.status || 'draft'))
const accountLines = computed(() => (props.settlement?.account_lines || []).filter((line) => line.account_id === props.accountId))
const accountCosts = computed(() => (props.settlement?.account_costs || []).filter((cost) => cost.account_id === props.accountId))
const formatMoney = (amount: number, currency = props.settlement?.currency || 'CNY') => formatPoolMoney(Number.isFinite(amount) ? amount : 0, currency, locale.value)
const formatPercent = (value: number) => `${(Number.isFinite(value) ? value : 0).toFixed(1)}%`
const formatDate = (value?: string | null) => value ? formatDateTimeToMinute(value, locale.value) : '-'
const dateOnly = (value?: string | null) => value ? value.slice(0, 10) : '-'
const lifecycleLabel = (event: SharedPoolLifecycleEvent['event_type']) => t(`admin.sharedPool.event.${event === 'banned_confirmed' ? 'banned' : event}`)
</script>
