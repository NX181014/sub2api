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

        <div v-if="activeTab !== 'accounts' && activeTab !== 'ledger'" class="flex flex-col gap-3 p-4 lg:flex-row lg:items-center">
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
              <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 id="pool-account-recovery-title" class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.sharedPool.overview.accountRecovery') }}
                  </h2>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.sharedPool.page.coverage', { start: dateOnly(overview.period_start), end: dateOnly(overview.period_end), count: filteredOverviewAccounts.length }) }}
                  </p>
                </div>
                <div class="w-full sm:w-40">
                  <Select
                    v-model="overviewRecoveryFilter"
                    :options="overviewRecoveryFilterOptions"
                    :aria-label="t('admin.sharedPool.page.recoveryFilter')"
                    @change="overviewPagination.page = 1"
                  />
                </div>
              </div>
              <div v-if="paginatedOverviewAccounts.length" class="h-52 border-b border-gray-100 p-3 dark:border-dark-700">
                <Bar :data="overviewChartData" :options="overviewChartOptions" />
              </div>
              <DataTable :columns="overviewColumns" :data="paginatedOverviewAccounts" row-key="id" :loading="loading" sort-storage-key="shared-pool-overview" :mobile-column-keys="['account_name', 'uploader_name', 'uploaded_at', 'roi_rate', 'remaining_cost']">
                <template #cell-account_name="{ row }">
                  <div class="min-w-0">
                    <button
                      type="button"
                      class="max-w-52 truncate text-left font-medium text-primary-600 hover:underline dark:text-primary-400"
                      :title="row.account_name"
                      @click="openAccountTrace(row.account_id)"
                    >
                      {{ row.account_name }}
                    </button>
                    <p class="max-w-52 truncate text-xs text-gray-500 dark:text-gray-400" :title="row.provider_identity">
                      {{ row.provider_identity || '-' }}
                    </p>
                    <div class="mt-1 flex gap-2 text-xs">
                      <button type="button" class="text-primary-600 hover:underline dark:text-primary-400" @click="openAccountContext('ledger', row)">{{ t('admin.sharedPool.tabs.ledger') }}</button>
                      <button type="button" class="text-primary-600 hover:underline dark:text-primary-400" @click="openAccountContext('settlement', row)">{{ t('admin.sharedPool.tabs.settlement') }}</button>
                    </div>
                  </div>
                </template>
                <template #cell-uploader_name="{ row }">
                  <span class="block max-w-40 truncate" :title="row.uploader_name || ''">{{ row.uploader_name || '-' }}</span>
                </template>
                <template #cell-uploaded_at="{ row }">
                  <span class="whitespace-nowrap">{{ row.uploaded_at ? formatDateTimeToMinute(row.uploaded_at, locale) : '-' }}</span>
                </template>
                <template #cell-status="{ row }">
                  <StatusBadge
                    :status="accountStatus(row.status).badge"
                    :label="t(`admin.sharedPool.status.${accountStatus(row.status).key}`)"
                  />
                </template>
                <template #cell-roi_rate="{ row }">
                  <span class="block font-medium tabular-nums" :class="row.roi_rate >= 100 ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'">
                    {{ formatPercent(row.roi_rate) }}
                  </span>
                  <span class="block text-xs text-gray-500 dark:text-gray-400">{{ t(`admin.sharedPool.page.recoveryStates.${recoveryState(row)}`) }}</span>
                </template>
                <template #cell-remaining_cost="{ row }">
                  <span class="tabular-nums">{{ formatMoney(row.remaining_cost, row.currency) }}</span>
                </template>
                <template #cell-usage_value="{ row }">
                  <span class="tabular-nums">{{ formatMoney(row.usage_value, row.currency) }}</span>
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
              <Pagination
                v-if="filteredOverviewAccounts.length"
                :page="overviewPagination.page"
                :page-size="overviewPagination.page_size"
                :total="filteredOverviewAccounts.length"
                @update:page="overviewPagination.page = $event"
                @update:page-size="changeOverviewPageSize"
              />
            </section>
          </div>
        </template>
        <EmptyState v-else :title="t('admin.sharedPool.empty.overview')" />
      </template>

      <template v-else-if="activeTab === 'accounts'">
        <div
          v-if="retryAccounts.length"
          class="flex items-center justify-between gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-300"
          role="alert"
        >
          <span>{{ t('admin.sharedPool.intake.retryPending', { count: retryAccounts.length }) }}</span>
          <button type="button" class="btn btn-secondary min-h-9 shrink-0 px-3 text-xs" @click="openPendingRetry">
            {{ t('admin.sharedPool.intake.retryAction') }}
          </button>
        </div>
        <AccountsView
          ref="accountsViewRef"
          embedded
          :pool-records="poolRecordsByAccountID"
          @pool-record="openAccountPoolRecord"
          @pool-create-request="prepareAccountAction('create')"
          @pool-import-request="prepareAccountAction('import')"
          @pool-created="completeCreatedAccountIntake"
          @pool-imported="completeImportedAccountsIntake"
        />
      </template>

      <template v-else-if="activeTab === 'ledger'">
        <CostLedgerPanel
          :initial-account-id="routeQueryID('account_id')"
          :initial-purchase-source-id="routeQueryID('purchase_source_id')"
          :initial-uploader-user-id="routeQueryID('uploader_user_id')"
          @open-account="openLedgerAccount"
          @trace-account="openAccountTrace"
          @edit-entry="openLedgerEntry"
        />
      </template>

      <template v-else-if="activeTab === 'settlement'">
        <section class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-6">
          <Select v-model="settlementFilters.account_id" :options="settlementAccountOptions" searchable :aria-label="t('admin.sharedPool.ledger.allAccounts')" @change="applySettlementFilters" />
          <Select v-model="settlementFilters.uploader_user_id" :options="settlementUploaderOptions" searchable :aria-label="t('admin.sharedPool.ledger.allUploaders')" @change="applySettlementFilters" />
          <Select v-model="settlementFilters.payer_user_id" :options="settlementPayerOptions" searchable :aria-label="t('admin.sharedPool.ledger.allPayers')" @change="applySettlementFilters" />
          <Select v-model="settlementFilters.purchase_source_id" :options="settlementSourceOptions" searchable :aria-label="t('admin.sharedPool.ledger.allSources')" @change="applySettlementFilters" />
          <Select v-model="settlementFilters.line_status" :options="settlementLineFilterOptions" :aria-label="t('admin.sharedPool.page.lineStatus')" @change="applySettlementFilters" />
          <button type="button" class="btn btn-secondary min-h-11" :disabled="loading" @click="loadActiveTab">
            <Icon name="refresh" size="sm" />
            {{ t('common.refresh') }}
          </button>
        </section>
        <dl v-if="selectedSettlementAccount" class="grid grid-cols-2 gap-3 rounded-lg border border-primary-100 bg-primary-50/60 px-4 py-3 text-sm dark:border-primary-900/60 dark:bg-primary-900/10 sm:grid-cols-4">
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.account') }}</dt><dd class="mt-1 truncate font-medium"><button type="button" class="max-w-full truncate text-primary-600 hover:underline dark:text-primary-400" :title="selectedSettlementAccount.name" @click="openAccountTrace(selectedSettlementAccount.id)">{{ selectedSettlementAccount.name }}</button></dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploader') }}</dt><dd class="mt-1 truncate font-medium">{{ selectedSettlementAccount.uploader_username || selectedSettlementAccount.uploader_email || '-' }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.page.importBatch') }}</dt><dd class="mt-1 truncate font-medium">{{ selectedSettlementImportBatchID || t('admin.sharedPool.page.singleImport') }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploadedAt') }}</dt><dd class="mt-1 whitespace-nowrap font-medium">{{ formatDateTimeToMinute(selectedSettlementAccount.created_at, locale) }}</dd></div>
        </dl>
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
              <div class="flex flex-wrap items-center justify-end gap-2">
                <span v-if="settlement.status === 'locked' && !settlementAllConfirmed" class="text-xs text-amber-600 dark:text-amber-400">
                  {{ t('admin.sharedPool.settlement.pendingConfirmations', { count: pendingSettlementConfirmations }) }}
                </span>
                <button
                  v-if="settlement.status === 'draft'"
                  type="button"
                  class="btn btn-primary min-h-11"
                  :disabled="settlement.unpriced_usage_count > 0 || locking"
                  @click="showLockConfirm = true"
                >
                  <Icon name="lock" size="sm" />
                  {{ t('admin.sharedPool.settlement.lock') }}
                </button>
                <button
                  v-else-if="settlement.status === 'locked' && settlementAllConfirmed"
                  type="button"
                  class="btn btn-primary min-h-11"
                  :disabled="markingSettlementPaid"
                  @click="showPaidConfirm = true"
                >
                  <Icon name="check" size="sm" />
                  {{ t('admin.sharedPool.settlement.markPaid') }}
                </button>
              </div>
            </div>
            <DataTable :columns="settlementColumns" :data="filteredSettlementLines" row-key="user_id" :loading="loading">
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
              <template #cell-confirmation_status="{ row }">
                <StatusBadge
                  :status="row.net_amount === 0 || row.confirmation_status === 'confirmed' ? 'success' : 'warning'"
                  :label="row.net_amount === 0 ? t('admin.sharedPool.settlement.confirmationNotRequired') : t(`admin.sharedPool.settlement.${row.confirmation_status}`)"
                />
              </template>
              <template #cell-actions="{ row }">
                <button
                  v-if="settlement.status === 'locked' && (row.user_id === authStore.user?.id || authStore.user?.is_primary_admin) && row.net_amount !== 0 && row.confirmation_status !== 'confirmed'"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="confirmingSettlement"
                  @click="confirmSettlementLine(row.user_id)"
                >
                  <Icon name="check" size="xs" />
                  {{ row.user_id === authStore.user?.id ? t('admin.sharedPool.settlement.confirmMine') : t('admin.sharedPool.settlement.resolveMember') }}
                </button>
              </template>
            </DataTable>
          </section>

          <section v-if="settlementAccountGroups.length" class="card overflow-hidden">
            <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.tabs.settlement') }} · {{ t('admin.sharedPool.columns.account') }}</h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.formula') }}</p>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <article v-for="group in paginatedSettlementAccountGroups" :key="group.account_id">
                <div class="flex min-h-14 items-center gap-2 px-3 py-2 sm:px-4">
                  <button type="button" class="min-w-0 flex-1 text-left" @click="toggleSettlementAccount(group.account_id)">
                    <span class="block truncate text-sm font-semibold" :title="group.account_name">{{ group.account_name }}</span>
                    <span class="block text-xs text-gray-500 dark:text-gray-400">{{ group.lines.length }} {{ t('admin.sharedPool.columns.member') }} · {{ formatMoney(group.total_cost, settlement.currency) }}</span>
                  </button>
                  <button type="button" class="inline-flex h-11 w-11 shrink-0 items-center justify-center text-primary-600 dark:text-primary-400" :title="t('admin.sharedPool.actions.poolRecord')" :aria-label="t('admin.sharedPool.actions.poolRecord')" @click="openAccountTrace(group.account_id)"><Icon name="eye" size="sm" /></button>
                  <button type="button" class="min-h-11 shrink-0 px-2 text-xs font-medium text-primary-600 dark:text-primary-400" @click="toggleSettlementAccount(group.account_id)">{{ expandedSettlementAccounts.has(group.account_id) ? t('common.collapse') : t('common.expand') }}</button>
                </div>
                <div v-if="expandedSettlementAccounts.has(group.account_id)" class="border-t border-gray-100 dark:border-dark-700">
                  <DataTable :columns="settlementAccountColumns" :data="group.lines" row-key="id" :mobile-column-keys="['user_name', 'account_usage_weight', 'allocated_cost', 'net_amount']">
                    <template #cell-account_usage_weight="{ row }"><span class="tabular-nums">{{ formatMoney(row.account_usage_weight, settlement.currency) }}</span></template>
                    <template #cell-usage_share="{ row }"><span class="tabular-nums">{{ formatPercent(row.usage_share) }}</span></template>
                    <template #cell-allocated_cost="{ row }"><span class="tabular-nums">{{ formatMoney(row.allocated_cost, settlement.currency) }}</span></template>
                    <template #cell-contribution_credit="{ row }"><span class="tabular-nums">{{ formatMoney(row.contribution_credit, settlement.currency) }}</span></template>
                    <template #cell-net_amount="{ row }"><span class="font-medium tabular-nums">{{ formatMoney(row.net_amount, settlement.currency) }}</span></template>
                    <template #cell-trace_quality="{ row }"><span class="text-xs uppercase text-gray-500 dark:text-gray-400">{{ row.trace_quality }}</span></template>
                  </DataTable>
                  <div v-if="group.costs.length" class="divide-y divide-gray-100 border-t border-gray-100 px-4 text-sm dark:divide-dark-700 dark:border-dark-700">
                    <div v-for="cost in group.costs" :key="cost.id" class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 py-2">
                      <span class="truncate">#{{ cost.cost_entry_id }} · {{ cost.kind }}</span>
                      <span class="font-medium tabular-nums">{{ formatMoney(cost.amount, settlement.currency) }}</span>
                    </div>
                  </div>
                </div>
              </article>
            </div>
            <Pagination v-if="settlementAccountGroups.length > settlementAccountPagination.page_size" :page="settlementAccountPagination.page" :page-size="settlementAccountPagination.page_size" :total="settlementAccountGroups.length" @update:page="settlementAccountPagination.page = $event" @update:page-size="changeSettlementAccountPageSize" />
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
          <section class="card min-w-0 overflow-hidden xl:col-span-3">
            <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.sources.title') }}</h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.sources.sampleHint') }}</p>
            </div>
            <div class="space-y-3 p-3 sm:p-4">
              <article v-for="group in paginatedSources" :key="sourceUploaderKey(group)" class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
                <button type="button" class="flex min-h-14 w-full items-center justify-between gap-3 px-3 py-2 text-left sm:px-4" @click="toggleSourceUploader(group)">
                  <span class="min-w-0">
                    <span class="block truncate font-semibold text-gray-900 dark:text-white">{{ group.uploader_name }}</span>
                    <span class="block text-xs text-gray-500 dark:text-gray-400">{{ group.account_count }} {{ t('admin.sharedPool.columns.accounts') }} / {{ group.sources.length }} {{ t('admin.sharedPool.columns.source') }}</span>
                  </span>
                  <span class="shrink-0 text-xs font-medium text-primary-600 dark:text-primary-400">{{ expandedSourceUploaders.has(sourceUploaderKey(group)) ? t('common.collapse') : t('common.expand') }}</span>
                </button>
                <dl class="grid grid-cols-3 gap-2 border-t border-gray-100 bg-gray-50 px-3 py-2 text-xs dark:border-dark-700 dark:bg-dark-800/60 sm:px-4">
                  <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.cost') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ formatMoney(group.purchase_cost) }}</dd></div>
                  <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.roi') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ formatPercent(group.roi_rate) }}</dd></div>
                  <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.ban30') }}</dt><dd class="mt-1 font-medium tabular-nums text-red-600 dark:text-red-400">{{ formatPercent(group.ban_rate_30d) }}</dd></div>
                </dl>
                <div v-if="expandedSourceUploaders.has(sourceUploaderKey(group))" class="space-y-2 border-t border-gray-100 p-2 dark:border-dark-700 sm:p-3">
                  <article v-for="source in paginatedSourceItems(group)" :key="sourceItemKey(group, source)" class="rounded-md border border-gray-100 dark:border-dark-700">
                    <button type="button" class="flex min-h-12 w-full items-center justify-between gap-3 px-3 py-2 text-left" @click="toggleSourceItem(group, source)">
                      <span class="min-w-0"><span class="block truncate text-sm font-medium">{{ source.name }}</span><span class="block text-xs text-gray-500 dark:text-gray-400">{{ source.account_count }} {{ t('admin.sharedPool.columns.accounts') }} / {{ formatPercent(source.roi_rate) }}</span></span>
                      <span class="shrink-0 text-xs text-primary-600 dark:text-primary-400">{{ expandedSourceItems.has(sourceItemKey(group, source)) ? t('common.collapse') : t('common.expand') }}</span>
                    </button>
                    <div v-if="expandedSourceItems.has(sourceItemKey(group, source))" class="border-t border-gray-100 dark:border-dark-700">
                      <div v-for="account in paginatedSourceAccounts(group, source)" :key="account.account_id" class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 border-b border-gray-100 px-3 py-2 text-sm last:border-b-0 dark:border-dark-700 sm:grid-cols-[minmax(0,1fr)_150px_110px]">
                        <div class="min-w-0"><button type="button" class="block min-h-11 max-w-full truncate text-left font-medium text-primary-600 hover:underline dark:text-primary-400" :title="account.account_name" @click="openAccountTrace(account.account_id)">{{ account.account_name }}</button><p class="truncate text-xs text-gray-500 dark:text-gray-400">{{ formatDateTimeToMinute(account.uploaded_at, locale) }}</p></div>
                        <span class="hidden self-center text-right tabular-nums sm:block">{{ formatMoney(account.purchase_cost) }}</span>
                        <span class="self-center text-right font-medium tabular-nums">{{ formatPercent(account.roi_rate) }}</span>
                      </div>
                      <Pagination
                        v-if="source.accounts.length > sourceDetailPageSize"
                        :page="sourceAccountPage(group, source)"
                        :page-size="sourceDetailPageSize"
                        :show-page-size-selector="false"
                        :total="source.accounts.length"
                        @update:page="setSourceAccountPage(group, source, $event)"
                      />
                      <div class="border-t border-gray-100 p-2 text-right dark:border-dark-700">
                        <button type="button" class="btn btn-secondary min-h-10 px-3 text-xs" :disabled="!sourceID(source)" @click="openSourceLedger(group, source)">{{ t('admin.sharedPool.sources.locateRecords') }}</button>
                      </div>
                    </div>
                  </article>
                  <Pagination
                    v-if="group.sources.length > sourceDetailPageSize"
                    :page="sourceItemPage(group)"
                    :page-size="sourceDetailPageSize"
                    :show-page-size-selector="false"
                    :total="group.sources.length"
                    @update:page="setSourceItemPage(group, $event)"
                  />
                </div>
              </article>
            </div>
            <Pagination v-if="sources.length" :page="sourcePagination.page" :page-size="sourcePagination.page_size" :total="sources.length" @update:page="sourcePagination.page = $event" @update:page-size="changeSourcePageSize" />
          </section>
        </div>
        <EmptyState v-else :title="t('admin.sharedPool.empty.sources')" />
      </template>
    </div>

    <AccountTracePanel
      :show="showTracePanel"
      :loading="traceLoading"
      :account-id="traceAccountId"
      :account="traceAccount"
      :entries="traceEntries"
      :settlement="traceSettlement"
      :recovery="traceRecovery"
      :lifecycle="traceLifecycle"
      @close="closeAccountTrace"
    />

    <BaseDialog :show="showCostDialog" :title="costDialogTitle" width="wide" @close="closeCostDialog">
      <form id="pool-cost-form" class="grid grid-cols-1 gap-4 md:grid-cols-2" @submit.prevent="saveAccountCost">
        <div
          v-if="preAccountDraft"
          class="md:col-span-2 rounded-lg border border-primary-200 bg-primary-50 px-4 py-3 text-sm text-primary-800 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300"
        >
          {{ t('admin.sharedPool.intake.prerequisiteHint') }}
        </div>
        <div
          v-if="recoveryRecord"
          class="md:col-span-2 grid grid-cols-2 gap-3 rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm dark:border-dark-700 dark:bg-dark-800 sm:grid-cols-4"
        >
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.status') }}</p>
            <StatusBadge class="mt-1" :status="accountStatus(recoveryRecord.status).badge" :label="t(`admin.sharedPool.status.${accountStatus(recoveryRecord.status).key}`)" />
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.usageValue') }}</p>
            <p class="mt-1 font-semibold tabular-nums">{{ formatMoney(recoveryRecord.usage_value, recoveryRecord.currency) }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.roiRate') }}</p>
            <p class="mt-1 font-semibold tabular-nums">{{ formatPercent(recoveryRecord.roi_rate) }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.pendingRecovery') }}</p>
            <p class="mt-1 font-semibold tabular-nums">{{ formatMoney(recoveryRecord.remaining_cost, recoveryRecord.currency) }}</p>
          </div>
        </div>
        <div v-if="!preAccountDraft" class="md:col-span-2">
          <label for="pool-account" class="input-label">{{ t('admin.sharedPool.form.account') }} *</label>
          <Select id="pool-account" v-model="costForm.account_id" :options="accountOptions" searchable :disabled="intakeMode || profileReadOnly" :aria-label="t('admin.sharedPool.form.account')" />
        </div>
        <div v-if="!(preAccountDraft && pendingAccountAction === 'import')">
          <label for="pool-provider-identity" class="input-label">{{ t('admin.sharedPool.form.providerIdentity') }} *</label>
          <input id="pool-provider-identity" v-model.trim="costForm.provider_identity" class="input" required :disabled="additionalCostMode" />
        </div>
        <p v-else class="rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-500 dark:bg-gray-800/60 dark:text-gray-400">
          {{ t('admin.sharedPool.intake.importIdentityAuto') }}
        </p>
        <div>
          <label for="pool-source" class="input-label">{{ t('admin.sharedPool.form.source') }} *</label>
          <input id="pool-source" v-model.trim="costForm.purchase_source_name" class="input" required list="pool-source-list" />
          <datalist id="pool-source-list">
            <option v-for="source in purchaseSources" :key="source.id" :value="source.name"></option>
          </datalist>
        </div>
        <div>
          <label for="pool-contributor" class="input-label">{{ t('admin.sharedPool.form.contributor') }} *</label>
          <Select id="pool-contributor" v-model="costForm.contributor_user_id" :options="userOptions" searchable :aria-label="t('admin.sharedPool.form.contributor')" />
        </div>
        <div>
          <label for="pool-uploader" class="input-label">{{ t('admin.sharedPool.form.uploader') }} *</label>
          <Select id="pool-uploader" v-model="costForm.uploader_user_id" :options="userOptions" searchable :disabled="additionalCostMode" :aria-label="t('admin.sharedPool.form.uploader')" />
        </div>
        <div v-if="profileReadOnly" class="flex items-center justify-between gap-4 md:col-span-2">
          <div>
            <label class="input-label mb-0">{{ t('admin.sharedPool.form.costSharingEnabled') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.form.costSharingEnabledHint') }}</p>
          </div>
          <Toggle v-model="costForm.cost_sharing_enabled" />
        </div>
        <div>
          <label for="pool-entry-type" class="input-label">{{ t('admin.sharedPool.form.entryType') }} *</label>
          <Select id="pool-entry-type" v-model="costForm.entry_type" :options="costEntryTypeOptions" :aria-label="t('admin.sharedPool.form.entryType')" />
        </div>
        <div>
          <label for="pool-cost" class="input-label">{{ preAccountDraft && pendingAccountAction === 'import' ? t('admin.sharedPool.form.costPerAccount') : t('admin.sharedPool.form.cost') }} *</label>
          <input id="pool-cost" v-model.number="costForm.purchase_cost" class="input" type="number" min="0.01" step="0.01" required />
          <p v-if="preAccountDraft && pendingAccountAction === 'import'" class="input-hint">{{ t('admin.sharedPool.form.costPerAccountHint') }}</p>
        </div>
        <div>
          <label for="pool-expected-tokens" class="input-label">{{ t('admin.sharedPool.ledger.expectedTokens') }} *</label>
          <input id="pool-expected-tokens" v-model.number="expectedTokenMillions" class="input" type="number" inputmode="decimal" min="0.1" step="0.1" required />
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
          <label for="pool-paid-at" class="input-label">{{ t('admin.sharedPool.ledger.paidAt') }} *</label>
          <input id="pool-paid-at" v-model="costForm.paid_at" class="input" type="date" required />
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
        <div v-if="profileReadOnly" class="flex w-full flex-wrap items-center justify-between gap-2">
          <div v-if="recoveryRecord && !editingLedgerEntry" class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" @click="openEventDialog(recoveryRecord)">
              {{ t('admin.sharedPool.accounts.recordEvent') }}
            </button>
            <button type="button" class="btn btn-primary" @click="beginAdditionalCost">
              <Icon name="plus" size="sm" />
              {{ t('admin.sharedPool.accounts.addCost') }}
            </button>
          </div>
          <FormDialogActions form="pool-cost-form" :submitting="savingCost" @cancel="closeCostDialog" />
        </div>
        <FormDialogActions v-else form="pool-cost-form" :submitting="savingCost" @cancel="closeCostDialog" />
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
        <div>
          <label for="pool-event-reason" class="input-label">{{ t('admin.sharedPool.event.reason') }}</label>
          <textarea id="pool-event-reason" v-model.trim="eventForm.reason" class="input min-h-20" rows="3"></textarea>
        </div>
      </form>
      <template #footer>
        <FormDialogActions form="pool-event-form" :submitting="savingEvent" @cancel="closeEventDialog" />
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
    <ConfirmDialog
      :show="showPaidConfirm"
      :title="t('admin.sharedPool.settlement.markPaidTitle')"
      :message="t('admin.sharedPool.settlement.markPaidMessage')"
      :confirm-text="t('admin.sharedPool.settlement.markPaid')"
      @confirm="confirmMarkSettlementPaid"
      @cancel="showPaidConfirm = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter, type LocationQueryRaw } from 'vue-router'
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
import AccountTracePanel from '@/views/admin/shared-pool/AccountTracePanel.vue'
import CostLedgerPanel from '@/views/admin/shared-pool/CostLedgerPanel.vue'
import {
  BaseDialog,
  ConfirmDialog,
  DataTable,
  EmptyState,
  FormDialogActions,
  LoadingSpinner,
  Pagination,
  StatCard
} from '@/components/common'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type {
  CreateSharedPoolCostRequest,
  CreateSharedPoolIntakeRequest,
  PoolLifecycleEventType,
  PoolAccountStatus,
  PoolPeriodType,
  PoolSettlementStatus,
  SharedPoolAccountCost,
  SharedPoolLifecycleEvent,
  SharedPoolOverview,
  SharedPoolSettlementAccountCost,
  SharedPoolSettlementAccountLine,
  SharedPoolSettlementPreview,
  SharedPoolSourceStat,
  SharedPoolUploaderSourceGroup,
  SharedPoolPurchaseSource,
  SharedPoolLedgerEntry
} from '@/api/admin/sharedPool'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import type { Account } from '@/types'
import { formatDateTimeToMinute } from '@/utils/format'
import {
  accountStatusPresentation,
  buildPoolPeriodParams,
  formatPoolMoney,
  latestPoolRecords,
  resolvePoolPeriod,
  settlementStatusPresentation
} from '@/utils/sharedPool'
import {
  DEFAULT_EXPECTED_TOKEN_COUNT,
  filterRecoveryAccounts,
  filterSettlementLines,
  millionsToTokens,
  recoveryState,
  tokensToMillions,
  type RecoveryFilter,
  type SettlementLineFilter
} from '@/utils/sharedPoolLedger'

ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend)

type TabKey = 'overview' | 'accounts' | 'ledger' | 'settlement' | 'sources'
type PendingAccountAction = 'create' | 'import'
type CreatedAccount = { id: number; name: string }
type AccountsViewExpose = {
  continueCreateWithPoolDraft: () => void
  continueImportWithPoolDraft: () => void
}

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const initialPeriod = resolvePoolPeriod('month')
const requestedTab = Array.isArray(route.query.tab) ? route.query.tab[0] : route.query.tab
const requestedSettlementLineStatus = Array.isArray(route.query.settlement_line_status) ? route.query.settlement_line_status[0] : route.query.settlement_line_status
const activeTab = ref<TabKey>(['overview', 'accounts', 'ledger', 'settlement', 'sources'].includes(String(requestedTab)) ? requestedTab as TabKey : 'accounts')
const periodType = ref<PoolPeriodType>('month')
const startDate = ref(initialPeriod.start)
const endDate = ref(initialPeriod.end)
const loading = ref(false)
const savingCost = ref(false)
const savingEvent = ref(false)
const savingFXRate = ref(false)
const locking = ref(false)
const confirmingSettlement = ref(false)
const markingSettlementPaid = ref(false)
const showCostDialog = ref(false)
const intakeMode = ref(false)
const additionalCostMode = ref(false)
const preAccountDraft = ref(false)
const showEventDialog = ref(false)
const showLockConfirm = ref(false)
const showPaidConfirm = ref(false)
const overview = ref<SharedPoolOverview | null>(null)
const accountCosts = ref<SharedPoolAccountCost[]>([])
const settlement = ref<SharedPoolSettlementPreview | null>(null)
const sources = ref<SharedPoolUploaderSourceGroup[]>([])
const purchaseSources = ref<SharedPoolPurchaseSource[]>([])
const accountOptions = ref<Array<{ value: number; label: string }>>([])
const accountReferences = ref<Account[]>([])
const userOptions = ref<Array<{ value: number; label: string }>>([])
const overviewRecoveryFilter = ref<RecoveryFilter>('all')
const settlementFilters = reactive({
  account_id: routeQueryID('account_id') || '',
  uploader_user_id: routeQueryID('uploader_user_id') || '',
  payer_user_id: routeQueryID('payer_user_id') || '',
  purchase_source_id: routeQueryID('purchase_source_id') || '',
  line_status: (['all', 'pending', 'paid', 'abnormal'].includes(String(requestedSettlementLineStatus)) ? requestedSettlementLineStatus : 'all') as SettlementLineFilter
})
const fxRate = ref(1)
const eventAccount = ref<SharedPoolAccountCost | null>(null)
const accountsViewRef = ref<AccountsViewExpose | null>(null)
const pendingAccountAction = ref<PendingAccountAction | null>(null)
const pendingIntakeDraft = ref<CreateSharedPoolCostRequest | null>(null)
const retryAccounts = ref<CreatedAccount[]>([])
const autoAccountIdentity = ref(false)
const recoveryRecord = ref<SharedPoolAccountCost | null>(null)
const editingLedgerEntry = ref<SharedPoolLedgerEntry | null>(null)
const overviewPagination = reactive({ page: 1, page_size: 10 })
const sourcePagination = reactive({ page: 1, page_size: 8 })
const settlementAccountPagination = reactive({ page: 1, page_size: 10 })
const expandedSourceUploaders = ref(new Set<string>())
const expandedSourceItems = ref(new Set<string>())
const expandedSettlementAccounts = ref(new Set<number>())
const sourceDetailPageSize = 5
const sourceItemPages = reactive<Record<string, number>>({})
const sourceAccountPages = reactive<Record<string, number>>({})
const showTracePanel = ref(false)
const traceLoading = ref(false)
const traceAccountId = ref(0)
const traceAccount = ref<Account | null>(null)
const traceEntries = ref<SharedPoolLedgerEntry[]>([])
const traceSettlement = ref<SharedPoolSettlementPreview | null>(null)
const traceRecovery = ref<SharedPoolAccountCost | null>(null)
const traceLifecycle = ref<SharedPoolLifecycleEvent[]>([])

const tabs = computed(() => [
  { key: 'overview' as const, label: t('admin.sharedPool.tabs.overview'), icon: 'chart' as const },
  { key: 'accounts' as const, label: t('admin.sharedPool.tabs.accounts'), icon: 'server' as const },
  { key: 'ledger' as const, label: t('admin.sharedPool.tabs.ledger'), icon: 'book' as const },
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
  { value: 'retired', label: t('admin.sharedPool.event.retired') }
])

const costEntryTypeOptions = computed(() => [
  { value: 'purchase', label: t('admin.sharedPool.entryTypes.purchase') },
  { value: 'renewal', label: t('admin.sharedPool.entryTypes.renewal') },
  { value: 'topup', label: t('admin.sharedPool.entryTypes.topup') },
  { value: 'price_version', label: t('admin.sharedPool.entryTypes.price_version') },
  { value: 'adjustment', label: t('admin.sharedPool.entryTypes.adjustment') }
])

const poolRecordsByAccountID = computed(() => latestPoolRecords(accountCosts.value))
const profileReadOnly = computed(() => (recoveryRecord.value !== null || editingLedgerEntry.value !== null) && !additionalCostMode.value && retryAccounts.value.length === 0)
const costDialogTitle = computed(() => {
  if (profileReadOnly.value) return t('admin.sharedPool.actions.poolRecord')
  if (preAccountDraft.value) {
    return t(`admin.sharedPool.intake.${pendingAccountAction.value === 'import' ? 'preImportTitle' : 'preCreateTitle'}`)
  }
  return intakeMode.value ? t('admin.sharedPool.intake.title') : t('admin.sharedPool.form.title')
})

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
  { key: 'uploader_name', label: t('admin.sharedPool.columns.uploader'), sortable: true },
  { key: 'uploaded_at', label: t('admin.sharedPool.columns.uploadedAt'), sortable: true },
  { key: 'purchase_source_name', label: t('admin.sharedPool.columns.source'), sortable: true },
  { key: 'status', label: t('admin.sharedPool.columns.status'), sortable: true },
  { key: 'usage_value', label: t('admin.sharedPool.metrics.usageValue'), sortable: true },
  { key: 'roi_rate', label: t('admin.sharedPool.columns.roi'), sortable: true },
  { key: 'remaining_cost', label: t('admin.sharedPool.columns.remaining'), sortable: true },
  { key: 'net_profit', label: t('admin.sharedPool.columns.netProfit'), sortable: true },
  { key: 'recovered_at', label: t('admin.sharedPool.columns.recoveredAt'), sortable: true }
])

const overviewRecoveryFilterOptions = computed(() => [
  { value: 'all', label: t('admin.sharedPool.page.recoveryStates.all') },
  { value: 'unrecovered', label: t('admin.sharedPool.page.recoveryStates.unrecovered') },
  { value: 'recovered', label: t('admin.sharedPool.page.recoveryStates.recovered') },
  { value: 'soon', label: t('admin.sharedPool.page.recoveryStates.soon') },
  { value: 'no_data', label: t('admin.sharedPool.page.recoveryStates.no_data') }
])
const filteredOverviewAccounts = computed(() => filterRecoveryAccounts(overview.value?.accounts || [], overviewRecoveryFilter.value))
const paginatedOverviewAccounts = computed(() => {
  const start = (overviewPagination.page - 1) * overviewPagination.page_size
  return filteredOverviewAccounts.value.slice(start, start + overviewPagination.page_size)
})

const overviewChartData = computed<ChartData<'bar'>>(() => ({
  labels: paginatedOverviewAccounts.value.map((account) => account.account_name),
  datasets: [{
    label: t('admin.sharedPool.columns.roi'),
    data: paginatedOverviewAccounts.value.map((account) => account.roi_rate),
    backgroundColor: 'rgba(34, 197, 94, 0.72)',
    borderRadius: 4
  }]
}))

const overviewChartOptions = computed<ChartOptions<'bar'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  indexAxis: 'y',
  plugins: { legend: { display: false } },
  scales: { x: { beginAtZero: true, ticks: { callback: (value) => `${value}%` } }, y: { grid: { display: false } } }
}))

