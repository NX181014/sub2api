<template>
  <AppLayout>
    <div class="min-w-0 space-y-5">
      <div class="card min-w-0">
        <div class="scrollbar-hide flex overflow-x-auto border-b border-gray-200 px-2 dark:border-dark-700 sm:px-4">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="-mb-px inline-flex min-h-11 shrink-0 items-center gap-1.5 border-b-2 px-3 text-sm font-medium transition-colors sm:px-4"
            :class="activeTab === tab.key
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400 dark:hover:border-dark-500 dark:hover:text-gray-200'"
            @click="switchTab(tab.key)"
          >
            <Icon :name="tab.icon" size="sm" />
            {{ tab.label }}
          </button>
        </div>

        <div v-if="activeTab !== 'accounts'" class="flex flex-col gap-3 p-4 lg:flex-row lg:items-center">
          <div class="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row sm:items-center">
            <label for="pool-period-type" class="shrink-0 text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.sharedPool.period.label') }}
            </label>
            <div class="w-full sm:w-36">
              <Select
                id="pool-period-type"
                v-model="periodType"
                :options="periodOptions"
                :aria-label="t('admin.sharedPool.period.label')"
                @change="handlePeriodTypeChange"
              />
            </div>
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="handleDateRangeChange"
            />
          </div>
          <div class="flex flex-wrap items-center justify-end gap-2">
            <label for="pool-fx-rate" class="text-xs font-medium text-gray-600 dark:text-gray-300">
              {{ t('admin.sharedPool.form.fxRate') }}
            </label>
            <input
              id="pool-fx-rate"
              v-model.number="fxRate"
              class="input h-10 w-24"
              type="number"
              min="0.0001"
              step="0.0001"
            />
            <button
              type="button"
              class="btn btn-secondary h-10 w-10 p-0"
              :disabled="savingFXRate"
              :title="t('admin.sharedPool.form.saveFxRate')"
              :aria-label="t('admin.sharedPool.form.saveFxRate')"
              @click="updateFXRate"
            >
              <LoadingSpinner v-if="savingFXRate" size="sm" />
              <Icon v-else name="check" size="sm" />
            </button>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.sharedPool.period.timezone') }}
            </span>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="loading" @click="loadActiveTab">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="loading && !hasActiveData" class="flex min-h-64 items-center justify-center">
        <LoadingSpinner />
      </div>

      <template v-else-if="activeTab === 'overview'">
        <template v-if="overview">
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard
              :title="t('admin.sharedPool.metrics.purchaseCost')"
              :value="formatMoney(overview.summary.total_purchase_cost)"
              :icon="CostIcon"
              icon-variant="primary"
            />
            <StatCard
              :title="t('admin.sharedPool.metrics.usageValue')"
              :value="formatMoney(overview.summary.total_usage_value)"
              :icon="ValueIcon"
              icon-variant="success"
            />
            <StatCard
              :title="t('admin.sharedPool.metrics.roiRate')"
              :value="formatPercent(overview.summary.roi_rate)"
              :icon="TrendIcon"
              :icon-variant="overview.summary.roi_rate >= 100 ? 'success' : 'warning'"
            />
            <StatCard
              :title="t('admin.sharedPool.metrics.bannedLoss')"
              :value="formatMoney(overview.summary.banned_loss)"
              :icon="BanIcon"
              icon-variant="danger"
            />
          </div>

          <div class="grid grid-cols-1 gap-5 xl:grid-cols-3">
            <section class="card p-5 xl:col-span-1" aria-labelledby="pool-recovery-title">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <h2 id="pool-recovery-title" class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.sharedPool.overview.recoveryTitle') }}
                  </h2>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.sharedPool.overview.recoverySubtitle') }}
                  </p>
                </div>
                <StatusBadge
                  :status="overview.summary.roi_rate >= 100 ? 'success' : 'warning'"
                  :label="overview.summary.roi_rate >= 100
                    ? t('admin.sharedPool.status.recovered')
                    : t('admin.sharedPool.status.recovering')"
                />
              </div>
              <div class="mt-6">
                <div class="mb-2 flex items-end justify-between gap-3">
                  <span class="text-3xl font-bold tabular-nums text-gray-900 dark:text-white">
                    {{ formatPercent(overview.summary.roi_rate) }}
                  </span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    {{ overview.summary.recovered_accounts }} / {{ overview.summary.total_accounts }}
                    {{ t('admin.sharedPool.overview.accountsRecovered') }}
                  </span>
                </div>
                <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div
                    class="h-full rounded-full bg-green-500 transition-[width] duration-300"
                    :style="{ width: `${Math.min(Math.max(overview.summary.roi_rate, 0), 100)}%` }"
                  ></div>
                </div>
              </div>
              <dl class="mt-6 grid grid-cols-2 gap-4 border-t border-gray-100 pt-4 text-sm dark:border-dark-700">
                <div>
                  <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.pendingRecovery') }}</dt>
                  <dd class="mt-1 font-semibold tabular-nums text-amber-600 dark:text-amber-400">
                    {{ formatMoney(overview.summary.pending_recovery) }}
                  </dd>
                </div>
                <div>
                  <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.activeAccounts') }}</dt>
                  <dd class="mt-1 font-semibold tabular-nums text-gray-900 dark:text-white">
                    {{ overview.summary.active_accounts }}
                  </dd>
                </div>
              </dl>
            </section>

            <section class="card overflow-hidden xl:col-span-2" aria-labelledby="pool-account-recovery-title">
              <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                <h2 id="pool-account-recovery-title" class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.sharedPool.overview.accountRecovery') }}
                </h2>
              </div>
              <DataTable :columns="overviewColumns" :data="overview.accounts" row-key="id" :loading="loading">
                <template #cell-account_name="{ row }">
                  <div class="min-w-0">
                    <p class="max-w-52 truncate font-medium text-gray-900 dark:text-white" :title="row.account_name">
                      {{ row.account_name }}
                    </p>
                    <p class="max-w-52 truncate text-xs text-gray-500 dark:text-gray-400" :title="row.provider_identity">
                      {{ row.provider_identity || '-' }}
                    </p>
                  </div>
                </template>
                <template #cell-status="{ row }">
                  <StatusBadge
                    :status="accountStatus(row.status).badge"
                    :label="t(`admin.sharedPool.status.${accountStatus(row.status).key}`)"
                  />
                </template>
                <template #cell-roi_rate="{ row }">
                  <span class="font-medium tabular-nums" :class="row.roi_rate >= 100 ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'">
                    {{ formatPercent(row.roi_rate) }}
                  </span>
                </template>
                <template #cell-remaining_cost="{ row }">
                  <span class="tabular-nums">{{ formatMoney(row.remaining_cost, row.currency) }}</span>
                </template>
                <template #cell-recovered_at="{ row }">
                  <span v-if="row.recovered_at" class="whitespace-nowrap text-green-600 dark:text-green-400">
                    {{ formatDateTimeToMinute(row.recovered_at, locale) }}
                  </span>
                  <span v-else-if="row.estimated_recovery_days" class="whitespace-nowrap text-gray-500 dark:text-gray-400">
                    {{ t('admin.sharedPool.overview.estimatedDays', { days: row.estimated_recovery_days }) }}
                  </span>
                  <span v-else>-</span>
                </template>
                <template #cell-net_profit="{ row }">
                  <span class="whitespace-nowrap tabular-nums" :class="(row.net_profit || 0) >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                    {{ formatMoney(row.net_profit || 0, row.currency) }}
                  </span>
                </template>
              </DataTable>
            </section>
          </div>
        </template>
        <EmptyState v-else :title="t('admin.sharedPool.empty.overview')" />
      </template>

      <template v-else-if="activeTab === 'accounts'">
        <AccountsView embedded @pool-record="openAccountPoolRecord" />
        <section class="card overflow-hidden">
          <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.accounts.title') }}</h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.accounts.subtitle') }}</p>
            </div>
            <button type="button" class="btn btn-primary min-h-11" @click="openCostDialog">
              <Icon name="plus" size="sm" />
              {{ t('admin.sharedPool.accounts.addCost') }}
            </button>
          </div>
          <DataTable :columns="costColumns" :data="accountCosts" row-key="id" :loading="loading">
            <template #cell-account_name="{ row }">
              <div class="min-w-0">
                <p class="max-w-56 truncate font-medium text-gray-900 dark:text-white" :title="row.account_name">{{ row.account_name }}</p>
                <p class="max-w-56 truncate text-xs text-gray-500 dark:text-gray-400" :title="row.provider_identity">{{ row.provider_identity || '-' }}</p>
              </div>
            </template>
            <template #cell-purchase_source_name="{ row }">
              <div>
                <p class="font-medium text-gray-800 dark:text-gray-200">{{ row.purchase_source_name || '-' }}</p>
                <p v-if="row.order_no" class="text-xs text-gray-500 dark:text-gray-400">{{ row.order_no }}</p>
              </div>
            </template>
            <template #cell-purchase_cost="{ row }">
              <span class="font-medium tabular-nums">{{ formatMoney(row.purchase_cost, row.currency) }}</span>
            </template>
            <template #cell-entry_type="{ row }">
              <span class="whitespace-nowrap">{{ t(`admin.sharedPool.entryTypes.${row.entry_type || 'purchase'}`) }}</span>
            </template>
            <template #cell-service_start="{ row }">
              <span class="whitespace-nowrap tabular-nums">{{ row.service_start }} - {{ row.service_end }}</span>
            </template>
            <template #cell-status="{ row }">
              <StatusBadge
                :status="accountStatus(row.status).badge"
                :label="t(`admin.sharedPool.status.${accountStatus(row.status).key}`)"
              />
            </template>
            <template #cell-actions="{ row }">
              <button
                type="button"
                class="btn btn-secondary min-h-9 whitespace-nowrap px-2 text-xs"
                @click="openEventDialog(row)"
              >
                <Icon name="edit" size="xs" />
                {{ t('admin.sharedPool.accounts.recordEvent') }}
              </button>
            </template>
          </DataTable>
        </section>
      </template>

      <template v-else-if="activeTab === 'settlement'">
        <template v-if="settlement">
          <div
            v-if="settlement.unpriced_usage_count > 0"
            class="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-300"
            role="alert"
          >
            <Icon name="exclamationTriangle" size="md" class="mt-0.5 shrink-0" />
            <span>{{ t('admin.sharedPool.settlement.unpricedWarning', { count: settlement.unpriced_usage_count }) }}</span>
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard :title="t('admin.sharedPool.settlement.totalCost')" :value="formatMoney(settlement.total_cost, settlement.currency)" :icon="CostIcon" />
            <StatCard :title="t('admin.sharedPool.settlement.usageWeight')" :value="formatMoney(settlement.total_usage_weight, settlement.currency)" :icon="ValueIcon" icon-variant="success" />
            <StatCard :title="t('admin.sharedPool.settlement.coverage')" :value="formatPercent(settlement.pricing_coverage)" :icon="TrendIcon" :icon-variant="settlement.pricing_coverage >= 99 ? 'success' : 'warning'" />
            <StatCard :title="t('admin.sharedPool.settlement.carryForward')" :value="formatMoney(settlement.carry_forward, settlement.currency)" :icon="CarryIcon" icon-variant="warning" />
          </div>

          <section class="card overflow-hidden">
            <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-center gap-3">
                <div>
                  <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.settlement.title') }}</h2>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.formula') }}</p>
                </div>
                <StatusBadge
                  :status="settlementStatus(settlement.status).badge"
                  :label="t(`admin.sharedPool.status.${settlementStatus(settlement.status).key}`)"
                />
              </div>
              <button
                type="button"
                class="btn btn-primary min-h-11"
                :disabled="settlement.status !== 'draft' || settlement.unpriced_usage_count > 0 || locking"
                @click="showLockConfirm = true"
              >
                <Icon name="lock" size="sm" />
                {{ t('admin.sharedPool.settlement.lock') }}
              </button>
            </div>
            <DataTable :columns="settlementColumns" :data="settlement.lines" row-key="user_id" :loading="loading">
              <template #cell-usage_weight="{ row }"><span class="tabular-nums">{{ formatMoney(row.usage_weight, settlement.currency) }}</span></template>
              <template #cell-usage_share="{ row }"><span class="tabular-nums">{{ formatPercent(row.usage_share) }}</span></template>
              <template #cell-allocated_cost="{ row }"><span class="tabular-nums">{{ formatMoney(row.allocated_cost, settlement.currency) }}</span></template>
              <template #cell-contribution_credit="{ row }"><span class="tabular-nums text-green-600 dark:text-green-400">-{{ formatMoney(row.contribution_credit, settlement.currency) }}</span></template>
              <template #cell-net_amount="{ row }">
                <span class="font-semibold tabular-nums" :class="row.net_amount > 0 ? 'text-red-600 dark:text-red-400' : row.net_amount < 0 ? 'text-green-600 dark:text-green-400' : 'text-gray-600 dark:text-gray-300'">
                  {{ row.net_amount > 0 ? t('admin.sharedPool.settlement.payable') : row.net_amount < 0 ? t('admin.sharedPool.settlement.receivable') : '' }}
                  {{ formatMoney(Math.abs(row.net_amount), settlement.currency) }}
                </span>
              </template>
            </DataTable>
          </section>
        </template>
        <EmptyState v-else :title="t('admin.sharedPool.empty.settlement')" />
      </template>

      <template v-else>
        <div v-if="sources.length" class="grid grid-cols-1 gap-5 xl:grid-cols-5">
          <section class="card p-4 xl:col-span-2" aria-labelledby="pool-source-chart-title">
            <h2 id="pool-source-chart-title" class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.sharedPool.sources.chartTitle') }}
            </h2>
            <div class="h-72">
              <Bar :data="sourceChartData" :options="sourceChartOptions" />
            </div>
          </section>
          <section class="card overflow-hidden xl:col-span-3">
            <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.sources.title') }}</h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.sources.sampleHint') }}</p>
            </div>
            <DataTable :columns="sourceColumns" :data="sources" row-key="name" :loading="loading">
              <template #cell-name="{ row }">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                  <span v-if="row.sample_size < 5" class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400">
                    {{ t('admin.sharedPool.sources.smallSample') }}
                  </span>
                </div>
              </template>
              <template #cell-roi_rate="{ row }"><span class="font-medium tabular-nums">{{ formatPercent(row.roi_rate) }}</span></template>
              <template #cell-ban_rate_30d="{ row }"><span class="tabular-nums text-red-600 dark:text-red-400">{{ formatPercent(row.ban_rate_30d) }}</span></template>
              <template #cell-purchase_cost="{ row }"><span class="tabular-nums">{{ formatMoney(row.purchase_cost) }}</span></template>
            </DataTable>
          </section>
        </div>
        <EmptyState v-else :title="t('admin.sharedPool.empty.sources')" />
      </template>
    </div>

    <BaseDialog :show="showCostDialog" :title="intakeMode ? t('admin.sharedPool.intake.title') : t('admin.sharedPool.form.title')" width="wide" @close="closeCostDialog">
      <form id="pool-cost-form" class="grid grid-cols-1 gap-4 md:grid-cols-2" @submit.prevent="saveAccountCost">
        <div class="md:col-span-2">
          <label for="pool-account" class="input-label">{{ t('admin.sharedPool.form.account') }} *</label>
          <Select id="pool-account" v-model="costForm.account_id" :options="accountOptions" searchable :disabled="intakeMode" :aria-label="t('admin.sharedPool.form.account')" />
        </div>
        <div>
          <label for="pool-provider-identity" class="input-label">{{ t('admin.sharedPool.form.providerIdentity') }} *</label>
          <input id="pool-provider-identity" v-model.trim="costForm.provider_identity" class="input" required />
        </div>
        <div>
          <label for="pool-source" class="input-label">{{ t('admin.sharedPool.form.source') }} *</label>
          <input id="pool-source" v-model.trim="costForm.purchase_source_name" class="input" required list="pool-source-list" />
          <datalist id="pool-source-list">
            <option v-for="source in sources" :key="source.name" :value="source.name"></option>
          </datalist>
        </div>
        <div>
          <label for="pool-contributor" class="input-label">{{ t('admin.sharedPool.form.contributor') }} *</label>
          <Select id="pool-contributor" v-model="costForm.contributor_user_id" :options="userOptions" searchable :aria-label="t('admin.sharedPool.form.contributor')" />
        </div>
        <div>
          <label for="pool-uploader" class="input-label">{{ t('admin.sharedPool.form.uploader') }} *</label>
          <Select id="pool-uploader" v-model="costForm.uploader_user_id" :options="userOptions" searchable :aria-label="t('admin.sharedPool.form.uploader')" />
        </div>
        <div>
          <label for="pool-entry-type" class="input-label">{{ t('admin.sharedPool.form.entryType') }} *</label>
          <Select id="pool-entry-type" v-model="costForm.entry_type" :options="costEntryTypeOptions" :aria-label="t('admin.sharedPool.form.entryType')" />
        </div>
        <div>
          <label for="pool-cost" class="input-label">{{ t('admin.sharedPool.form.cost') }} *</label>
          <input id="pool-cost" v-model.number="costForm.purchase_cost" class="input" type="number" min="0.01" step="0.01" required />
        </div>
        <div>
          <label for="pool-currency" class="input-label">{{ t('admin.sharedPool.form.currency') }}</label>
          <Select id="pool-currency" v-model="costForm.currency" :options="currencyOptions" :aria-label="t('admin.sharedPool.form.currency')" />
        </div>
        <div>
          <label for="pool-service-start" class="input-label">{{ t('admin.sharedPool.form.serviceStart') }} *</label>
          <input id="pool-service-start" v-model="costForm.service_start" class="input" type="date" required />
        </div>
        <div>
          <label for="pool-service-end" class="input-label">{{ t('admin.sharedPool.form.serviceEnd') }} *</label>
          <input id="pool-service-end" v-model="costForm.service_end" class="input" type="date" :min="costForm.service_start" required />
        </div>
        <div>
          <label for="pool-warranty-end" class="input-label">{{ t('admin.sharedPool.form.warrantyEnd') }}</label>
          <input id="pool-warranty-end" v-model="costForm.warranty_end" class="input" type="date" />
        </div>
        <div>
          <label for="pool-order" class="input-label">{{ t('admin.sharedPool.form.orderNo') }}</label>
          <input id="pool-order" v-model.trim="costForm.order_no" class="input" />
        </div>
        <div class="md:col-span-2">
          <label for="pool-purchase-url" class="input-label">{{ t('admin.sharedPool.form.purchaseUrl') }}</label>
          <input id="pool-purchase-url" v-model.trim="costForm.purchase_url" class="input" type="url" />
        </div>
        <div class="md:col-span-2">
          <label for="pool-notes" class="input-label">{{ t('admin.sharedPool.form.notes') }}</label>
          <textarea id="pool-notes" v-model.trim="costForm.notes" class="input min-h-20" rows="3"></textarea>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeCostDialog">{{ t('common.cancel') }}</button>
          <button type="submit" form="pool-cost-form" class="btn btn-primary" :disabled="savingCost">
            <LoadingSpinner v-if="savingCost" size="sm" />
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="showEventDialog" :title="t('admin.sharedPool.event.title')" @close="closeEventDialog">
      <form id="pool-event-form" class="space-y-4" @submit.prevent="saveLifecycleEvent">
        <div>
          <label for="pool-event-account" class="input-label">{{ t('admin.sharedPool.form.account') }}</label>
          <input id="pool-event-account" class="input" :value="eventAccount?.account_name || ''" disabled />
        </div>
        <div>
          <label for="pool-event-type" class="input-label">{{ t('admin.sharedPool.event.type') }} *</label>
          <Select id="pool-event-type" v-model="eventForm.event_type" :options="eventOptions" />
        </div>
        <div>
          <label for="pool-event-date" class="input-label">{{ t('admin.sharedPool.event.date') }} *</label>
          <input id="pool-event-date" v-model="eventForm.date" class="input" type="date" required />
        </div>
        <template v-if="eventForm.event_type === 'refund'">
          <div>
            <label for="pool-refund-amount" class="input-label">{{ t('admin.sharedPool.event.refundAmount') }}</label>
            <input id="pool-refund-amount" v-model.number="eventForm.amount" class="input" type="number" min="0" step="0.01" />
          </div>
        </template>
        <template v-if="eventForm.event_type === 'replaced'">
          <div>
            <label for="pool-replacement-account" class="input-label">{{ t('admin.sharedPool.event.replacementAccount') }} *</label>
            <Select id="pool-replacement-account" v-model="eventForm.replacement_account_id" :options="replacementAccountOptions" searchable />
          </div>
          <div>
            <label for="pool-transfer-amount" class="input-label">{{ t('admin.sharedPool.event.transferAmount') }}</label>
            <input id="pool-transfer-amount" v-model.number="eventForm.amount" class="input" type="number" min="0" step="0.01" />
          </div>
        </template>
        <div>
          <label for="pool-event-reason" class="input-label">{{ t('admin.sharedPool.event.reason') }}</label>
          <textarea id="pool-event-reason" v-model.trim="eventForm.reason" class="input min-h-20" rows="3"></textarea>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeEventDialog">{{ t('common.cancel') }}</button>
          <button type="submit" form="pool-event-form" class="btn btn-primary" :disabled="savingEvent">
            <LoadingSpinner v-if="savingEvent" size="sm" />
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showLockConfirm"
      :title="t('admin.sharedPool.settlement.lockTitle')"
      :message="t('admin.sharedPool.settlement.lockMessage')"
      :confirm-text="t('admin.sharedPool.settlement.lock')"
      @confirm="confirmLockSettlement"
      @cancel="showLockConfirm = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bar } from 'vue-chartjs'
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Tooltip,
  type ChartData,
  type ChartOptions
} from 'chart.js'
import AppLayout from '@/components/layout/AppLayout.vue'
import AccountsView from '@/views/admin/AccountsView.vue'
import {
  BaseDialog,
  ConfirmDialog,
  DataTable,
  EmptyState,
  LoadingSpinner,
  StatCard
} from '@/components/common'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type {
  CreateSharedPoolCostRequest,
  PoolLifecycleEventType,
  PoolAccountStatus,
  PoolPeriodType,
  PoolSettlementStatus,
  SharedPoolAccountCost,
  SharedPoolOverview,
  SharedPoolSettlementPreview,
  SharedPoolSourceStat
} from '@/api/admin/sharedPool'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import type { Account } from '@/types'
import { formatDateTimeToMinute } from '@/utils/format'
import {
  accountStatusPresentation,
  buildPoolPeriodParams,
  formatPoolMoney,
  resolvePoolPeriod,
  settlementStatusPresentation
} from '@/utils/sharedPool'

ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend)

