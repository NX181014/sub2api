<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import type { GroupPlatform } from '@/types'
import type { Column } from '@/components/common/types'
import {
  markSettlementTransferPaid,
  type SharedPoolAccountContext,
  type SharedPoolSettlementAccountLine,
  type SharedPoolSettlementLine,
  type SharedPoolSettlementTransfer
} from '@/api/admin/sharedPool'
import { formatPoolMoney } from '@/utils/sharedPool'
import { settlementLineState, type SettlementLineFilter } from '@/utils/sharedPoolLedger'
import {
  buildSettlementTransferPreview,
  type SettlementTransferPreview
} from '@/utils/settlementTransfers'

const props = withDefaults(defineProps<{
  settlementId?: number
  settlementUserId?: number | null
  status?: 'draft' | 'locked' | 'paid'
  lineStatus?: SettlementLineFilter
  lines: SharedPoolSettlementLine[]
  accountLines?: SharedPoolSettlementAccountLine[]
  accountNames?: Record<number, string>
  accountContexts?: SharedPoolAccountContext[]
  transfers?: SharedPoolSettlementTransfer[]
  calculatedAt?: string
  validAccountCount?: number
  currency?: string
}>(), {
  accountLines: () => [],
  accountNames: () => ({}),
  accountContexts: () => [],
  status: 'draft',
  lineStatus: 'all',
  currency: 'CNY'
})

const emit = defineEmits<{ settled: [] }>()
const { t, locale } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const activeTransfer = ref<SettlementTransferPreview | null>(null)
const settling = ref(false)

const sourceLines = computed(() => props.accountLines.length ? props.accountLines : props.lines)
const transfers = computed(() => buildSettlementTransferPreview(
	sourceLines.value,
	props.accountNames,
	Object.fromEntries((props.transfers || []).map(transfer => [`${transfer.from_user_id}:${transfer.to_user_id}:${Math.round(transfer.amount * 100)}`, transfer.payment_status === 'paid' ? 'paid' : 'pending']))
))
const serverTransfers = computed(() => (props.transfers || []).map((transfer): SettlementTransferPreview => ({
  id: transfer.id,
  member_user_id: transfer.from_user_id,
  member_user_name: transfer.from_user_name,
  from_user_id: transfer.from_user_id,
  from_user_name: transfer.from_user_name,
  to_user_id: transfer.to_user_id,
  to_user_name: transfer.to_user_name,
  amount: transfer.amount,
  payment_status: transfer.payment_status === 'paid' ? 'paid' : 'pending',
  allocation_ids: transfer.account_line_ids,
  account_ids: transfer.account_ids,
  account_names: transfer.account_ids.map(id => props.accountNames[id] || `#${id}`),
  allocations: transfer.account_line_ids.map(id => {
    const line = sourceLines.value.find(item => 'id' in item && item.id === id)
    return { id, account_id: line && 'account_id' in line ? line.account_id : undefined, account_name: line && 'account_id' in line ? props.accountNames[line.account_id] || `#${line.account_id}` : '-', net_amount: line?.net_amount || 0 }
  })
})))
const effectiveTransfers = computed(() => serverTransfers.value.length ? serverTransfers.value : transfers.value)
const visibleTransfers = computed(() => {
	if (props.lineStatus === 'all') return effectiveTransfers.value
	const visibleMembers = new Set(props.lines
		.filter(line => settlementLineState(line) === props.lineStatus)
		.map(line => line.user_id))
	return effectiveTransfers.value.filter(item => visibleMembers.has(item.from_user_id) || visibleMembers.has(item.to_user_id))
})
const pendingTransfers = computed(() => visibleTransfers.value.filter(item => item.payment_status !== 'paid'))
const allPendingTransfers = computed(() => effectiveTransfers.value.filter(item => item.payment_status !== 'paid'))
const pendingTotal = computed(() => pendingTransfers.value.reduce((sum, item) => sum + item.amount, 0))
const calculatedLabel = computed(() => props.calculatedAt ? new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(props.calculatedAt)) : '-')
const contextByAccountID = computed(() => new Map(props.accountContexts.map(item => [item.id, item])))

