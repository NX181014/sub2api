<template>
  <AppLayout>
    <div class="shared-pool-shell min-w-0 space-y-5">
      <header class="min-w-0 border-b border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 px-4 py-4 sm:px-5 lg:flex-row lg:items-center lg:justify-between">
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.title') }}</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.description') }}</p>
          </div>
          <div class="sm:hidden">
            <Select
              :model-value="activeTab"
              :options="tabs.map(tab => ({ value: tab.key, label: tab.label }))"
              :aria-label="t('admin.sharedPool.title')"
              @change="switchTab($event as TabKey)"
            />
          </div>
          <nav class="hidden flex-wrap items-center gap-1 sm:flex" :aria-label="t('admin.sharedPool.title')">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="inline-flex min-h-11 items-center gap-2 rounded-md px-3 text-sm font-medium transition-colors"
            :class="activeTab === tab.key
              ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900'
              : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
            :aria-current="activeTab === tab.key ? 'page' : undefined"
            @click="switchTab(tab.key)"
          >
            <Icon :name="tab.icon" size="sm" />
            {{ tab.label }}
          </button>
          </nav>
        </div>

        <div v-if="activeTab !== 'accounts' && activeTab !== 'ledger'" class="flex flex-col gap-3 border-t border-gray-100 bg-gray-50/70 p-3 dark:border-dark-700 dark:bg-dark-800/40 sm:p-4 lg:flex-row lg:items-end">
          <div class="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row sm:items-end">
            <div class="w-full sm:w-36">
              <label for="pool-period-type" class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300">
                {{ t('admin.sharedPool.period.label') }}
              </label>
              <Select
                id="pool-period-type"
                v-model="periodType"
                :options="periodOptions"
                :aria-label="t('admin.sharedPool.period.label')"
                @change="handlePeriodTypeChange"
              />
            </div>
            <div class="min-w-0">
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="handleDateRangeChange"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.sharedPool.period.timezone') }}
              </p>
            </div>
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
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="loading" @click="loadActiveTab">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </div>
        </div>
      </header>

      <div v-if="loading && !hasActiveData" class="flex min-h-64 items-center justify-center">
        <LoadingSpinner />
      </div>

      <template v-else-if="activeTab === 'overview'">
        <template v-if="overview">
          <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800" aria-labelledby="pool-recovery-title">
            <div class="grid gap-5 p-4 sm:p-5 lg:grid-cols-[minmax(260px,1.3fr)_repeat(4,minmax(120px,1fr))] lg:items-center">
              <div>
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
              <div class="mt-4">
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
              </div>
              <dl class="contents text-sm">
                <div class="border-l-0 border-gray-100 lg:border-l lg:pl-5 dark:border-dark-700"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.purchaseCost') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatMoney(overview.summary.total_purchase_cost) }}</dd></div>
                <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.usageValue') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-green-600 dark:text-green-400">{{ formatMoney(overview.summary.total_usage_value) }}</dd></div>
                <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.pendingRecovery') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-amber-600 dark:text-amber-400">{{ formatMoney(overview.summary.pending_recovery) }}</dd></div>
                <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.bannedLoss') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-red-600 dark:text-red-400">{{ formatMoney(overview.summary.banned_loss) }}</dd></div>
              </dl>
            </div>
          </section>

          <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800" aria-labelledby="pool-account-recovery-title">
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
                    @change="applyOverviewRecoveryFilter"
                  />
                </div>
              </div>
              <div v-if="paginatedOverviewAccounts.length" class="divide-y divide-gray-100 dark:divide-dark-700">
                <article v-for="row in paginatedOverviewAccounts" :key="row.id" class="grid min-w-0 gap-4 px-4 py-4 lg:grid-cols-[minmax(220px,1.35fr)_minmax(180px,1fr)_repeat(3,minmax(120px,.75fr))_auto] lg:items-center">
                  <div class="min-w-0 lg:pr-3">
                    <button
                      type="button"
                      class="block min-h-11 max-w-full truncate text-left font-semibold text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
                      :title="row.account_name"
                      @click="openAccountTrace(row.account_id)"
                    >
                      {{ row.account_name }}
                    </button>
                    <p class="max-w-52 truncate text-xs text-gray-500 dark:text-gray-400" :title="row.provider_identity">
                      {{ row.provider_identity || '-' }}
                    </p>
                    <p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ row.uploader_name || '-' }} · {{ row.purchase_source_name || '-' }}</p>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center justify-between gap-3"><StatusBadge :status="availabilityPresentation(row.availability_status, row.account_status).badge" :label="t(`admin.accounts.status.${availabilityPresentation(row.availability_status, row.account_status).key}`)" /><span class="font-semibold tabular-nums" :class="row.roi_rate >= 100 ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'">{{ formatPercent(row.roi_rate) }}</span></div>
                    <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700"><div class="h-full rounded-full bg-green-500" :style="{ width: `${Math.min(Math.max(row.roi_rate, 0), 100)}%` }"></div></div>
                    <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t(`admin.sharedPool.page.recoveryStates.${recoveryState(row)}`) }}</span>
                  </div>
                  <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.metrics.usageValue') }}</span><span class="mt-1 block font-medium tabular-nums">{{ formatMoney(row.usage_value, row.currency) }}</span></div>
                  <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.remaining') }}</span><span class="mt-1 block font-medium tabular-nums">{{ formatMoney(row.remaining_cost, row.currency) }}</span></div>
                  <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.netProfit') }}</span><span class="mt-1 block font-medium tabular-nums" :class="(row.net_profit || 0) >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">{{ formatMoney(row.net_profit || 0, row.currency) }}</span></div>
                  <div class="flex flex-wrap gap-2 lg:justify-end"><button type="button" class="btn btn-secondary min-h-11 px-3 text-xs" @click="openAccountContext('ledger', row)">{{ t('admin.sharedPool.tabs.ledger') }}</button><button type="button" class="btn btn-secondary min-h-11 px-3 text-xs" @click="openAccountContext('settlement', row)">{{ t('admin.sharedPool.tabs.settlement') }}</button></div>
                </article>
              </div>
              <EmptyState v-else :title="t('admin.sharedPool.empty.overview')" />
              <Pagination
                v-if="filteredOverviewAccounts.length"
                :page="overviewPagination.page"
                :page-size="overviewPagination.page_size"
                :total="filteredOverviewAccounts.length"
                @update:page="changeOverviewPage"
                @update:page-size="changeOverviewPageSize"
              />
          </section>
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
          :initial-workbench-context="initialWorkbenchContext"
          :pool-records="poolRecordsByAccountID"
          @pool-record="openAccountPoolRecord"
          @trace-account="openAccountTrace"
          @workbench-context="syncWorkbenchContext"
          @refreshed="refreshAccountPoolRecords"
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
        <dl v-if="selectedSettlementIdentity" class="grid grid-cols-2 gap-3 rounded-lg border border-primary-100 bg-primary-50/60 px-4 py-3 text-sm dark:border-primary-900/60 dark:bg-primary-900/10 sm:grid-cols-4">
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.account') }}</dt><dd class="mt-1 truncate font-medium"><button type="button" class="max-w-full truncate text-primary-600 hover:underline dark:text-primary-400" :title="selectedSettlementIdentity.name" @click="openAccountTrace(selectedSettlementIdentity.id)">{{ selectedSettlementIdentity.name }}</button></dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploader') }}</dt><dd class="mt-1 truncate font-medium">{{ selectedSettlementIdentity.uploader || '-' }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.page.importBatch') }}</dt><dd class="mt-1 truncate font-medium">{{ selectedSettlementIdentity.importBatch || t('admin.sharedPool.page.singleImport') }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.uploadedAt') }}</dt><dd class="mt-1 whitespace-nowrap font-medium">{{ selectedSettlementIdentity.createdAt ? formatDateTimeToMinute(selectedSettlementIdentity.createdAt, locale) : '-' }}</dd></div>
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

          <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <ol class="grid grid-cols-2 border-b border-gray-100 dark:border-dark-700 sm:grid-cols-4" :aria-label="t('admin.sharedPool.tabs.settlement')">
              <li v-for="(stage, index) in settlementStages" :key="stage.key" class="relative flex min-h-16 items-center gap-3 px-4 py-3" :class="index < settlementStages.length - 1 ? 'sm:border-r sm:border-gray-100 sm:dark:border-dark-700' : ''">
                <span class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold" :class="index <= settlementStageIndex ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'">{{ index + 1 }}</span>
                <span class="text-sm font-medium" :class="index <= settlementStageIndex ? 'text-gray-900 dark:text-white' : 'text-gray-500 dark:text-gray-400'">{{ stage.label }}</span>
              </li>
            </ol>
            <dl class="grid grid-cols-2 divide-x divide-y divide-gray-100 text-sm dark:divide-dark-700 lg:grid-cols-4 lg:divide-y-0">
              <div class="p-4"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.totalCost') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums">{{ formatMoney(settlement.total_cost, settlement.currency) }}</dd></div>
              <div class="p-4"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.usageWeight') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums">{{ formatMoney(settlement.total_usage_weight, settlement.currency) }}</dd></div>
              <div class="p-4"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.coverage') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums" :class="settlement.pricing_coverage >= 99 ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'">{{ formatPercent(settlement.pricing_coverage) }}</dd></div>
              <div class="p-4"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.carryForward') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums">{{ formatMoney(settlement.carry_forward, settlement.currency) }}</dd></div>
            </dl>
          </section>

          <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
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
            <div v-if="filteredSettlementLines.length" class="divide-y divide-gray-100 dark:divide-dark-700">
              <article v-for="row in filteredSettlementLines" :key="row.user_id" class="grid min-w-0 gap-4 px-4 py-4 lg:grid-cols-[minmax(180px,1.2fr)_repeat(3,minmax(120px,.8fr))_minmax(150px,1fr)_auto] lg:items-center">
                <div class="min-w-0"><span class="block truncate font-semibold text-gray-900 dark:text-white" :title="row.user_name">{{ row.user_name }}</span><StatusBadge class="mt-2" :status="row.net_amount === 0 || row.confirmation_status === 'confirmed' ? 'success' : 'warning'" :label="row.net_amount === 0 ? t('admin.sharedPool.settlement.confirmationNotRequired') : t(`admin.sharedPool.settlement.${row.confirmation_status}`)" /></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.usageWeight') }}</span><span class="mt-1 block font-medium tabular-nums">{{ formatMoney(row.usage_weight, settlement.currency) }}</span><span class="text-xs text-gray-500 dark:text-gray-400">{{ formatPercent(row.usage_share) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.allocated') }}</span><span class="mt-1 block font-medium tabular-nums">{{ formatMoney(row.allocated_cost, settlement.currency) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.credit') }}</span><span class="mt-1 block font-medium tabular-nums text-green-600 dark:text-green-400">-{{ formatMoney(row.contribution_credit, settlement.currency) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.net') }}</span><span class="mt-1 block font-semibold tabular-nums" :class="row.net_amount > 0 ? 'text-red-600 dark:text-red-400' : row.net_amount < 0 ? 'text-green-600 dark:text-green-400' : 'text-gray-600 dark:text-gray-300'">{{ row.net_amount > 0 ? t('admin.sharedPool.settlement.payable') : row.net_amount < 0 ? t('admin.sharedPool.settlement.receivable') : '' }} {{ formatMoney(Math.abs(row.net_amount), settlement.currency) }}</span></div>
                <button
                  v-if="settlement.status === 'locked' && (row.user_id === authStore.user?.id || authStore.user?.is_primary_admin) && row.net_amount !== 0 && row.confirmation_status !== 'confirmed'"
                  type="button"
                  class="btn btn-secondary min-h-11"
                  :disabled="confirmingSettlement"
                  @click="confirmSettlementLine(row.user_id)"
                >
                  <Icon name="check" size="xs" />
                  {{ row.user_id === authStore.user?.id ? t('admin.sharedPool.settlement.confirmMine') : t('admin.sharedPool.settlement.resolveMember') }}
                </button>
              </article>
            </div>
            <EmptyState v-else :title="t('admin.sharedPool.empty.settlement')" />
          </section>

          <section v-if="settlementAccountGroups.length" class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.tabs.settlement') }} · {{ t('admin.sharedPool.columns.account') }}</h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.settlement.formula') }}</p>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <article v-for="group in paginatedSettlementAccountGroups" :key="group.account_id">
                <div class="flex min-h-14 items-center gap-2 px-3 py-2 sm:px-4">
                  <button type="button" class="min-w-0 flex-1 py-2 text-left" @click="openSettlementAccountDetail(group.account_id)">
                    <span class="block truncate text-sm font-semibold" :title="group.account_name">{{ group.account_name }}</span>
                    <span class="block text-xs text-gray-500 dark:text-gray-400">{{ group.lines.length }} {{ t('admin.sharedPool.columns.member') }} · {{ formatMoney(group.total_cost, settlement.currency) }}</span>
                  </button>
                  <div v-if="group.context" class="hidden shrink-0 flex-col items-end gap-1 sm:flex">
                    <StatusBadge :status="availabilityPresentation(group.context.availability_status, group.context.account_status).badge" :label="t(`admin.accounts.status.${availabilityPresentation(group.context.availability_status, group.context.account_status).key}`)" />
                    <span class="text-xs text-gray-500 dark:text-gray-400">{{ lifecycleStatusLabel(group.context.latest_lifecycle_status) }}</span>
                  </div>
                  <button type="button" class="inline-flex h-11 w-11 shrink-0 items-center justify-center text-primary-600 dark:text-primary-400" :title="t('admin.sharedPool.actions.poolRecord')" :aria-label="t('admin.sharedPool.actions.poolRecord')" @click="openAccountTrace(group.account_id)"><Icon name="eye" size="sm" /></button>
                  <button type="button" class="btn btn-secondary min-h-11 shrink-0 px-3 text-xs" @click="openSettlementAccountDetail(group.account_id)">{{ t('common.view') }}</button>
                </div>
              </article>
            </div>
            <Pagination v-if="settlementAccountGroups.length > settlementAccountPagination.page_size" :page="settlementAccountPagination.page" :page-size="settlementAccountPagination.page_size" :total="settlementAccountGroups.length" @update:page="changeSettlementAccountPage" @update:page-size="changeSettlementAccountPageSize" />
          </section>
        </template>
        <EmptyState v-else :title="t('admin.sharedPool.empty.settlement')" />
      </template>

      <template v-else>
        <div v-if="sourceRankings.length" class="grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-[minmax(280px,.8fr)_minmax(0,1.7fr)]">
          <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800" aria-labelledby="pool-source-chart-title">
            <h2 id="pool-source-chart-title" class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.sharedPool.sources.chartTitle') }}
            </h2>
            <div class="h-72">
              <Bar :data="sourceChartData" :options="sourceChartOptions" />
            </div>
          </section>
          <section class="min-w-0 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
              <div><h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.sources.title') }}</h2><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.sources.sampleHint') }}</p></div>
              <div class="w-full sm:w-48"><Select v-model="sourceUploaderFilter" :options="sourceUploaderOptions" searchable :aria-label="t('admin.sharedPool.columns.uploader')" @change="applySourceUploaderFilter" /></div>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <article v-for="(source, index) in paginatedSources" :key="source.name" class="grid min-w-0 gap-4 px-4 py-4 sm:grid-cols-2 lg:grid-cols-[44px_minmax(160px,1.4fr)_repeat(4,minmax(90px,.7fr))_auto] lg:items-center">
                <span class="inline-flex h-8 w-8 items-center justify-center rounded-md bg-gray-100 text-sm font-semibold tabular-nums text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ (sourcePagination.page - 1) * sourcePagination.page_size + index + 1 }}</span>
                <div class="min-w-0"><h3 class="truncate font-semibold text-gray-900 dark:text-white" :title="source.name">{{ source.name }}</h3><p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" :title="source.uploaderNames.join(', ')">{{ source.account_count }} {{ t('admin.sharedPool.columns.accounts') }} · {{ source.uploaderNames.join(', ') || '-' }}</p></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.roi') }}</span><span class="mt-1 block font-semibold tabular-nums" :class="source.roi_rate >= 100 ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'">{{ formatPercent(source.roi_rate) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.ban30') }}</span><span class="mt-1 block font-medium tabular-nums text-red-600 dark:text-red-400">{{ formatPercent(source.ban_rate_30d) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.survival') }}</span><span class="mt-1 block font-medium tabular-nums">{{ source.average_survival_days.toFixed(1) }}</span></div>
                <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.cost') }}</span><span class="mt-1 block font-medium tabular-nums">{{ formatMoney(source.purchase_cost) }}</span></div>
                <button type="button" class="btn btn-secondary min-h-11 px-3 text-xs" @click="selectedSourceName = source.name">{{ t('common.view') }}</button>
              </article>
            </div>
            <Pagination v-if="filteredSourceRankings.length" :page="sourcePagination.page" :page-size="sourcePagination.page_size" :total="filteredSourceRankings.length" @update:page="changeSourcePage" @update:page-size="changeSourcePageSize" />
          </section>
        </div>
        <EmptyState v-else :title="t('admin.sharedPool.empty.sources')" />
      </template>
    </div>

    <BaseDialog
      :show="!!selectedSettlementAccountGroup"
      :title="selectedSettlementAccountGroup?.account_name || t('admin.sharedPool.tabs.settlement')"
      width="wide"
      @close="selectedSettlementAccountID = null"
    >
      <template v-if="selectedSettlementAccountGroup">
        <dl class="grid grid-cols-2 gap-3 border-b border-gray-100 pb-4 text-sm dark:border-dark-700 sm:grid-cols-4">
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.member') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ selectedSettlementAccountGroup.lines.length }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.cost') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ formatMoney(selectedSettlementAccountGroup.total_cost, settlement?.currency) }}</dd></div>
          <div v-if="selectedSettlementAccountGroup.context"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.status') }}</dt><dd class="mt-1"><StatusBadge :status="availabilityPresentation(selectedSettlementAccountGroup.context.availability_status, selectedSettlementAccountGroup.context.account_status).badge" :label="t(`admin.accounts.status.${availabilityPresentation(selectedSettlementAccountGroup.context.availability_status, selectedSettlementAccountGroup.context.account_status).key}`)" /></dd></div>
          <div v-if="selectedSettlementAccountGroup.context"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.impacts.lifecycle_events') }}</dt><dd class="mt-1 font-medium">{{ lifecycleStatusLabel(selectedSettlementAccountGroup.context.latest_lifecycle_status) }}</dd></div>
        </dl>
        <div class="mt-4 divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
          <article v-for="line in selectedSettlementAccountGroup.lines" :key="line.id" class="grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-[minmax(140px,1fr)_repeat(4,minmax(100px,.7fr))]">
            <div class="min-w-0"><span class="block truncate font-semibold" :title="line.user_name">{{ line.user_name }}</span><span class="mt-1 block text-xs uppercase text-gray-500 dark:text-gray-400">{{ line.trace_quality }}</span></div>
            <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.usageWeight') }}</span><span class="mt-1 block tabular-nums">{{ formatMoney(line.account_usage_weight, settlement?.currency) }}</span></div>
            <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.share') }}</span><span class="mt-1 block tabular-nums">{{ formatPercent(line.usage_share) }}</span></div>
            <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.allocated') }}</span><span class="mt-1 block tabular-nums">{{ formatMoney(line.allocated_cost, settlement?.currency) }}</span></div>
            <div><span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.net') }}</span><span class="mt-1 block font-semibold tabular-nums">{{ formatMoney(line.net_amount, settlement?.currency) }}</span></div>
          </article>
        </div>
        <div v-if="selectedSettlementAccountGroup.costs.length" class="mt-4 divide-y divide-gray-100 rounded-lg border border-gray-200 px-4 text-sm dark:divide-dark-700 dark:border-dark-700">
          <div v-for="cost in selectedSettlementAccountGroup.costs" :key="cost.id" class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 py-3"><span class="truncate">#{{ cost.cost_entry_id }} · {{ cost.kind }}</span><span class="font-medium tabular-nums">{{ formatMoney(cost.amount, settlement?.currency) }}</span></div>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!selectedSource" :title="selectedSource?.name || t('admin.sharedPool.sources.title')" width="wide" @close="selectedSourceName = ''">
      <template v-if="selectedSource">
        <dl class="grid grid-cols-2 gap-3 border-b border-gray-100 pb-4 text-sm dark:border-dark-700 sm:grid-cols-4">
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.accounts') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ selectedSource.account_count }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.roi') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ formatPercent(selectedSource.roi_rate) }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.ban30') }}</dt><dd class="mt-1 font-semibold tabular-nums text-red-600 dark:text-red-400">{{ formatPercent(selectedSource.ban_rate_30d) }}</dd></div>
          <div><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.survival') }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ selectedSource.average_survival_days.toFixed(1) }}</dd></div>
        </dl>
        <div class="mt-4 divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
          <article v-for="account in selectedSource.accounts" :key="`${account.uploader_user_id}:${account.account_id}`" class="grid min-w-0 gap-3 p-4 sm:grid-cols-[minmax(180px,1.4fr)_minmax(120px,1fr)_auto_auto] sm:items-center">
            <div class="min-w-0"><button type="button" class="block min-h-11 max-w-full truncate text-left font-semibold text-primary-600 hover:underline dark:text-primary-400" :title="account.account_name" @click="openAccountTrace(account.account_id)">{{ account.account_name }}</button><p class="truncate text-xs text-gray-500 dark:text-gray-400">{{ formatDateTimeToMinute(account.uploaded_at, locale) }}</p></div>
            <span class="truncate text-sm text-gray-600 dark:text-gray-300" :title="account.uploader_name">{{ account.uploader_name }}</span>
            <StatusBadge :status="availabilityPresentation(account.availability_status, account.account_status).badge" :label="t(`admin.accounts.status.${availabilityPresentation(account.availability_status, account.account_status).key}`)" />
            <span class="font-semibold tabular-nums">{{ formatPercent(account.roi_rate) }}</span>
          </article>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="selectedSourceName = ''">{{ t('common.close') }}</button>
          <button v-if="selectedSource" type="button" class="btn btn-primary" :disabled="!sourceID(selectedSource)" @click="openSourceLedger(selectedSource)">{{ t('admin.sharedPool.sources.locateRecords') }}</button>
        </div>
      </template>
    </BaseDialog>

    <AccountTracePanel
      :show="showTracePanel"
      :loading="traceLoading"
      :account-id="traceAccountId"
      :account="traceAccount"
      :entries="traceEntries"
      :entries-page="traceEntryPage"
      :entries-total="traceEntryTotal"
      :settlement="traceSettlement"
      :settlement-page="traceSettlementPage"
      :settlement-total="traceSettlementTotal"
      :recovery="traceRecovery"
      :lifecycle="traceLifecycle"
      :approvals="traceApprovals"
      :approvals-page="traceApprovalPage"
      :approvals-total="traceApprovalTotal"
      @close="closeAccountTrace"
      @entry-page="loadTraceEntryPage"
      @settlement-page="loadTraceSettlementPage"
      @approval-page="loadTraceApprovalPage"
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
          <label for="pool-paid-at" class="input-label">{{ t('admin.sharedPool.ledger.paidAt') }}</label>
          <input id="pool-paid-at" v-model="costForm.paid_at" class="input" type="datetime-local" :aria-invalid="costPaidAtFuture" aria-describedby="pool-paid-at-hint" />
          <p id="pool-paid-at-hint" class="input-hint" :class="costPaidAtFuture ? 'text-red-600 dark:text-red-400' : ''">{{ t(costPaidAtFuture ? 'admin.sharedPool.errors.futurePaidAt' : 'admin.sharedPool.ledger.paidAtHint') }}</p>
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
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
  EmptyState,
  FormDialogActions,
  LoadingSpinner,
  Pagination
} from '@/components/common'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { AccountListFilters } from '@/api/admin/accounts'
import type {
  CreateSharedPoolCostRequest,
  CreateSharedPoolIntakeRequest,
  PoolApproval,
  PoolAvailabilityStatus,
  PoolLifecycleEventType,
  PoolAccountStatus,
  PoolPeriodType,
  PoolSettlementStatus,
  SharedPoolAccountCost,
  SharedPoolAccountContext,
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
  formatPoolPaidAtInput,
  formatPoolMoney,
  isPoolPaidAtFuture,
  latestPoolRecords,
  poolPaidAtToISOString,
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
  reload: (resetPage?: boolean) => Promise<void>
}
type WorkbenchScope = 'all' | 'standalone' | 'uploader' | 'batch'
type WorkbenchUsageStatus = NonNullable<AccountListFilters['usage_status']>
type WorkbenchContext = {
  scope: WorkbenchScope
  import_batch_id?: string
  uploader_user_id?: number | string
  usage_status?: WorkbenchUsageStatus
  page: number
  page_size: number
  search: string
  platform: string
  type: string
  status: string
  group: string
  privacy_mode: string
  sort_by: string
  sort_order: 'asc' | 'desc'
}
type RankedSourceAccount = SharedPoolSourceStat['accounts'][number] & {
  uploader_name: string
  uploader_user_id?: number | null
}
type RankedSource = Omit<SharedPoolSourceStat, 'accounts'> & {
  uploaderNames: string[]
  accounts: RankedSourceAccount[]
}

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const initialPeriod = resolvePoolPeriod('month')
const requestedTab = Array.isArray(route.query.tab) ? route.query.tab[0] : route.query.tab
const requestedSettlementLineStatus = Array.isArray(route.query.settlement_line_status) ? route.query.settlement_line_status[0] : route.query.settlement_line_status
const queryString = (key: string) => String(Array.isArray(route.query[key]) ? route.query[key]?.[0] || '' : route.query[key] || '')
const queryPage = (key: string, fallback: number) => Math.max(1, Number(queryString(key)) || fallback)
const routeTab = (): TabKey => ['overview', 'accounts', 'ledger', 'settlement', 'sources'].includes(queryString('tab'))
  ? queryString('tab') as TabKey
  : 'accounts'