type TabKey = 'overview' | 'accounts' | 'settlement' | 'sources'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const initialPeriod = resolvePoolPeriod('month')
const activeTab = ref<TabKey>('accounts')
const periodType = ref<PoolPeriodType>('month')
const startDate = ref(initialPeriod.start)
const endDate = ref(initialPeriod.end)
const loading = ref(false)
const savingCost = ref(false)
const savingEvent = ref(false)
const savingFXRate = ref(false)
const locking = ref(false)
const showCostDialog = ref(false)
const intakeMode = ref(false)
const showEventDialog = ref(false)
const showLockConfirm = ref(false)
const overview = ref<SharedPoolOverview | null>(null)
const accountCosts = ref<SharedPoolAccountCost[]>([])
const settlement = ref<SharedPoolSettlementPreview | null>(null)
const sources = ref<SharedPoolSourceStat[]>([])
const accountOptions = ref<Array<{ value: number; label: string }>>([])
const userOptions = ref<Array<{ value: number; label: string }>>([])
const fxRate = ref(1)
const eventAccount = ref<SharedPoolAccountCost | null>(null)

const tabs = computed(() => [
  { key: 'overview' as const, label: t('admin.sharedPool.tabs.overview'), icon: 'chart' as const },
  { key: 'accounts' as const, label: t('admin.sharedPool.tabs.accounts'), icon: 'server' as const },
  { key: 'settlement' as const, label: t('admin.sharedPool.tabs.settlement'), icon: 'calculator' as const },
  { key: 'sources' as const, label: t('admin.sharedPool.tabs.sources'), icon: 'link' as const }
])

