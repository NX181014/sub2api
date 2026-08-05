<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import type {
  SharedPoolSettlementAccountLine,
  SharedPoolSettlementLine
} from '@/api/admin/sharedPool'
import { formatPoolMoney } from '@/utils/sharedPool'
import { buildSettlementTransferPreview } from '@/utils/settlementTransfers'

const props = withDefaults(defineProps<{
  lines: SharedPoolSettlementLine[]
  accountLines?: SharedPoolSettlementAccountLine[]
  accountNames?: Record<number, string>
  currency?: string
}>(), {
  accountLines: () => [],
  accountNames: () => ({}),
  currency: 'CNY'
})

const { t } = useI18n()
const transfers = computed(() => buildSettlementTransferPreview(
  props.accountLines.length ? props.accountLines : props.lines
))
const groups = computed(() => {
  const result = new Map<number, typeof transfers.value>()
  for (const transfer of transfers.value) {
    const accountID = transfer.account_id || 0
    const group = result.get(accountID)
    if (group) group.push(transfer)
    else result.set(accountID, [transfer])
  }
  return [...result.entries()].map(([accountID, items]) => ({ accountID, items }))
})
const total = computed(() => transfers.value.reduce((sum, item) => sum + item.amount, 0))
const money = (value: number) => formatPoolMoney(value, props.currency)
const accountLabel = (accountID: number) => props.accountNames[accountID]
  || t('admin.sharedPool.settlement.accountFallback', { id: accountID })
</script>

<template>
  <section class="min-w-0 overflow-hidden border-y border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:px-5">
      <div class="min-w-0">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.settlement.transferPreview') }}</h2>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.transferSummary', { count: transfers.length, total: money(total) }) }}</p>
      </div>
      <Icon name="arrowRight" size="sm" class="text-gray-400" />
    </header>
    <div v-if="transfers.length" class="divide-y divide-gray-200 dark:divide-dark-700">
      <section v-for="group in groups" :key="group.accountID" class="min-w-0">
        <h3 v-if="group.accountID" class="truncate bg-gray-50 px-4 py-2 text-xs font-semibold text-gray-600 dark:bg-dark-700/40 dark:text-gray-300 sm:px-5" :title="accountLabel(group.accountID)">
          {{ accountLabel(group.accountID) }}
        </h3>
        <div class="divide-y divide-gray-100 dark:divide-dark-700">
          <article v-for="transfer in group.items" :key="`${transfer.from_user_id}-${transfer.to_user_id}`" class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-2 px-4 py-3 sm:flex-nowrap sm:px-5">
            <span class="min-w-24 flex-1 truncate text-sm font-medium text-gray-900 dark:text-white" :title="transfer.from_user_name">{{ transfer.from_user_name }}</span>
            <Icon name="arrowRight" size="xs" class="shrink-0 text-gray-400" />
            <span class="min-w-24 flex-1 truncate text-sm font-medium text-gray-900 dark:text-white" :title="transfer.to_user_name">{{ transfer.to_user_name }}</span>
            <span class="ml-auto shrink-0 text-sm font-semibold tabular-nums text-red-600 dark:text-red-400">{{ money(transfer.amount) }}</span>
          </article>
        </div>
      </section>
    </div>
    <EmptyState v-else :title="t('admin.sharedPool.settlement.noTransfers')" />
  </section>
</template>