const columns = computed<Column[]>(() => [
  { key: 'transfer', label: t('admin.sharedPool.settlement.transferDirection') },
  { key: 'accounts', label: t('admin.sharedPool.columns.accounts') },
  { key: 'amount', label: t('admin.sharedPool.settlement.transferAmount') },
  { key: 'actions', label: t('admin.sharedPool.columns.actions'), class: 'text-right' }
])

const money = (value: number) => formatPoolMoney(value, props.currency)
const accountContext = (accountID?: number) => accountID ? contextByAccountID.value.get(accountID) : undefined
const accountPlatform = (accountID?: number) => accountContext(accountID)?.platform as GroupPlatform | undefined
const accountTitle = (transfer: SettlementTransferPreview) => transfer.account_names.join(', ') || '-'
const transferKey = (transfer: SettlementTransferPreview) => transfer.id || `${transfer.from_user_id}:${transfer.to_user_id}:${transfer.amount}`

const openTransfer = (transfer: SettlementTransferPreview) => {
  activeTransfer.value = transfer
}

const closeTransfer = () => {
  if (!settling.value) activeTransfer.value = null
}

const copyTransfer = async () => {
  if (!activeTransfer.value) return
  const transfer = activeTransfer.value
  await copyToClipboard([
    t('admin.sharedPool.settlement.copyTitle'),
    `${t('admin.sharedPool.settlement.payer')}: ${transfer.from_user_name}`,
    `${t('admin.sharedPool.settlement.payee')}: ${transfer.to_user_name}`,
    `${t('admin.sharedPool.settlement.transferAmount')}: ${money(transfer.amount)}`,
    `${t('admin.sharedPool.columns.accounts')}: ${accountTitle(transfer)}`
  ].join('\n'), t('admin.sharedPool.settlement.copied'))
}

const settleTransfer = async () => {
  const transfer = activeTransfer.value
  if (!transfer || !props.settlementId || transfer.payment_status === 'paid' || settling.value) return
  settling.value = true
  try {
    if (!transfer.id) return
    await markSettlementTransferPaid(props.settlementId, transfer.id)
    appStore.showSuccess(t('admin.sharedPool.settlement.memberSettled'))
    activeTransfer.value = null
    emit('settled')
  } catch (error: any) {
    appStore.showError(error?.response?.data?.message || error?.message || t('admin.sharedPool.errors.markPaid'))
  } finally {
    settling.value = false
  }
}
</script>