const periodOptions = computed(() => [
  { value: 'day', label: t('admin.sharedPool.period.day') },
  { value: 'week', label: t('admin.sharedPool.period.week') },
  { value: 'month', label: t('admin.sharedPool.period.month') },
  { value: 'custom', label: t('admin.sharedPool.period.custom') }
])

const eventOptions = computed(() => [
  { value: 'banned_confirmed', label: t('admin.sharedPool.event.banned') },
  { value: 'recovered', label: t('admin.sharedPool.event.recovered') },
  { value: 'refund', label: t('admin.sharedPool.event.refund') },
  { value: 'replaced', label: t('admin.sharedPool.event.replaced') },
  { value: 'retired', label: t('admin.sharedPool.event.retired') }
])

const costEntryTypeOptions = computed(() => [
  { value: 'purchase', label: t('admin.sharedPool.entryTypes.purchase') },
  { value: 'renewal', label: t('admin.sharedPool.entryTypes.renewal') },
  { value: 'topup', label: t('admin.sharedPool.entryTypes.topup') },
  { value: 'price_version', label: t('admin.sharedPool.entryTypes.price_version') },
  { value: 'adjustment', label: t('admin.sharedPool.entryTypes.adjustment') }
])

const replacementAccountOptions = computed(() =>
  accountOptions.value.filter((account) => account.value !== eventAccount.value?.account_id)
)