const routeUsageStatus = (): WorkbenchUsageStatus => {
  const value = queryString('account_usage_status')
  return ['in_use', 'ready', 'unused', 'attention', 'error', 'restricted', 'disabled'].includes(value)
    ? value as WorkbenchUsageStatus
    : 'all'
}
const initialWorkbenchContext = computed<Partial<WorkbenchContext>>(() => {
  const requestedScope = queryString('account_scope')
  return {
    scope: (['all', 'standalone', 'uploader', 'batch'].includes(requestedScope) ? requestedScope : 'all') as WorkbenchScope,
    import_batch_id: queryString('import_batch_id'),
    uploader_user_id: queryString('uploader_user_id'),
    page: queryPage('account_page', 1),
    page_size: queryPage('account_page_size', 20),
    search: queryString('account_search'),
    platform: queryString('account_platform'),
    type: queryString('account_type'),
    status: queryString('account_status'),
    usage_status: routeUsageStatus(),
    group: queryString('account_group'),
    privacy_mode: queryString('account_privacy_mode'),
    sort_by: queryString('account_sort_by'),
    sort_order: queryString('account_sort_order') === 'asc' ? 'asc' : 'desc'
  }
})
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
const overviewRecoveryFilter = ref<RecoveryFilter>(
  ['all', 'unrecovered', 'recovered', 'soon', 'no_data'].includes(queryString('overview_recovery'))
    ? queryString('overview_recovery') as RecoveryFilter
    : 'all'
)
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
const overviewPagination = reactive({ page: queryPage('overview_page', 1), page_size: queryPage('overview_page_size', 10) })
const sourcePagination = reactive({ page: queryPage('source_page', 1), page_size: queryPage('source_page_size', 8) })
const sourceUploaderFilter = ref<number | string>(routeQueryID('uploader_user_id') || '')
const settlementAccountPagination = reactive({ page: queryPage('settlement_page', 1), page_size: queryPage('settlement_page_size', 10) })
const selectedSourceName = ref('')
const selectedSettlementAccountID = ref<number | null>(null)
const showTracePanel = ref(false)
const traceLoading = ref(false)
const traceAccountId = ref(0)
const traceAccount = ref<Account | null>(null)
const traceEntries = ref<SharedPoolLedgerEntry[]>([])
const traceEntryPage = ref(1)
const traceEntryTotal = ref(0)
const traceSettlement = ref<SharedPoolSettlementPreview | null>(null)
const traceSettlementPage = ref(1)
const traceSettlementTotal = ref(0)
const traceRecovery = ref<SharedPoolAccountCost | null>(null)
const traceLifecycle = ref<SharedPoolLifecycleEvent[]>([])
const traceApprovals = ref<PoolApproval[]>([])
const traceApprovalPage = ref(1)
const traceApprovalTotal = ref(0)

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