const settlementColumns = computed<Column[]>(() => [
  { key: 'user_name', label: t('admin.sharedPool.columns.member'), sortable: true },
  { key: 'usage_weight', label: t('admin.sharedPool.columns.usageWeight'), sortable: true },
  { key: 'usage_share', label: t('admin.sharedPool.columns.share'), sortable: true },
  { key: 'allocated_cost', label: t('admin.sharedPool.columns.allocated'), sortable: true },
  { key: 'contribution_credit', label: t('admin.sharedPool.columns.credit'), sortable: true },
  { key: 'net_amount', label: t('admin.sharedPool.columns.net'), sortable: true },
  { key: 'confirmation_status', label: t('admin.sharedPool.columns.confirmation'), sortable: true },
  { key: 'actions', label: t('admin.sharedPool.columns.actions') }
])
const settlementAccountColumns = computed<Column[]>(() => [
  { key: 'user_name', label: t('admin.sharedPool.columns.member') },
  { key: 'account_usage_weight', label: t('admin.sharedPool.columns.usageWeight') },
  { key: 'usage_share', label: t('admin.sharedPool.columns.share') },
  { key: 'allocated_cost', label: t('admin.sharedPool.columns.allocated') },
  { key: 'contribution_credit', label: t('admin.sharedPool.columns.credit') },
  { key: 'net_amount', label: t('admin.sharedPool.columns.net') },
  { key: 'trace_quality', label: t('admin.sharedPool.approval.technicalDetails') }
])
const settlementAccountGroups = computed(() => {
  const groups = new Map<number, { account_id: number; account_name: string; lines: SharedPoolSettlementAccountLine[]; costs: SharedPoolSettlementAccountCost[]; total_cost: number }>()
  const ensure = (accountID: number) => {
    let group = groups.get(accountID)
    if (!group) {
      group = {
        account_id: accountID,
        account_name: accountReferences.value.find((account) => account.id === accountID)?.name || `#${accountID}`,
        lines: [],
        costs: [],
        total_cost: 0
      }
      groups.set(accountID, group)
    }
    return group
  }
  for (const line of settlement.value?.account_lines || []) ensure(line.account_id).lines.push(line)
  for (const cost of settlement.value?.account_costs || []) {
    const group = ensure(cost.account_id)
    group.costs.push(cost)
    group.total_cost += cost.amount
  }
  return [...groups.values()].sort((a, b) => a.account_id - b.account_id)
})
const paginatedSettlementAccountGroups = computed(() => {
  const start = (settlementAccountPagination.page - 1) * settlementAccountPagination.page_size
  return settlementAccountGroups.value.slice(start, start + settlementAccountPagination.page_size)
})