const currencyOptions = [
  { value: 'CNY', label: 'CNY' }
]

const CostIcon = { render: () => h(Icon, { name: 'dollar', size: 'lg' }) }
const ValueIcon = { render: () => h(Icon, { name: 'chartBar', size: 'lg' }) }
const TrendIcon = { render: () => h(Icon, { name: 'trendingUp', size: 'lg' }) }
const BanIcon = { render: () => h(Icon, { name: 'ban', size: 'lg' }) }
const CarryIcon = { render: () => h(Icon, { name: 'swap', size: 'lg' }) }

const overviewColumns = computed<Column[]>(() => [
  { key: 'account_name', label: t('admin.sharedPool.columns.account'), sortable: true },
  { key: 'purchase_source_name', label: t('admin.sharedPool.columns.source'), sortable: true },
  { key: 'status', label: t('admin.sharedPool.columns.status'), sortable: true },
  { key: 'roi_rate', label: t('admin.sharedPool.columns.roi'), sortable: true },
  { key: 'remaining_cost', label: t('admin.sharedPool.columns.remaining'), sortable: true },
  { key: 'net_profit', label: t('admin.sharedPool.columns.netProfit'), sortable: true },
  { key: 'recovered_at', label: t('admin.sharedPool.columns.recoveredAt'), sortable: true }
])