<template>
  <div class="min-w-0 space-y-4">
    <section class="card min-w-0 overflow-visible">
      <header class="card-header !px-4 !py-3 sm:!px-5">
        <div class="flex min-w-0 items-center gap-3">
          <Icon name="creditCard" size="md" class="shrink-0 text-primary-600 dark:text-primary-400" />
          <div class="min-w-0">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.settlement.transferWorkbench') }}</h2>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.hubHint') }}</p>
          </div>
        </div>
      </header>

      <div class="card-body !p-4 sm:!p-5">
        <div class="grid min-w-0 gap-4 sm:grid-cols-[minmax(0,1.3fr)_repeat(3,minmax(0,1fr))] sm:items-end">
          <div class="min-w-0">
            <p class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.autoNetting') }}</p>
            <p class="mt-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.sharedPool.settlement.autoNettingHint') }}</p>
          </div>
          <dl class="grid grid-cols-3 gap-3 text-sm">
            <div class="min-w-0">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.pendingMembers') }}</dt>
              <dd class="mt-1 font-semibold tabular-nums text-gray-900 dark:text-white">{{ pendingTransfers.length }}</dd>
            </div>
            <div class="min-w-0">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.accounts') }}</dt>
              <dd class="mt-1 font-semibold tabular-nums text-gray-900 dark:text-white">{{ validAccountCount ?? new Set(pendingTransfers.flatMap(item => item.account_ids)).size }}</dd>
            </div>
            <div class="min-w-0">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.pendingAmount') }}</dt>
              <dd class="mt-1 truncate font-semibold tabular-nums text-gray-900 dark:text-white" :title="money(pendingTotal)">{{ money(pendingTotal) }}</dd>
            </div>
          </dl>
        </div>
      </div>

      <footer class="card-footer flex flex-wrap items-center justify-between gap-2 !px-4 !py-3 text-xs text-gray-500 dark:text-gray-400 sm:!px-5">
        <span>{{ t('admin.sharedPool.settlement.singleTransferHint') }} · {{ calculatedLabel }}</span>
        <StatusBadge
          :status="allPendingTransfers.length ? 'warning' : 'success'"
          :label="allPendingTransfers.length ? t('admin.sharedPool.settlement.pendingSummary', { count: allPendingTransfers.length }) : t('admin.sharedPool.settlement.allSettled')"
        />
      </footer>
    </section>

    <section class="min-w-0">
      <div class="mb-3 flex flex-wrap items-end justify-between gap-2 px-1">
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.settlement.transferList') }}</h3>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.transferSummary', { count: visibleTransfers.length, total: money(visibleTransfers.reduce((sum, item) => sum + item.amount, 0)) }) }}</p>
        </div>
      </div>

      <DataTable
        :columns="columns"
        :data="visibleTransfers"
        :row-key="transferKey"
        :sticky-first-column="false"
        :sticky-actions-column="false"
        :expandable-actions="false"
        :mobile-column-keys="['transfer', 'accounts', 'amount']"
      >
        <template #cell-transfer="{ row }">
          <div class="flex min-w-0 items-center gap-2">
            <span class="max-w-28 truncate font-medium" :title="row.from_user_name">{{ row.from_user_name }}</span>
            <Icon name="arrowRight" size="xs" class="shrink-0 text-gray-400" />
            <span class="max-w-28 truncate font-medium" :title="row.to_user_name">{{ row.to_user_name }}</span>
          </div>
        </template>

        <template #cell-accounts="{ row }">
          <div class="flex min-w-0 items-center gap-2" :title="accountTitle(row)">
            <span class="flex shrink-0 -space-x-1">
              <span
                v-for="accountID in row.account_ids.slice(0, 3)"
                :key="accountID"
                class="inline-flex h-6 w-6 items-center justify-center rounded-full border-2 border-white bg-gray-100 text-gray-600 dark:border-dark-900 dark:bg-dark-700 dark:text-gray-300"
              >
                <PlatformIcon v-if="accountPlatform(accountID)" :platform="accountPlatform(accountID)" size="xs" />
                <Icon v-else name="key" size="xs" />
              </span>
            </span>
            <span class="max-w-36 truncate text-xs text-gray-600 dark:text-gray-300">{{ row.account_names[0] || '-' }}</span>
            <span v-if="row.account_ids.length > 1" class="shrink-0 text-xs text-gray-500 dark:text-gray-400">+{{ row.account_ids.length - 1 }}</span>
          </div>
        </template>

        <template #cell-amount="{ row }">
          <div class="flex items-center justify-end gap-3 md:justify-start">
            <span class="font-semibold tabular-nums text-gray-900 dark:text-white">{{ money(row.amount) }}</span>
            <StatusBadge
              :status="row.payment_status === 'paid' ? 'success' : 'warning'"
              :label="row.payment_status === 'paid' ? t('admin.sharedPool.settlement.settled') : t('admin.sharedPool.settlement.pendingTransfer')"
            />
          </div>
        </template>

        <template #cell-actions="{ row }">
          <div class="flex justify-end">
            <button type="button" class="btn btn-secondary min-h-11 px-3 text-xs" @click="openTransfer(row)">
              <Icon :name="row.payment_status === 'paid' ? 'eye' : 'creditCard'" size="sm" />
              {{ row.payment_status === 'paid' ? t('common.view') : t('admin.sharedPool.settlement.handleTransfer') }}
            </button>
          </div>
        </template>

        <template #empty>
          <div class="py-5 text-center">
            <Icon name="checkCircle" size="xl" class="mx-auto text-green-500" />
            <p class="mt-3 font-medium text-gray-900 dark:text-white">{{ t('admin.sharedPool.settlement.noTransfers') }}</p>
          </div>
        </template>
      </DataTable>
    </section>

    <BaseDialog
      :show="!!activeTransfer"
      :title="t('admin.sharedPool.settlement.handleTransfer')"
      width="wide"
      @close="closeTransfer"
    >
      <template v-if="activeTransfer">
        <div class="flex min-w-0 flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 items-center gap-3">
            <Icon name="creditCard" size="lg" class="shrink-0 text-primary-600 dark:text-primary-400" />
            <div class="flex min-w-0 items-center gap-2">
              <span class="truncate font-semibold text-gray-900 dark:text-white" :title="activeTransfer.from_user_name">{{ activeTransfer.from_user_name }}</span>
              <Icon name="arrowRight" size="sm" class="shrink-0 text-gray-400" />
              <span class="truncate font-semibold text-gray-900 dark:text-white" :title="activeTransfer.to_user_name">{{ activeTransfer.to_user_name }}</span>
            </div>
          </div>
          <span class="shrink-0 text-xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ money(activeTransfer.amount) }}</span>
        </div>

        <dl class="mt-5 grid grid-cols-2 gap-4 border-y border-gray-100 py-4 text-sm dark:border-dark-700 sm:grid-cols-4">
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.payer') }}</dt><dd class="mt-1 truncate font-medium" :title="activeTransfer.from_user_name">{{ activeTransfer.from_user_name }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.payee') }}</dt><dd class="mt-1 truncate font-medium" :title="activeTransfer.to_user_name">{{ activeTransfer.to_user_name }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.accounts') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ activeTransfer.account_ids.length }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.status') }}</dt><dd class="mt-1"><StatusBadge :status="activeTransfer.payment_status === 'paid' ? 'success' : 'warning'" :label="activeTransfer.payment_status === 'paid' ? t('admin.sharedPool.settlement.settled') : t('admin.sharedPool.settlement.pendingTransfer')" /></dd></div>
        </dl>

        <div class="mt-5">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.settlement.allocationDetails') }}</h4>
          <div class="mt-2 divide-y divide-gray-100 border-y border-gray-100 dark:divide-dark-700 dark:border-dark-700">
            <div v-for="allocation in activeTransfer.allocations" :key="allocation.id || `${allocation.account_id}-${allocation.net_amount}`" class="flex min-w-0 items-center gap-3 py-3">
              <span class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                <PlatformIcon v-if="accountPlatform(allocation.account_id)" :platform="accountPlatform(allocation.account_id)" size="sm" />
                <Icon v-else name="key" size="sm" />
              </span>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-medium text-gray-900 dark:text-white" :title="allocation.account_name">{{ allocation.account_name }}</p>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">#{{ allocation.id || '-' }}</p>
              </div>
              <span class="shrink-0 text-sm font-semibold tabular-nums" :class="allocation.net_amount > 0 ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'">{{ money(allocation.net_amount) }}</span>
            </div>
          </div>
        </div>
      </template>

      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" class="btn btn-secondary min-h-11" :disabled="settling" @click="closeTransfer">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-secondary min-h-11" :disabled="settling" @click="copyTransfer">
            <Icon name="copy" size="sm" />
            {{ t('admin.sharedPool.settlement.copyTransfer') }}
          </button>
          <button
            type="button"
            class="btn btn-primary min-h-11"
            :disabled="!settlementId || status === 'paid' || settling || activeTransfer?.payment_status === 'paid'"
            @click="settleTransfer"
          >
            <Icon :name="settling ? 'refresh' : 'checkCircle'" size="sm" :class="settling ? 'animate-spin' : ''" />
            {{ activeTransfer?.payment_status === 'paid' ? t('admin.sharedPool.settlement.settled') : t('admin.sharedPool.settlement.markTransferPaid') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </div>
</template>