const pendingSettlementConfirmations = computed(() => settlement.value?.lines.filter(
  (line) => line.net_amount !== 0 && line.confirmation_status !== 'confirmed'
).length || 0)
const settlementAllConfirmed = computed(() => pendingSettlementConfirmations.value === 0)
const settlementAccountOptions = computed(() => [{ value: '', label: t('admin.sharedPool.ledger.allAccounts') }, ...accountOptions.value])
const settlementUploaderOptions = computed(() => [{ value: '', label: t('admin.sharedPool.ledger.allUploaders') }, ...userOptions.value])
const settlementPayerOptions = computed(() => [{ value: '', label: t('admin.sharedPool.ledger.allPayers') }, ...userOptions.value])
const settlementSourceOptions = computed(() => [
  { value: '', label: t('admin.sharedPool.ledger.allSources') },
  ...purchaseSources.value.map(source => ({ value: source.id, label: source.name }))
])
const settlementLineFilterOptions = computed(() => [
  { value: 'all', label: t('admin.sharedPool.page.allSettlementStates') },
  { value: 'pending', label: t('admin.sharedPool.page.pendingPayment') },
  { value: 'paid', label: t('admin.sharedPool.page.paidPayment') },
  { value: 'abnormal', label: t('admin.sharedPool.page.abnormal') }
])
const selectedSettlementAccount = computed(() => accountReferences.value.find((account) => account.id === Number(settlementFilters.account_id)))
const selectedSettlementImportBatchID = computed(() => {
  const value = selectedSettlementAccount.value?.extra?.import_batch_id
  return typeof value === 'string' ? value : ''
})
const filteredSettlementLines = computed(() => filterSettlementLines(settlement.value?.lines || [], settlementFilters.line_status))