const costColumns = computed<Column[]>(() => [
  { key: 'account_name', label: t('admin.sharedPool.columns.account'), sortable: true },
  { key: 'contributor_name', label: t('admin.sharedPool.columns.contributor'), sortable: true },
  { key: 'uploader_name', label: t('admin.sharedPool.columns.uploader'), sortable: true },
  { key: 'purchase_source_name', label: t('admin.sharedPool.columns.source'), sortable: true },
  { key: 'entry_type', label: t('admin.sharedPool.columns.costType'), sortable: true },
  { key: 'purchase_cost', label: t('admin.sharedPool.columns.cost'), sortable: true },
  { key: 'service_start', label: t('admin.sharedPool.columns.servicePeriod'), sortable: true },
  { key: 'warranty_end', label: t('admin.sharedPool.columns.warranty'), sortable: true },
  { key: 'status', label: t('admin.sharedPool.columns.status'), sortable: true },
  { key: 'actions', label: t('admin.sharedPool.columns.actions') }
])

const settlementColumns = computed<Column[]>(() => [
  { key: 'user_name', label: t('admin.sharedPool.columns.member'), sortable: true },
  { key: 'usage_weight', label: t('admin.sharedPool.columns.usageWeight'), sortable: true },
  { key: 'usage_share', label: t('admin.sharedPool.columns.share'), sortable: true },
  { key: 'allocated_cost', label: t('admin.sharedPool.columns.allocated'), sortable: true },
  { key: 'contribution_credit', label: t('admin.sharedPool.columns.credit'), sortable: true },
  { key: 'net_amount', label: t('admin.sharedPool.columns.net'), sortable: true }
])