const settlementContextByID = computed(() => new Map(
  (settlement.value?.account_contexts || []).map((context) => [context.id, context])
))
const settlementAccountGroups = computed(() => {
  const groups = new Map<number, { account_id: number; account_name: string; context?: SharedPoolAccountContext; lines: SharedPoolSettlementAccountLine[]; costs: SharedPoolSettlementAccountCost[]; total_cost: number }>()
  const ensure = (accountID: number) => {
    let group = groups.get(accountID)
    if (!group) {
      const context = settlementContextByID.value.get(accountID)
      group = {
        account_id: accountID,
        account_name: context?.name || `#${accountID}`,
        context,
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
const selectedSettlementAccountGroup = computed(() => settlementAccountGroups.value.find(
  (group) => group.account_id === selectedSettlementAccountID.value
))

const pendingSettlementConfirmations = computed(() => settlement.value?.lines.filter(
  (line) => line.net_amount !== 0 && line.confirmation_status !== 'confirmed'
).length || 0)
const settlementAllConfirmed = computed(() => pendingSettlementConfirmations.value === 0)
const settlementStages = computed(() => [
  { key: 'draft', label: t('admin.sharedPool.status.draft') },
  { key: 'locked', label: t('admin.sharedPool.status.locked') },
  { key: 'confirmed', label: t('admin.sharedPool.settlement.confirmed') },
  { key: 'paid', label: t('admin.sharedPool.status.paid') }
])
const settlementStageIndex = computed(() => {
  if (settlement.value?.status === 'paid') return 3
  if (settlement.value?.status === 'locked') return settlementAllConfirmed.value ? 2 : 1
  return 0
})
const settlementAccountOptions = computed(() => {
  const options = new Map<number, string>()
  for (const context of settlement.value?.account_contexts || []) options.set(context.id, context.name)
  for (const option of accountOptions.value) if (!options.has(option.value)) options.set(option.value, option.label)
  return [{ value: '', label: t('admin.sharedPool.ledger.allAccounts') }, ...[...options].map(([value, label]) => ({ value, label }))]
})
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
const selectedSettlementContext = computed(() => settlementContextByID.value.get(Number(settlementFilters.account_id)))
const selectedSettlementIdentity = computed(() => {
  const account = selectedSettlementAccount.value
  const context = selectedSettlementContext.value
  if (!account && !context) return null
  const batch = account?.extra?.import_batch_id
  return {
    id: account?.id || context!.id,
    name: account?.name || context!.name,
    uploader: account?.uploader_username || account?.uploader_email || context?.created_by_username || context?.created_by_email || '',
    importBatch: typeof batch === 'string' ? batch : context?.import_batch_id || '',
    createdAt: account?.created_at || context?.created_at || ''
  }
})
const filteredSettlementLines = computed(() => filterSettlementLines(settlement.value?.lines || [], settlementFilters.line_status))

const sourceRankings = computed<RankedSource[]>(() => {
  const grouped = new Map<string, Array<{ group: SharedPoolUploaderSourceGroup; source: SharedPoolSourceStat }>>()
  for (const group of sources.value) {
    if (sourceUploaderFilter.value && group.uploader_user_id !== Number(sourceUploaderFilter.value)) continue
    for (const source of group.sources) {
      const key = source.name.trim().toLocaleLowerCase()
      grouped.set(key, [...(grouped.get(key) || []), { group, source }])
    }
  }
  return [...grouped.values()].map((parts) => {
    const first = parts[0].source
    const sampleSize = parts.reduce((sum, part) => sum + part.source.sample_size, 0)
    const weighted = (field: 'ban_rate_7d' | 'ban_rate_30d' | 'ban_rate_90d' | 'refund_rate' | 'average_survival_days') => {
      const totalWeight = parts.reduce((sum, part) => sum + Math.max(part.source.sample_size, part.source.account_count, 1), 0)
      return parts.reduce((sum, part) => sum + part.source[field] * Math.max(part.source.sample_size, part.source.account_count, 1), 0) / totalWeight
    }
    const accounts = [...new Map(parts.flatMap(({ group, source }) => source.accounts.map((account) => [account.account_id, {
      ...account,
      uploader_name: group.uploader_name,
      uploader_user_id: group.uploader_user_id
    }] as const))).values()]
    const purchaseCost = parts.reduce((sum, part) => sum + part.source.purchase_cost, 0)
    const usageValue = parts.reduce((sum, part) => sum + part.source.usage_value, 0)
    const recoveryParts = parts.filter((part) => part.source.average_recovery_days != null)
    return {
      ...first,
      account_count: accounts.length,
      sample_size: sampleSize,
      purchase_cost: purchaseCost,
      usage_value: usageValue,
      roi_rate: purchaseCost > 0 ? usageValue / purchaseCost * 100 : 0,
      ban_rate_7d: weighted('ban_rate_7d'),
      ban_rate_30d: weighted('ban_rate_30d'),
      ban_rate_90d: weighted('ban_rate_90d'),
      refund_rate: weighted('refund_rate'),
      average_survival_days: weighted('average_survival_days'),
      average_recovery_days: recoveryParts.length
        ? recoveryParts.reduce((sum, part) => sum + Number(part.source.average_recovery_days), 0) / recoveryParts.length
        : null,
      uploaderNames: [...new Set(parts.map((part) => part.group.uploader_name))],
      accounts
    }
  }).sort((a, b) => b.roi_rate - a.roi_rate || b.account_count - a.account_count)
})
const sourceUploaderOptions = computed(() => [
  { value: '', label: t('admin.sharedPool.ledger.allUploaders') },
  ...[...new Map(sources.value.filter((group) => group.uploader_user_id).map((group) => [group.uploader_user_id!, group.uploader_name])).entries()]
    .map(([value, label]) => ({ value, label }))
])
const filteredSourceRankings = computed(() => sourceRankings.value)
const paginatedSources = computed(() => {
  const start = (sourcePagination.page - 1) * sourcePagination.page_size
  return filteredSourceRankings.value.slice(start, start + sourcePagination.page_size)
})
const selectedSource = computed(() => sourceRankings.value.find((source) => source.name === selectedSourceName.value))

const hasActiveData = computed(() => {
  if (activeTab.value === 'overview') return !!overview.value
  if (activeTab.value === 'accounts') return true
  if (activeTab.value === 'ledger') return true
  if (activeTab.value === 'settlement') return !!settlement.value
  return sourceRankings.value.length > 0
})

const sourceChartData = computed<ChartData<'bar'>>(() => ({
  labels: paginatedSources.value.map((source) => source.name),
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
	paid_at: '',
  notes: ''
})

const costForm = reactive<CreateSharedPoolCostRequest>(emptyCostForm())
const costPaidAtFuture = computed(() => isPoolPaidAtFuture(costForm.paid_at))
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
const periodParams = () => ({
  ...buildPoolPeriodParams(periodType.value, startDate.value, endDate.value),
  account_id: routeQueryID('account_id') || undefined
})
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
const availabilityPresentation = (availability?: PoolAvailabilityStatus | null, accountStatusValue?: string | null) => {
  const value = availability || (accountStatusValue === 'error' ? 'error' : accountStatusValue === 'active' ? 'normal' : 'inactive')
  const states = {
    normal: { badge: 'success', key: 'active' },
    error: { badge: 'danger', key: 'error' },
    rate_limited: { badge: 'warning', key: 'rateLimited' },
    overloaded: { badge: 'danger', key: 'overloaded' },
    temp_unschedulable: { badge: 'warning', key: 'tempUnschedulable' },
    inactive: { badge: 'inactive', key: 'inactive' },
    manual_unschedulable: { badge: 'inactive', key: 'unschedulable' }
  } as const
  return states[value]
}
const lifecycleStatusLabel = (status: string) => {
  const state = accountStatus(status === 'banned_confirmed' ? 'banned' : status === 'refund' ? 'warning' : status === 'retired' || status === 'replaced' ? 'inactive' : 'active')
  return t(`admin.sharedPool.status.${state.key}`)
}

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
	paid_at: poolPaidAtToISOString(form.paid_at),
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

async function syncWorkbenchContext(context: WorkbenchContext) {
  const query: LocationQueryRaw = { ...route.query, tab: 'accounts', account_scope: context.scope }
  if (context.import_batch_id) query.import_batch_id = context.import_batch_id
  else delete query.import_batch_id
  if (context.uploader_user_id) query.uploader_user_id = String(context.uploader_user_id)
  else delete query.uploader_user_id
  query.account_page = String(context.page)
  query.account_page_size = String(context.page_size)
  query.account_sort_by = context.sort_by
  query.account_sort_order = context.sort_order
  for (const [key, value] of Object.entries({
    account_search: context.search,
    account_platform: context.platform,
    account_type: context.type,
    account_status: context.status,
    account_usage_status: context.usage_status === 'all' ? '' : context.usage_status,
    account_group: context.group,
    account_privacy_mode: context.privacy_mode
  })) {
    if (value) query[key] = value
    else delete query[key]
  }
  await router.replace({ query })
}

async function refreshAccountPoolRecords() {
  try {
    const requests: Array<Promise<unknown>> = [adminAPI.sharedPool.listAccountCosts(periodParams())]
    if (showTracePanel.value && traceAccountId.value) requests.push(adminAPI.accounts.getById(traceAccountId.value))
    const [costs, account] = await Promise.all(requests)
    accountCosts.value = (costs as { items: SharedPoolAccountCost[] }).items || []
    if (account) traceAccount.value = account as Account
  } catch (error) {
    console.error('Failed to refresh shared-pool account context:', error)
  }
}

const sourceID = (source: SharedPoolSourceStat) => purchaseSources.value.find(
  (item) => item.name.trim().toLocaleLowerCase() === source.name.trim().toLocaleLowerCase()
)?.id
const openSourceLedger = async (source: SharedPoolSourceStat) => {
  const id = sourceID(source)
  if (!id) return
  selectedSourceName.value = ''
  await router.replace({ query: { ...route.query, tab: 'ledger', purchase_source_id: String(id) } })
  activeTab.value = 'ledger'
}

async function openAccountContext(tab: 'ledger' | 'settlement', row: SharedPoolAccountCost) {
  await router.replace({
    query: {
      ...route.query,
      tab,
      account_id: String(row.account_id),
      ...(tab === 'ledger' ? { ledger_view: 'entries' } : {}),
      ...(row.uploader_user_id ? { uploader_user_id: String(row.uploader_user_id) } : {})
    }
  })
  if (activeTab.value === tab) return
  activeTab.value = tab
  if (tab === 'settlement') {
    syncSettlementFiltersFromRoute()
    await loadActiveTab()
  }
}

async function syncPageQuery(prefix: 'overview' | 'source' | 'settlement', page: number, pageSize: number) {
  await router.replace({ query: { ...route.query, [`${prefix}_page`]: String(page), [`${prefix}_page_size`]: String(pageSize) } })
}
function changeOverviewPage(page: number) {
  overviewPagination.page = page
  void syncPageQuery('overview', page, overviewPagination.page_size)
}
function changeOverviewPageSize(size: number) {
  overviewPagination.page_size = size
  changeOverviewPage(1)
}
function changeSourcePage(page: number) {
  sourcePagination.page = page
  void syncPageQuery('source', page, sourcePagination.page_size)
}
function changeSourcePageSize(size: number) {
  sourcePagination.page_size = size
  changeSourcePage(1)
}
async function applySourceUploaderFilter() {
  sourcePagination.page = 1
  const query: LocationQueryRaw = { ...route.query, source_page: '1' }
  if (sourceUploaderFilter.value) query.uploader_user_id = String(sourceUploaderFilter.value)
  else delete query.uploader_user_id
  await router.replace({ query })
}
function changeSettlementAccountPage(page: number) {
  settlementAccountPagination.page = page
  void syncPageQuery('settlement', page, settlementAccountPagination.page_size)
}
function changeSettlementAccountPageSize(size: number) {
  settlementAccountPagination.page_size = size
  changeSettlementAccountPage(1)
}
async function applyOverviewRecoveryFilter() {
  overviewPagination.page = 1
  const query: LocationQueryRaw = { ...route.query, overview_page: '1' }
  if (overviewRecoveryFilter.value === 'all') delete query.overview_recovery
  else query.overview_recovery = overviewRecoveryFilter.value
  await router.replace({ query })
}
function openSettlementAccountDetail(accountID: number) {
  selectedSettlementAccountID.value = accountID
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
  settlementAccountPagination.page = 1
  query.settlement_page = '1'
  await router.replace({ query })
  selectedSettlementAccountID.value = null
  await loadActiveTab()
}

async function loadActiveTab() {
  if (activeTab.value === 'ledger') return
  loading.value = true
  try {
    if (activeTab.value === 'overview') {
      overview.value = await adminAPI.sharedPool.getOverview(periodParams())
    } else if (activeTab.value === 'accounts') {
      const response = await adminAPI.sharedPool.listAccountCosts(periodParams())
      accountCosts.value = response.items || []
    } else if (activeTab.value === 'settlement') {
      if (!accountOptions.value.length || !userOptions.value.length) await loadReferenceOptions()
      settlement.value = await adminAPI.sharedPool.previewSettlement(settlementParams())
      selectedSettlementAccountID.value = null
    } else {
      const [sourceResponse, sourceOptions] = await Promise.all([
        adminAPI.sharedPool.listSources(periodParams()),
        adminAPI.sharedPool.listPurchaseSources({ referenced: true })
      ])
      sources.value = sourceResponse.items || []
      selectedSourceName.value = ''
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
	paid_at: formatPoolPaidAtInput(existing?.paid_at),
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
			paid_at: formatPoolPaidAtInput(entry.paid_at),
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

async function fetchTraceSettlementPage(accountID: number, page: number) {
  const response = await adminAPI.sharedPool.listSettlements({ page, page_size: 1, account_id: accountID })
  const summary = response.items[0]
  return {
    page: response.page || page,
    total: response.total || 0,
    settlement: summary?.id ? await adminAPI.sharedPool.getSettlement(summary.id) : summary || null
  }
}

async function loadTraceSettlementPage(page: number) {
  const accountID = traceAccountId.value
  if (!accountID || page < 1) return
  traceLoading.value = true
  try {
    const result = await fetchTraceSettlementPage(accountID, page)
    if (traceAccountId.value !== accountID) return
    traceSettlement.value = result.settlement
    traceSettlementPage.value = result.page
    traceSettlementTotal.value = result.total
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.load'))
  } finally {
    if (traceAccountId.value === accountID) traceLoading.value = false
  }
}

async function loadTraceEntryPage(page: number) {
  const accountID = traceAccountId.value
  if (!accountID || page < 1) return
  traceLoading.value = true
  try {
    const result = await adminAPI.sharedPool.listLedgerEntries({ page, page_size: 20, account_id: accountID })
    if (traceAccountId.value !== accountID) return
    traceEntries.value = result.items || []
    traceEntryPage.value = result.page || page
    traceEntryTotal.value = result.total || 0
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.load'))
  } finally {
    if (traceAccountId.value === accountID) traceLoading.value = false
  }
}

async function loadTraceApprovalPage(page: number) {
  const accountID = traceAccountId.value
  if (!accountID || page < 1) return
  traceLoading.value = true
  try {
    const result = await adminAPI.sharedPool.listApprovals({ account_id: accountID, page, page_size: 20 })
    if (traceAccountId.value !== accountID) return
    traceApprovals.value = result.items || []
    traceApprovalPage.value = result.page || page
    traceApprovalTotal.value = result.total || 0
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.errors.load'))
  } finally {
    if (traceAccountId.value === accountID) traceLoading.value = false
  }
}

async function openAccountTrace(accountID: number, syncRoute = true) {
  if (!Number.isSafeInteger(accountID) || accountID <= 0) return
  traceAccountId.value = accountID
  showTracePanel.value = true
  traceLoading.value = true
  traceAccount.value = null
  traceEntries.value = []
  traceEntryPage.value = 1
  traceEntryTotal.value = 0
  traceSettlement.value = null
  traceSettlementPage.value = 1
  traceSettlementTotal.value = 0
  traceRecovery.value = null
  traceLifecycle.value = []
  traceApprovals.value = []
  traceApprovalPage.value = 1
  traceApprovalTotal.value = 0
  if (syncRoute) await router.replace({ query: { ...route.query, account_id: String(accountID), trace: '1' } })

  const currentRecovery = overview.value?.accounts.find((item) => item.account_id === accountID)
  const results = await Promise.allSettled([
    adminAPI.accounts.getById(accountID),
    adminAPI.sharedPool.listLedgerEntries({ page: 1, page_size: 20, account_id: accountID }),
    fetchTraceSettlementPage(accountID, 1),
    currentRecovery ? Promise.resolve(currentRecovery) : adminAPI.sharedPool.getOverview({ ...periodParams(), account_id: accountID }).then((item) => item.accounts.find((account) => account.account_id === accountID) || null),
    adminAPI.sharedPool.listLifecycle(accountID),
    adminAPI.sharedPool.listApprovals({ account_id: accountID, page: 1, page_size: 20 })
  ])
  if (traceAccountId.value !== accountID) return
  const [accountResult, entriesResult, settlementResult, recoveryResult, lifecycleResult, approvalResult] = results
  if (accountResult.status === 'fulfilled') traceAccount.value = accountResult.value
  if (entriesResult.status === 'fulfilled') {
    traceEntries.value = entriesResult.value.items || []
    traceEntryPage.value = entriesResult.value.page || 1
    traceEntryTotal.value = entriesResult.value.total || 0
  }
  if (settlementResult.status === 'fulfilled') {
    traceSettlement.value = settlementResult.value.settlement
    traceSettlementPage.value = settlementResult.value.page
    traceSettlementTotal.value = settlementResult.value.total
  }
  if (recoveryResult.status === 'fulfilled') traceRecovery.value = recoveryResult.value
  if (lifecycleResult.status === 'fulfilled') traceLifecycle.value = lifecycleResult.value
  if (approvalResult.status === 'fulfilled') {
    traceApprovals.value = approvalResult.value.items || []
    traceApprovalPage.value = approvalResult.value.page || 1
    traceApprovalTotal.value = approvalResult.value.total || 0
  }
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
  if (ledgerEntryID) {
    const query = { ...route.query }
    delete query.ledger_entry_id
    await router.replace({ query })
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
  if (costPaidAtFuture.value) {
    appStore.showError(t('admin.sharedPool.errors.futurePaidAt'))
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
		paid_at: payload.paid_at,
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

const syncSettlementFiltersFromRoute = () => {
  const next = {
    account_id: routeQueryID('account_id') || '',
    uploader_user_id: routeQueryID('uploader_user_id') || '',
    payer_user_id: routeQueryID('payer_user_id') || '',
    purchase_source_id: routeQueryID('purchase_source_id') || '',
    line_status: (['all', 'pending', 'paid', 'abnormal'].includes(queryString('settlement_line_status'))
      ? queryString('settlement_line_status')
      : 'all') as SettlementLineFilter
  }
  const changed = Object.entries(next).some(([key, value]) =>
    String(settlementFilters[key as keyof typeof settlementFilters]) !== String(value)
  )
  if (changed) Object.assign(settlementFilters, next)
  return changed
}

watch(
  () => route.query.tab,
  () => {
    const tab = routeTab()
    if (tab === activeTab.value) return
    activeTab.value = tab
    if (tab === 'settlement') syncSettlementFiltersFromRoute()
    if (tab !== 'ledger') void loadActiveTab()
  }
)

watch(
  () => [
    route.query.account_id,
    route.query.uploader_user_id,
    route.query.payer_user_id,
    route.query.purchase_source_id,
    route.query.settlement_line_status
  ],
  (next, previous) => {
    if (activeTab.value === 'settlement') {
      if (syncSettlementFiltersFromRoute()) void loadActiveTab()
      return
    }
    if ((activeTab.value === 'overview' || activeTab.value === 'sources') && String(next[0] || '') !== String(previous?.[0] || '')) {
      void loadActiveTab()
    }
  }
)

watch(
  () => [
    route.query.overview_page, route.query.overview_page_size, route.query.overview_recovery,
    route.query.source_page, route.query.source_page_size, route.query.uploader_user_id,
    route.query.settlement_page, route.query.settlement_page_size
  ],
  () => {
    overviewPagination.page = queryPage('overview_page', 1)
    overviewPagination.page_size = queryPage('overview_page_size', 10)
    overviewRecoveryFilter.value = ['unrecovered', 'recovered', 'soon', 'no_data'].includes(queryString('overview_recovery'))
      ? queryString('overview_recovery') as RecoveryFilter
      : 'all'
    sourcePagination.page = queryPage('source_page', 1)
    sourcePagination.page_size = queryPage('source_page_size', 8)
    sourceUploaderFilter.value = routeQueryID('uploader_user_id') || ''
    settlementAccountPagination.page = queryPage('settlement_page', 1)
    settlementAccountPagination.page_size = queryPage('settlement_page_size', 10)
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

<style scoped>
.shared-pool-shell :deep(.date-picker-trigger) {
  min-height: 2.75rem;
}
</style>