const paginatedSources = computed(() => {
  const start = (sourcePagination.page - 1) * sourcePagination.page_size
  return sources.value.slice(start, start + sourcePagination.page_size)
})

const hasActiveData = computed(() => {
  if (activeTab.value === 'overview') return !!overview.value
  if (activeTab.value === 'accounts') return true
  if (activeTab.value === 'ledger') return true
  if (activeTab.value === 'settlement') return !!settlement.value
  return sources.value.length > 0
})

const sourceChartData = computed<ChartData<'bar'>>(() => ({
  labels: paginatedSources.value.map((source) => source.uploader_name),
  datasets: [
    {
      label: t('admin.sharedPool.columns.roi'),
      data: paginatedSources.value.map((source) => source.roi_rate),
      backgroundColor: 'rgba(34, 197, 94, 0.72)',
      borderRadius: 4
    },
    {
      label: t('admin.sharedPool.columns.ban30'),
      data: paginatedSources.value.map((source) => source.ban_rate_30d),
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

const shanghaiToday = () => new Date(Date.now() + 8 * 60 * 60 * 1000).toISOString().slice(0, 10)
const emptyCostForm = (): CreateSharedPoolCostRequest => ({
  account_id: 0,
  provider_identity: '',
  contributor_user_id: 0,
  uploader_user_id: 0,
  purchase_source_name: '',
  purchase_url: '',
  order_no: '',
  purchase_cost: 0,
  expected_token_count: DEFAULT_EXPECTED_TOKEN_COUNT,
  cost_sharing_enabled: true,
  entry_type: 'purchase',
  currency: 'CNY',
  service_start: startDate.value,
  service_end: endDate.value,
  warranty_end: '',
	paid_at: shanghaiToday(),
  notes: ''
})

const costForm = reactive<CreateSharedPoolCostRequest>(emptyCostForm())
const expectedTokenMillions = computed({
  get: () => tokensToMillions(costForm.expected_token_count),
  set: (value: number) => { costForm.expected_token_count = millionsToTokens(Number(value)) }
})
const emptyEventForm = () => ({
  event_type: 'banned_confirmed' as PoolLifecycleEventType,
  date: shanghaiToday(),
  reason: ''
})
const eventForm = reactive(emptyEventForm())
const periodParams = () => buildPoolPeriodParams(periodType.value, startDate.value, endDate.value)
const settlementParams = () => ({
  ...periodParams(),
  account_id: Number(settlementFilters.account_id) || undefined,
  uploader_user_id: Number(settlementFilters.uploader_user_id) || undefined,
  payer_user_id: Number(settlementFilters.payer_user_id) || undefined,
  purchase_source_id: Number(settlementFilters.purchase_source_id) || undefined
})

const formatMoney = (amount: number, currency = overview.value?.currency || 'CNY') =>
  formatPoolMoney(amount, currency, locale.value)
const formatPercent = (value: number) => `${(Number.isFinite(value) ? value : 0).toFixed(1)}%`
const dateOnly = (value?: string | null) => value ? value.slice(0, 10) : '-'
const accountStatus = (status: PoolAccountStatus) => accountStatusPresentation(status)
const settlementStatus = (status: PoolSettlementStatus) => settlementStatusPresentation(status)

function resetPendingIntake() {
  pendingAccountAction.value = null
  pendingIntakeDraft.value = null
  retryAccounts.value = []
  autoAccountIdentity.value = false
  preAccountDraft.value = false
  additionalCostMode.value = false
	editingLedgerEntry.value = null
}

function intakePayload(form: CreateSharedPoolCostRequest): CreateSharedPoolIntakeRequest {
  return {
    provider_identity: form.provider_identity,
    contributor_user_id: form.contributor_user_id,
    uploader_user_id: form.uploader_user_id,
    purchase_source_name: form.purchase_source_name,
    entry_type: form.entry_type,
    original_amount: form.purchase_cost.toFixed(2),
    expected_token_count: form.expected_token_count,
    currency: form.currency,
    fx_rate: form.currency === 'CNY' ? '1' : String(fxRate.value),
    cny_amount_minor: Math.round(form.purchase_cost * (form.currency === 'CNY' ? 1 : fxRate.value) * 100),
    service_start: form.service_start,
    service_end: form.service_end,
    warranty_end: form.warranty_end || undefined,
	paid_at: form.paid_at ? new Date(`${form.paid_at}T12:00:00+08:00`).toISOString() : undefined,
    order_no: form.order_no || undefined,
    purchase_url: form.purchase_url || undefined,
    notes: form.notes || undefined
  }
}

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
  if (tab === 'settlement') {
    settlementFilters.account_id = routeQueryID('account_id') || ''
    settlementFilters.uploader_user_id = routeQueryID('uploader_user_id') || ''
    settlementFilters.payer_user_id = routeQueryID('payer_user_id') || ''
    settlementFilters.purchase_source_id = routeQueryID('purchase_source_id') || ''
  }
  void router.replace({ query: { ...route.query, tab } })
  if (tab !== 'ledger') void loadActiveTab()
}

const sourceID = (source: SharedPoolSourceStat) => purchaseSources.value.find(
  (item) => item.name.trim().toLocaleLowerCase() === source.name.trim().toLocaleLowerCase()
)?.id
const sourceUploaderKey = (group: SharedPoolUploaderSourceGroup) => String(group.uploader_user_id || `name:${group.uploader_name}`)
const sourceItemKey = (group: SharedPoolUploaderSourceGroup, source: SharedPoolSourceStat) => `${sourceUploaderKey(group)}:${source.name}`
const sourceItemPage = (group: SharedPoolUploaderSourceGroup) => sourceItemPages[sourceUploaderKey(group)] || 1
const sourceAccountPage = (group: SharedPoolUploaderSourceGroup, source: SharedPoolSourceStat) => sourceAccountPages[sourceItemKey(group, source)] || 1
const pageItems = <T,>(items: T[], page: number) => items.slice((page - 1) * sourceDetailPageSize, page * sourceDetailPageSize)
const paginatedSourceItems = (group: SharedPoolUploaderSourceGroup) => pageItems(group.sources, sourceItemPage(group))
const paginatedSourceAccounts = (group: SharedPoolUploaderSourceGroup, source: SharedPoolSourceStat) => pageItems(source.accounts, sourceAccountPage(group, source))
const setSourceItemPage = (group: SharedPoolUploaderSourceGroup, page: number) => { sourceItemPages[sourceUploaderKey(group)] = page }
const setSourceAccountPage = (group: SharedPoolUploaderSourceGroup, source: SharedPoolSourceStat, page: number) => { sourceAccountPages[sourceItemKey(group, source)] = page }
const toggleSetValue = (target: Set<string>, key: string) => {
  const next = new Set(target)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  return next
}
const toggleSourceUploader = (group: SharedPoolUploaderSourceGroup) => {
  expandedSourceUploaders.value = toggleSetValue(expandedSourceUploaders.value, sourceUploaderKey(group))
}
const toggleSourceItem = (group: SharedPoolUploaderSourceGroup, source: SharedPoolSourceStat) => {
  expandedSourceItems.value = toggleSetValue(expandedSourceItems.value, sourceItemKey(group, source))
}
const openSourceLedger = async (group: SharedPoolUploaderSourceGroup, source: SharedPoolSourceStat) => {
  const id = sourceID(source)
  if (!id) return
  await router.replace({ query: { ...route.query, tab: 'ledger', purchase_source_id: String(id), ...(group.uploader_user_id ? { uploader_user_id: String(group.uploader_user_id) } : {}) } })
  activeTab.value = 'ledger'
}

async function openAccountContext(tab: 'ledger' | 'settlement', row: SharedPoolAccountCost) {
  await router.replace({ query: { ...route.query, tab, account_id: String(row.account_id), ...(row.uploader_user_id ? { uploader_user_id: String(row.uploader_user_id) } : {}) } })
  activeTab.value = tab
  if (tab === 'settlement') {
    settlementFilters.account_id = row.account_id
    settlementFilters.uploader_user_id = row.uploader_user_id || ''
    await loadActiveTab()
  }
}

function changeOverviewPageSize(size: number) { overviewPagination.page_size = size; overviewPagination.page = 1 }
function changeSourcePageSize(size: number) { sourcePagination.page_size = size; sourcePagination.page = 1 }
function changeSettlementAccountPageSize(size: number) { settlementAccountPagination.page_size = size; settlementAccountPagination.page = 1 }
function toggleSettlementAccount(accountID: number) {
  const next = new Set(expandedSettlementAccounts.value)
  if (next.has(accountID)) next.delete(accountID)
  else next.add(accountID)
  expandedSettlementAccounts.value = next
}

async function applySettlementFilters() {
  const query: LocationQueryRaw = { ...route.query, tab: 'settlement' }
  const values: Record<string, string | number> = {
    account_id: settlementFilters.account_id,
    uploader_user_id: settlementFilters.uploader_user_id,
    payer_user_id: settlementFilters.payer_user_id,
    purchase_source_id: settlementFilters.purchase_source_id,
    settlement_line_status: settlementFilters.line_status
  }
  for (const [key, value] of Object.entries(values)) {
    if (value) query[key] = String(value)
    else delete query[key]
  }
  await router.replace({ query })
  settlementAccountPagination.page = 1
  expandedSettlementAccounts.value = new Set()
  await loadActiveTab()
}

async function loadActiveTab() {
  if (activeTab.value === 'ledger') return
  loading.value = true
  try {
    if (activeTab.value === 'overview') {
      overview.value = await adminAPI.sharedPool.getOverview(periodParams())
      overviewPagination.page = 1
    } else if (activeTab.value === 'accounts') {
      const response = await adminAPI.sharedPool.listAccountCosts(periodParams())
      accountCosts.value = response.items || []
    } else if (activeTab.value === 'settlement') {
      if (!accountOptions.value.length || !userOptions.value.length) await loadReferenceOptions()
      settlement.value = await adminAPI.sharedPool.previewSettlement(settlementParams())
      settlementAccountPagination.page = 1
      expandedSettlementAccounts.value = new Set()
    } else {
      const [sourceResponse, sourceOptions] = await Promise.all([
        adminAPI.sharedPool.listSources(periodParams()),
        adminAPI.sharedPool.listPurchaseSources()
      ])
      sources.value = sourceResponse.items || []
      sourcePagination.page = 1
      expandedSourceUploaders.value = new Set()
      expandedSourceItems.value = new Set()
      Object.keys(sourceItemPages).forEach((key) => delete sourceItemPages[key])
      Object.keys(sourceAccountPages).forEach((key) => delete sourceAccountPages[key])
      purchaseSources.value = sourceOptions
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
    adminAPI.sharedPool.listPurchaseSources()
  ])
  const accountItems = [...accounts.items]
  const targetAccountID = routeQueryID('account_id')
  if (targetAccountID && !accountItems.some((account) => account.id === targetAccountID)) {
    accountItems.unshift(await adminAPI.accounts.getById(targetAccountID))
  }
  accountOptions.value = accountItems.map((account) => ({ value: account.id, label: account.name }))
  accountReferences.value = accountItems
  userOptions.value = users.items.map((user) => ({ value: user.id, label: user.username || user.email }))
  purchaseSources.value = sourceResponse
}

async function resolvePurchaseSourceID(name: string): Promise<number | undefined> {
  const normalized = name.trim().toLocaleLowerCase()
  if (!normalized) return undefined
  const existing = purchaseSources.value.find((source) => source.name.trim().toLocaleLowerCase() === normalized)
  if (existing) return existing.id
  const created = await adminAPI.sharedPool.createPurchaseSource(name.trim())
  purchaseSources.value.push(created)
  return created.id
}

async function openAccountPoolRecord(account: Pick<Account, 'id' | 'name'>, record?: SharedPoolAccountCost) {
  intakeMode.value = true
  additionalCostMode.value = false
  const existing = record || poolRecordsByAccountID.value[account.id]
	editingLedgerEntry.value = null
  recoveryRecord.value = existing || null
  Object.assign(costForm, emptyCostForm(), {
    account_id: account.id,
    provider_identity: existing?.provider_identity || account.name,
    contributor_user_id: existing?.contributor_user_id || authStore.user?.id || 0,
    uploader_user_id: existing?.uploader_user_id || authStore.user?.id || 0,
    purchase_source_name: existing?.purchase_source_name || '',
    purchase_url: existing?.purchase_url || '',
    order_no: existing?.order_no || '',
    purchase_cost: existing?.purchase_cost || 0,
    expected_token_count: existing?.expected_token_count || DEFAULT_EXPECTED_TOKEN_COUNT,
    cost_sharing_enabled: existing?.cost_sharing_enabled ?? true,
    entry_type: existing?.entry_type || 'purchase',
    currency: existing?.currency || 'CNY',
    service_start: existing?.service_start || startDate.value,
    service_end: existing?.service_end || endDate.value,
    warranty_end: existing?.warranty_end || '',
	paid_at: existing?.paid_at?.slice(0, 10) || shanghaiToday(),
    notes: existing?.notes || ''
  })
  showCostDialog.value = true
  try {
    await loadReferenceOptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.options'))
  }
}

async function openLedgerEntry(entry: SharedPoolLedgerEntry) {
	try {
		const account = await adminAPI.accounts.getById(entry.account_id)
		await loadReferenceOptions()
		editingLedgerEntry.value = entry
		recoveryRecord.value = null
		intakeMode.value = true
		additionalCostMode.value = false
		Object.assign(costForm, emptyCostForm(), {
			account_id: entry.account_id,
			provider_identity: account.provider_identity || account.name,
			contributor_user_id: account.contributor_user_id || entry.payer_user_id,
			uploader_user_id: account.created_by_user_id || authStore.user?.id || 0,
			purchase_source_name: entry.purchase_source || '',
			purchase_url: entry.purchase_url || '',
			order_no: entry.order_no || '',
			purchase_cost: Number(entry.original_amount),
			expected_token_count: entry.expected_token_count || DEFAULT_EXPECTED_TOKEN_COUNT,
			cost_sharing_enabled: account.cost_sharing_enabled ?? true,
			entry_type: entry.entry_type,
			currency: entry.currency,
			service_start: entry.service_start.slice(0, 10),
			service_end: entry.service_end.slice(0, 10),
			warranty_end: entry.warranty_end?.slice(0, 10) || '',
			paid_at: entry.paid_at.slice(0, 10),
			notes: entry.note || ''
		})
		showCostDialog.value = true
	} catch (error: any) {
		appStore.showError(error?.message || t('admin.sharedPool.errors.load'))
	}
}

function beginAdditionalCost() {
  if (!recoveryRecord.value) return
  const nextServiceStart = costForm.service_end
  additionalCostMode.value = true
  Object.assign(costForm, {
    entry_type: 'renewal',
    purchase_cost: 0,
    order_no: '',
    purchase_url: '',
    service_start: nextServiceStart,
    service_end: '',
    warranty_end: '',
    notes: ''
  })
}

function routeQueryID(key: 'account_id' | 'ledger_entry_id' | 'purchase_source_id' | 'uploader_user_id' | 'payer_user_id') {
  const raw = route.query[key]
  const id = Number(Array.isArray(raw) ? raw[0] : raw)
  return Number.isSafeInteger(id) && id > 0 ? id : 0
}

async function openAccountTrace(accountID: number, syncRoute = true) {
  if (!Number.isSafeInteger(accountID) || accountID <= 0) return
  traceAccountId.value = accountID
  showTracePanel.value = true
  traceLoading.value = true
  traceAccount.value = null
  traceEntries.value = []
  traceSettlement.value = null
  traceRecovery.value = null
  traceLifecycle.value = []
  if (syncRoute) await router.replace({ query: { ...route.query, account_id: String(accountID), trace: '1' } })

  const currentRecovery = overview.value?.accounts.find((item) => item.account_id === accountID)
  const results = await Promise.allSettled([
    adminAPI.accounts.getById(accountID),
    adminAPI.sharedPool.listLedgerEntries({ page: 1, page_size: 200, account_id: accountID }),
    adminAPI.sharedPool.listSettlements({ page: 1, page_size: 1, account_id: accountID }).then((page) => {
      const id = page.items[0]?.id
      return id ? adminAPI.sharedPool.getSettlement(id) : null
    }),
    currentRecovery ? Promise.resolve(currentRecovery) : adminAPI.sharedPool.getOverview({ ...periodParams(), account_id: accountID }).then((item) => item.accounts.find((account) => account.account_id === accountID) || null),
    adminAPI.sharedPool.listLifecycle(accountID)
  ])
  if (traceAccountId.value !== accountID) return
  const [accountResult, entriesResult, settlementResult, recoveryResult, lifecycleResult] = results
  if (accountResult.status === 'fulfilled') traceAccount.value = accountResult.value
  if (entriesResult.status === 'fulfilled') traceEntries.value = entriesResult.value.items || []
  if (settlementResult.status === 'fulfilled') traceSettlement.value = settlementResult.value
  if (recoveryResult.status === 'fulfilled') traceRecovery.value = recoveryResult.value
  if (lifecycleResult.status === 'fulfilled') traceLifecycle.value = lifecycleResult.value
  traceLoading.value = false
  const failed = results.find((result) => result.status === 'rejected')
  if (failed?.status === 'rejected') appStore.showError(failed.reason?.message || t('admin.sharedPool.errors.load'))
}

function closeAccountTrace() {
  showTracePanel.value = false
  const query = { ...route.query }
  delete query.trace
  void router.replace({ query })
}

async function openLedgerAccount(accountID: number) {
  try {
    const account = await adminAPI.accounts.getById(accountID)
    await openAccountPoolRecord(account)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.load'))
  }
}

async function openRouteTarget() {
  const accountID = routeQueryID('account_id')
  if (!accountID) return
  if (route.query.trace === '1') {
    await openAccountTrace(accountID, false)
    return
  }
  if (activeTab.value !== 'accounts') return
  const ledgerEntryID = routeQueryID('ledger_entry_id')
  await router.replace({ query: { tab: 'accounts' } })
  if (ledgerEntryID) {
    const response = await adminAPI.sharedPool.listLedgerEntries({ page: 1, page_size: 100, account_id: accountID })
    const entry = response.items.find((item) => item.id === ledgerEntryID)
    if (entry) await openLedgerEntry(entry)
    return
  }
  const record = poolRecordsByAccountID.value[accountID]
  const accountName = accountOptions.value.find((item) => item.value === accountID)?.label || record?.account_name || `#${accountID}`
  await openAccountPoolRecord({ id: accountID, name: accountName }, record)
}

async function prepareAccountAction(action: PendingAccountAction) {
  if (pendingIntakeDraft.value) {
    if (retryAccounts.value.length) openPendingRetry()
    else await reopenDraftForManualSelection()
    return
  }
  pendingAccountAction.value = action
  pendingIntakeDraft.value = null
  retryAccounts.value = []
  preAccountDraft.value = true
  additionalCostMode.value = false
  intakeMode.value = false
  recoveryRecord.value = null
  Object.assign(costForm, emptyCostForm(), {
    contributor_user_id: authStore.user?.id || 0,
    uploader_user_id: authStore.user?.id || 0
  })
  showCostDialog.value = true
  try {
    await loadReferenceOptions()
  } catch (error: any) {
    showCostDialog.value = false
    resetPendingIntake()
    appStore.showError(error?.message || t('admin.sharedPool.errors.options'))
  }
}

function openPendingRetry() {
  const draft = pendingIntakeDraft.value
  if (!draft || !retryAccounts.value.length) return
  const account = retryAccounts.value[0]
  Object.assign(costForm, draft, {
    account_id: account.id,
    provider_identity: draft.provider_identity || account.name
  })
  preAccountDraft.value = false
  intakeMode.value = true
  recoveryRecord.value = poolRecordsByAccountID.value[account.id] || null
  showCostDialog.value = true
}

function closeCostDialog() {
  if (savingCost.value) return
  showCostDialog.value = false
  additionalCostMode.value = false
	editingLedgerEntry.value = null
  if (preAccountDraft.value) resetPendingIntake()
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
  savingEvent.value = true
  try {
    await adminAPI.sharedPool.recordLifecycleEvent({
      account_id: eventAccount.value.account_id,
      event_type: eventForm.event_type,
      occurred_at: new Date(`${eventForm.date}T12:00:00+08:00`).toISOString(),
      reason: eventForm.reason || undefined
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

async function persistPendingAccounts(accounts: CreatedAccount[]) {
  const draft = pendingIntakeDraft.value
  if (!draft) return
  if (pendingAccountAction.value === 'import' || accounts.length > 1) autoAccountIdentity.value = true

  savingCost.value = true
  const failed: CreatedAccount[] = []
  for (const account of accounts) {
    try {
      await adminAPI.sharedPool.createAccountIntake(account.id, intakePayload({
        ...draft,
        account_id: account.id,
        provider_identity: autoAccountIdentity.value ? account.name : (draft.provider_identity || account.name)
      }))
    } catch {
      failed.push(account)
    }
  }

  retryAccounts.value = failed
  savingCost.value = false
  if (failed.length) {
    const account = failed[0]
    Object.assign(costForm, draft, {
      account_id: account.id,
      provider_identity: autoAccountIdentity.value ? account.name : (draft.provider_identity || account.name)
    })
    preAccountDraft.value = false
    intakeMode.value = true
    recoveryRecord.value = null
    showCostDialog.value = true
    appStore.showError(t('admin.sharedPool.intake.retryFailed', { count: failed.length }))
  } else {
    showCostDialog.value = false
    resetPendingIntake()
    appStore.showSuccess(t('admin.sharedPool.form.saved'))
  }
  await loadActiveTab()
}

async function reopenDraftForManualSelection() {
  if (!pendingIntakeDraft.value) return
  Object.assign(costForm, pendingIntakeDraft.value)
  preAccountDraft.value = false
  intakeMode.value = false
  showCostDialog.value = true
  try {
    await loadReferenceOptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.options'))
  }
  appStore.showWarning(t('admin.sharedPool.intake.selectCreatedRetry'))
}

async function completeCreatedAccountIntake(accounts: CreatedAccount[] = []) {
  if (!pendingIntakeDraft.value) return
  if (accounts.length) {
    await persistPendingAccounts(accounts)
    return
  }

  await reopenDraftForManualSelection()
}

async function completeImportedAccountsIntake(accounts: CreatedAccount[]) {
  if (!pendingIntakeDraft.value) return
  if (accounts.length) {
    await persistPendingAccounts(accounts)
    return
  }
  await reopenDraftForManualSelection()
}

async function saveAccountCost() {
  const autoImportIdentity = preAccountDraft.value && pendingAccountAction.value === 'import'
  if ((!preAccountDraft.value && !costForm.account_id) || (!autoImportIdentity && !costForm.provider_identity) || !costForm.purchase_source_name || !costForm.contributor_user_id || !costForm.uploader_user_id) {
    appStore.showError(t('admin.sharedPool.errors.required'))
    return
  }
  if (costForm.purchase_cost <= 0 || costForm.service_end <= costForm.service_start) {
    appStore.showError(t('admin.sharedPool.errors.invalidCostPeriod'))
    return
  }
  if (!Number.isSafeInteger(costForm.expected_token_count) || costForm.expected_token_count <= 0) {
    appStore.showError(t('admin.sharedPool.errors.invalidExpectedTokens'))
    return
  }

  if (preAccountDraft.value) {
    pendingIntakeDraft.value = { ...costForm }
    preAccountDraft.value = false
    showCostDialog.value = false
    if (pendingAccountAction.value === 'import') accountsViewRef.value?.continueImportWithPoolDraft()
    else accountsViewRef.value?.continueCreateWithPoolDraft()
    return
  }

  if (retryAccounts.value.length) {
    pendingIntakeDraft.value = { ...costForm, account_id: 0 }
    await persistPendingAccounts([...retryAccounts.value])
    return
  }

  savingCost.value = true
  try {
    const payload = intakePayload(costForm)
    const editedCost = editingLedgerEntry.value || recoveryRecord.value
    if (profileReadOnly.value && editedCost) {
      const purchaseSourceID = await resolvePurchaseSourceID(costForm.purchase_source_name)
      const result = await adminAPI.sharedPool.createCost({
        account_id: costForm.account_id,
        payer_user_id: costForm.contributor_user_id,
        purchase_source_id: purchaseSourceID,
        entry_type: payload.entry_type,
        original_amount: payload.original_amount,
        expected_token_count: payload.expected_token_count,
        currency: payload.currency,
        fx_rate: payload.fx_rate,
        service_start: payload.service_start,
        service_end: payload.service_end,
        warranty_end: payload.warranty_end,
		paid_at: costForm.paid_at ? new Date(`${costForm.paid_at}T12:00:00+08:00`).toISOString() : undefined,
        order_no: payload.order_no,
        purchase_url: payload.purchase_url,
        notes: payload.notes,
		supersedes_id: editedCost.id,
        provider_identity: costForm.provider_identity,
        contributor_user_id: costForm.contributor_user_id,
        uploader_user_id: costForm.uploader_user_id,
		cost_sharing_enabled: costForm.cost_sharing_enabled
      })
      appStore.showSuccess(t(result.approval_required ? 'admin.accounts.approval.updateSubmitted' : 'admin.sharedPool.form.saved'))
    } else if (additionalCostMode.value && recoveryRecord.value) {
      const purchaseSourceID = await resolvePurchaseSourceID(costForm.purchase_source_name)
      await adminAPI.sharedPool.createCost({
        account_id: costForm.account_id,
        payer_user_id: costForm.contributor_user_id,
        purchase_source_id: purchaseSourceID,
        entry_type: payload.entry_type,
        original_amount: payload.original_amount,
        expected_token_count: payload.expected_token_count,
        currency: payload.currency,
        fx_rate: payload.fx_rate,
        service_start: payload.service_start,
        service_end: payload.service_end,
        warranty_end: payload.warranty_end,
		paid_at: payload.paid_at,
        order_no: payload.order_no,
        purchase_url: payload.purchase_url,
        notes: payload.notes
      })
    } else {
      await adminAPI.sharedPool.createAccountIntake(costForm.account_id, payload)
    }
    if (!profileReadOnly.value) appStore.showSuccess(t('admin.sharedPool.form.saved'))
    showCostDialog.value = false
    resetPendingIntake()
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

async function confirmSettlementLine(userId: number) {
  if (!settlement.value?.id || confirmingSettlement.value) return
  confirmingSettlement.value = true
  try {
    settlement.value = await adminAPI.sharedPool.confirmSettlement(settlement.value.id, userId)
    appStore.showSuccess(t('admin.sharedPool.settlement.confirmedSuccess'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.confirmSettlement'))
  } finally {
    confirmingSettlement.value = false
  }
}

async function confirmMarkSettlementPaid() {
  if (!settlement.value?.id || markingSettlementPaid.value) return
  showPaidConfirm.value = false
  markingSettlementPaid.value = true
  try {
    settlement.value = await adminAPI.sharedPool.markSettlementPaid(settlement.value.id)
    appStore.showSuccess(t('admin.sharedPool.settlement.markedPaidSuccess'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.markPaid'))
  } finally {
    markingSettlementPaid.value = false
  }
}

watch(
  () => [route.query.trace, route.query.account_id],
  () => {
    const accountID = routeQueryID('account_id')
    if (route.query.trace === '1' && accountID && (accountID !== traceAccountId.value || !showTracePanel.value)) void openAccountTrace(accountID, false)
    if (route.query.trace !== '1') showTracePanel.value = false
  }
)

onMounted(async () => {
  await loadActiveTab()
  try {
    await openRouteTarget()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.load'))
  }
  void adminAPI.sharedPool.getLatestFXRate()
    .then((rate) => { fxRate.value = rate })
    .catch((error: any) => appStore.showError(error?.message || t('admin.sharedPool.errors.fxRate')))
})
</script>