const sourceColumns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.sharedPool.columns.source'), sortable: true },
  { key: 'account_count', label: t('admin.sharedPool.columns.accounts'), sortable: true },
  { key: 'purchase_cost', label: t('admin.sharedPool.columns.cost'), sortable: true },
  { key: 'roi_rate', label: t('admin.sharedPool.columns.roi'), sortable: true },
  { key: 'ban_rate_30d', label: t('admin.sharedPool.columns.ban30'), sortable: true },
  { key: 'average_survival_days', label: t('admin.sharedPool.columns.survival'), sortable: true }
])

const hasActiveData = computed(() => {
  if (activeTab.value === 'overview') return !!overview.value
  if (activeTab.value === 'accounts') return true
  if (activeTab.value === 'settlement') return !!settlement.value
  return sources.value.length > 0
})

const sourceChartData = computed<ChartData<'bar'>>(() => ({
  labels: sources.value.map((source) => source.name),
  datasets: [
    {
      label: t('admin.sharedPool.columns.roi'),
      data: sources.value.map((source) => source.roi_rate),
      backgroundColor: 'rgba(34, 197, 94, 0.72)',
      borderRadius: 4
    },
    {
      label: t('admin.sharedPool.columns.ban30'),
      data: sources.value.map((source) => source.ban_rate_30d),
      backgroundColor: 'rgba(239, 68, 68, 0.68)',
      borderRadius: 4
    }
  ]
}))

const sourceChartOptions = computed<ChartOptions<'bar'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index', intersect: false },
  plugins: {
    legend: { position: 'bottom', labels: { boxWidth: 12 } },
    tooltip: { callbacks: { label: (context) => `${context.dataset.label}: ${Number(context.raw).toFixed(1)}%` } }
  },
  scales: {
    x: { grid: { display: false } },
    y: { beginAtZero: true, ticks: { callback: (value) => `${value}%` } }
  }
}))

const emptyCostForm = (): CreateSharedPoolCostRequest => ({
  account_id: 0,
  provider_identity: '',
  contributor_user_id: 0,
  uploader_user_id: 0,
  purchase_source_name: '',
  purchase_url: '',
  order_no: '',
  purchase_cost: 0,
  entry_type: 'purchase',
  currency: 'CNY',
  service_start: startDate.value,
  service_end: endDate.value,
  warranty_end: '',
  notes: ''
})

const costForm = reactive<CreateSharedPoolCostRequest>(emptyCostForm())
const shanghaiToday = () => new Date(Date.now() + 8 * 60 * 60 * 1000).toISOString().slice(0, 10)
const emptyEventForm = () => ({
  event_type: 'banned_confirmed' as PoolLifecycleEventType,
  date: shanghaiToday(),
  amount: 0,
  replacement_account_id: 0,
  reason: ''
})
const eventForm = reactive(emptyEventForm())
const periodParams = () => buildPoolPeriodParams(periodType.value, startDate.value, endDate.value)

const formatMoney = (amount: number, currency = overview.value?.currency || 'CNY') =>
  formatPoolMoney(amount, currency, locale.value)
const formatPercent = (value: number) => `${(Number.isFinite(value) ? value : 0).toFixed(1)}%`
const accountStatus = (status: PoolAccountStatus) => accountStatusPresentation(status)
const settlementStatus = (status: PoolSettlementStatus) => settlementStatusPresentation(status)

function handlePeriodTypeChange(value: string | number | boolean | null) {
  if (value !== 'custom') {
    const range = resolvePoolPeriod(value as PoolPeriodType)
    startDate.value = range.start
    endDate.value = range.end
  }
  void loadActiveTab()
}

function handleDateRangeChange(range: { startDate: string; endDate: string }) {
  startDate.value = range.startDate
  endDate.value = range.endDate
  periodType.value = 'custom'
  void loadActiveTab()
}

function switchTab(tab: TabKey) {
  activeTab.value = tab
  void loadActiveTab()
}

async function loadActiveTab() {
  loading.value = true
  try {
    if (activeTab.value === 'overview') {
      overview.value = await adminAPI.sharedPool.getOverview(periodParams())
    } else if (activeTab.value === 'accounts') {
      const response = await adminAPI.sharedPool.listAccountCosts(periodParams())
      accountCosts.value = response.items || []
    } else if (activeTab.value === 'settlement') {
      settlement.value = await adminAPI.sharedPool.previewSettlement(periodParams())
    } else {
      const response = await adminAPI.sharedPool.listSources(periodParams())
      sources.value = response.items || []
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.load'))
  } finally {
    loading.value = false
  }
}

async function loadReferenceOptions() {
  const [accounts, users, sourceResponse] = await Promise.all([
    adminAPI.accounts.list(1, 200, { sort_by: 'name', sort_order: 'asc' }),
    adminAPI.users.list(1, 200, { status: 'active', sort_by: 'email', sort_order: 'asc' }),
    adminAPI.sharedPool.listSources(periodParams())
  ])
  accountOptions.value = accounts.items.map((account) => ({ value: account.id, label: account.name }))
  userOptions.value = users.items.map((user) => ({ value: user.id, label: user.username || user.email }))
  sources.value = sourceResponse.items || []
}

async function openCostDialog() {
  intakeMode.value = false
  Object.assign(costForm, emptyCostForm())
  showCostDialog.value = true
  try {
    await loadReferenceOptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.options'))
  }
}

async function openAccountPoolRecord(account: Account) {
  intakeMode.value = true
  Object.assign(costForm, emptyCostForm(), {
    account_id: account.id,
    provider_identity: account.name,
    contributor_user_id: authStore.user?.id || 0,
    uploader_user_id: authStore.user?.id || 0
  })
  showCostDialog.value = true
  try {
    await loadReferenceOptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.options'))
  }
}

function closeCostDialog() {
  if (!savingCost.value) showCostDialog.value = false
}

async function openEventDialog(row: SharedPoolAccountCost) {
  eventAccount.value = row
  Object.assign(eventForm, emptyEventForm())
  showEventDialog.value = true
  if (!accountOptions.value.length) {
    try {
      await loadReferenceOptions()
    } catch (error: any) {
      appStore.showError(error?.message || t('admin.sharedPool.errors.options'))
    }
  }
}

function closeEventDialog() {
  if (!savingEvent.value) showEventDialog.value = false
}

async function saveLifecycleEvent() {
  if (!eventAccount.value || !eventForm.date) return
  if (eventForm.event_type === 'replaced' && !eventForm.replacement_account_id) {
    appStore.showError(t('admin.sharedPool.errors.replacementRequired'))
    return
  }
  savingEvent.value = true
  try {
    await adminAPI.sharedPool.recordLifecycleEvent({
      account_id: eventAccount.value.account_id,
      event_type: eventForm.event_type,
      occurred_at: new Date(`${eventForm.date}T12:00:00+08:00`).toISOString(),
      reason: eventForm.reason || undefined,
      payer_user_id: eventAccount.value.contributor_user_id || undefined,
      refund_amount: eventForm.event_type === 'refund' ? eventForm.amount : undefined,
      replacement_account_id: eventForm.event_type === 'replaced' ? eventForm.replacement_account_id : undefined,
      transferred_cost: eventForm.event_type === 'replaced' ? eventForm.amount : undefined
    })
    appStore.showSuccess(t('admin.sharedPool.event.saved'))
    showEventDialog.value = false
    await loadActiveTab()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.event'))
  } finally {
    savingEvent.value = false
  }
}

async function updateFXRate() {
  if (!Number.isFinite(fxRate.value) || fxRate.value <= 0) {
    appStore.showError(t('admin.sharedPool.errors.fxRate'))
    return
  }
  savingFXRate.value = true
  try {
    fxRate.value = await adminAPI.sharedPool.saveFXRate(fxRate.value, startDate.value)
    appStore.showSuccess(t('admin.sharedPool.form.fxRateSaved'))
    await loadActiveTab()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.fxRate'))
  } finally {
    savingFXRate.value = false
  }
}

async function saveAccountCost() {
  if (!costForm.account_id || !costForm.contributor_user_id || !costForm.uploader_user_id) {
    appStore.showError(t('admin.sharedPool.errors.required'))
    return
  }
  if (costForm.purchase_cost <= 0 || costForm.service_end <= costForm.service_start) {
    appStore.showError(t('admin.sharedPool.errors.invalidCostPeriod'))
    return
  }

  savingCost.value = true
  try {
    await adminAPI.sharedPool.createAccountIntake(costForm.account_id, {
      provider_identity: costForm.provider_identity,
      contributor_user_id: costForm.contributor_user_id,
      uploader_user_id: costForm.uploader_user_id,
      purchase_source_name: costForm.purchase_source_name,
      entry_type: costForm.entry_type,
      original_amount: costForm.purchase_cost.toFixed(2),
      currency: costForm.currency,
      fx_rate: costForm.currency === 'CNY' ? '1' : String(fxRate.value),
      cny_amount_minor: Math.round(costForm.purchase_cost * (costForm.currency === 'CNY' ? 1 : fxRate.value) * 100),
      service_start: costForm.service_start,
      service_end: costForm.service_end,
      warranty_end: costForm.warranty_end || undefined,
      order_no: costForm.order_no || undefined,
      purchase_url: costForm.purchase_url || undefined,
      notes: costForm.notes || undefined
    })
    appStore.showSuccess(t('admin.sharedPool.form.saved'))
    showCostDialog.value = false
    await loadActiveTab()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.save'))
  } finally {
    savingCost.value = false
  }
}

async function confirmLockSettlement() {
  if (!settlement.value) return
  showLockConfirm.value = false
  locking.value = true
  try {
    settlement.value = await adminAPI.sharedPool.lockSettlement({
      ...periodParams(),
      settlement_id: settlement.value.id
    })
    appStore.showSuccess(t('admin.sharedPool.settlement.lockedSuccess'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.lock'))
  } finally {
    locking.value = false
  }
}

onMounted(() => {
  void loadActiveTab()
  void adminAPI.sharedPool.getLatestFXRate()
    .then((rate) => { fxRate.value = rate })
    .catch((error: any) => appStore.showError(error?.message || t('admin.sharedPool.errors.fxRate')))
})
</script>
