<template>
  <section class="card min-w-0 overflow-hidden">
    <div class="flex flex-col gap-3 border-b border-gray-100 p-4 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.sharedPool.ledger.title') }}</h2>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.subtitle') }}</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="inline-flex rounded-md border border-gray-200 p-0.5 dark:border-dark-600" role="tablist">
          <button
            v-for="view in ledgerViews"
            :key="view.value"
            type="button"
            class="min-h-11 px-3 text-sm font-medium"
            :class="activeView === view.value ? 'rounded bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300' : 'text-gray-500 dark:text-gray-400'"
            role="tab"
            :aria-selected="activeView === view.value"
            @click="switchView(view.value)"
          >
            {{ view.label }}
          </button>
        </div>
        <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="loading" @click="loadActiveView">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('common.refresh') }}
        </button>
        <button type="button" class="btn btn-primary min-h-11 px-3" @click="openBatchDialog">
          <Icon name="plus" size="sm" />
          {{ t('admin.sharedPool.ledger.batchAdd') }}
        </button>
      </div>
    </div>

    <div v-if="activeView === 'summary'" class="min-w-0">
      <div class="grid grid-cols-1 gap-3 border-b border-gray-100 p-4 dark:border-dark-700 sm:grid-cols-2 xl:grid-cols-8">
        <SearchInput
          v-model="summaryFilters.search"
          class="xl:col-span-2"
          :placeholder="t('admin.sharedPool.ledger.searchPlaceholder')"
          @search="applySummaryFilters"
        />
        <Select v-model="summaryFilters.uploader_user_id" :options="uploaderFilterOptions" searchable :aria-label="t('admin.sharedPool.columns.uploader')" @change="applySummaryFilters" />
        <Select v-model="summaryFilters.payer_user_id" :options="payerFilterOptions" searchable :aria-label="t('admin.sharedPool.ledger.payer')" @change="applySummaryFilters" />
        <Select v-model="summaryFilters.purchase_source_id" :options="sourceFilterOptions" searchable :aria-label="t('admin.sharedPool.columns.source')" @change="applySummaryFilters" />
        <Select v-model="summaryFilters.availability_status" :options="availabilityFilterOptions" :aria-label="t('admin.accounts.columns.status')" @change="applySummaryFilters" />
        <Select v-model="summaryFilters.lifecycle_status" :options="statusFilterOptions" :aria-label="t('admin.sharedPool.columns.status')" @change="applySummaryFilters" />
        <Select v-model="summaryFilters.has_cost" :options="hasCostFilterOptions" :aria-label="t('admin.sharedPool.ledger.costState')" @change="applySummaryFilters" />
      </div>

      <div class="space-y-3 p-3 sm:p-4">
        <article v-for="group in uploaderSummaries" :key="uploaderGroupKey(group)" class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
          <button type="button" class="flex min-h-14 w-full items-center justify-between gap-3 px-3 py-2 text-left sm:px-4" @click="toggleUploaderGroup(group)">
            <span class="min-w-0"><span class="block truncate font-semibold text-gray-900 dark:text-white">{{ uploaderGroupName(group) }}</span><span class="block text-xs text-gray-500 dark:text-gray-400">{{ group.account_count }} {{ t('admin.sharedPool.columns.accounts') }}</span></span>
            <span class="shrink-0 text-xs font-medium text-primary-600 dark:text-primary-400">{{ expandedUploaderGroups.has(uploaderGroupKey(group)) ? t('common.collapse') : t('common.expand') }}</span>
          </button>
          <dl class="grid grid-cols-3 gap-2 border-t border-gray-100 bg-gray-50 px-3 py-2 text-xs dark:border-dark-700 dark:bg-dark-800/60 sm:px-4">
            <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.netCost') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ formatMinor(group.net_cost_minor) }}</dd></div>
            <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.recognizedCost') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ formatMinor(group.recognized_cost_minor) }}</dd></div>
            <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.remaining') }}</dt><dd class="mt-1 font-medium tabular-nums text-amber-600 dark:text-amber-400">{{ formatMinor(group.remaining_cost_minor) }}</dd></div>
          </dl>
          <div v-if="expandedUploaderGroups.has(uploaderGroupKey(group))" class="divide-y divide-gray-100 border-t border-gray-100 dark:divide-dark-700 dark:border-dark-700">
            <div v-if="uploaderAccountStates[uploaderGroupKey(group)]?.loading" class="flex min-h-24 items-center justify-center"><LoadingSpinner /></div>
            <article v-for="row in uploaderAccountStates[uploaderGroupKey(group)]?.items || []" :key="row.account_id" class="p-3 sm:p-4">
              <div class="flex min-w-0 items-start justify-between gap-3">
                <div class="min-w-0"><button type="button" class="block min-h-11 max-w-full truncate text-left text-sm font-semibold text-primary-600 hover:underline dark:text-primary-400" :title="row.account_name" @click="emit('trace-account', row.account_id)">{{ row.account_name }}</button><p class="truncate text-xs text-gray-500 dark:text-gray-400" :title="row.provider_identity || ''">{{ row.provider_identity || '-' }}</p></div>
                <div class="flex shrink-0 flex-col items-end gap-1">
                  <StatusBadge :status="availabilityPresentation(row.availability_status, row.account_status).badge" :label="t(`admin.accounts.status.${availabilityPresentation(row.availability_status, row.account_status).key}`)" />
                  <span class="text-xs text-gray-500 dark:text-gray-400">{{ t(`admin.sharedPool.status.${statusPresentation(row.latest_lifecycle_status).key}`) }}</span>
                </div>
              </div>
              <dl class="mt-3 grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.netCost') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ formatMinor(row.net_cost_minor) }}</dd></div>
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.usageExpected') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ formatTokens(row.total_usage_tokens) }} / {{ formatTokens(row.expected_token_count || 0) }}</dd></div>
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.recognizedCost') }}</dt><dd class="mt-1 font-medium tabular-nums">{{ formatMinor(row.recognized_cost_minor) }}</dd></div>
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.remaining') }}</dt><dd class="mt-1 font-medium tabular-nums text-amber-600 dark:text-amber-400">{{ formatMinor(row.remaining_cost_minor) }}</dd></div>
              </dl>
              <dl class="mt-3 grid grid-cols-2 gap-2 border-t border-gray-100 pt-2 text-xs dark:border-dark-700 sm:grid-cols-4">
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.payer') }}</dt><dd class="mt-1 truncate font-medium" :title="row.latest_payer_email || ''">{{ row.latest_payer_email || '-' }}</dd></div>
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.columns.source') }}</dt><dd class="mt-1 truncate font-medium" :title="row.latest_purchase_source || ''">{{ row.latest_purchase_source || '-' }}</dd></div>
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.page.purchaseDate') }}</dt><dd class="mt-1 font-medium">{{ dateOnly(row.purchased_at) }}</dd></div>
                <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.page.deadline') }}</dt><dd class="mt-1 font-medium">{{ dateOnly(row.latest_service_end) }}</dd></div>
              </dl>
              <div class="mt-3 grid grid-cols-2 border-t border-gray-100 pt-2 dark:border-dark-700">
                <button type="button" class="min-h-11 text-sm font-medium text-primary-600 dark:text-primary-400" @click="emit('open-account', row.account_id)">{{ t('admin.sharedPool.actions.poolRecord') }}</button>
                <button type="button" class="min-h-11 border-l border-gray-100 text-sm font-medium text-primary-600 dark:border-dark-700 dark:text-primary-400" @click="showAccountEntries(row.account_id)">{{ t('admin.sharedPool.ledger.viewEntries') }}</button>
              </div>
            </article>
            <Pagination
              v-if="(uploaderAccountStates[uploaderGroupKey(group)]?.total || 0) > (uploaderAccountStates[uploaderGroupKey(group)]?.pageSize || 10)"
              :page="uploaderAccountStates[uploaderGroupKey(group)]?.page || 1"
              :page-size="uploaderAccountStates[uploaderGroupKey(group)]?.pageSize || 10"
              :show-page-size-selector="false"
              :total="uploaderAccountStates[uploaderGroupKey(group)]?.total || 0"
              @update:page="loadUploaderAccounts(group, $event)"
            />
          </div>
        </article>
        <EmptyState v-if="!loading && !uploaderSummaries.length" :title="t('admin.sharedPool.ledger.emptySummary')" />
      </div>
      <Pagination v-if="summaryPagination.total" :page="summaryPagination.page" :page-size="summaryPagination.page_size" :total="summaryPagination.total" @update:page="changeSummaryPage" @update:page-size="changeSummaryPageSize" />
    </div>

    <div v-else class="min-w-0">
      <div class="grid grid-cols-1 gap-3 border-b border-gray-100 p-4 dark:border-dark-700 sm:grid-cols-2 xl:grid-cols-6">
        <SearchInput v-model="entryFilters.search" class="xl:col-span-2" :placeholder="t('admin.sharedPool.ledger.searchEntries')" @search="applyEntryFilters" />
        <Select v-model="entryFilters.account_id" :options="accountFilterOptions" searchable :aria-label="t('admin.sharedPool.form.account')" @change="applyEntryFilters" />
        <Select v-model="entryFilters.uploader_user_id" :options="uploaderFilterOptions" searchable :aria-label="t('admin.sharedPool.columns.uploader')" @change="applyEntryFilters" />
        <Select v-model="entryFilters.purchase_source_id" :options="sourceFilterOptions" searchable :aria-label="t('admin.sharedPool.columns.source')" @change="applyEntryFilters" />
        <Select v-model="entryFilters.entry_type" :options="entryTypeFilterOptions" :aria-label="t('admin.sharedPool.columns.costType')" @change="applyEntryFilters" />
        <Select v-model="entryFilters.payer_user_id" :options="payerFilterOptions" searchable :aria-label="t('admin.sharedPool.ledger.payer')" @change="applyEntryFilters" />
        <div><label for="ledger-entry-start" class="input-label">{{ t('admin.sharedPool.ledger.startDate') }}</label><input id="ledger-entry-start" v-model="entryFilters.start_date" class="input" type="date" @change="applyEntryFilters" /></div>
        <div><label for="ledger-entry-end" class="input-label">{{ t('admin.sharedPool.ledger.endDate') }}</label><input id="ledger-entry-end" v-model="entryFilters.end_date" class="input" type="date" :min="entryFilters.start_date" @change="applyEntryFilters" /></div>
      </div>
      <div class="hidden md:block">
        <DataTable :columns="entryColumns" :data="entries" row-key="id" :loading="loading">
          <template #cell-account_name="{ row }"><button type="button" class="min-h-11 font-medium text-primary-600 hover:underline dark:text-primary-400" @click="emit('trace-account', row.account_id)">{{ row.account_name }}</button></template>
          <template #cell-entry_type="{ row }">{{ t(`admin.sharedPool.entryTypes.${row.entry_type}`) }}</template>
          <template #cell-availability_status="{ row }"><StatusBadge :status="availabilityPresentation(row.availability_status, row.account_status).badge" :label="t(`admin.accounts.status.${availabilityPresentation(row.availability_status, row.account_status).key}`)" /></template>
          <template #cell-original_amount="{ row }"><span class="tabular-nums">{{ formatMoney(Number(row.original_amount), row.currency) }}</span></template>
          <template #cell-cny_amount_minor="{ row }"><span class="block tabular-nums">{{ formatMinor(row.cny_amount_minor) }}</span><span class="block whitespace-nowrap text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.page.fxRateAt', { currency: row.currency, rate: row.fx_rate, date: dateOnly(row.paid_at) }) }}</span></template>
          <template #cell-service_start="{ row }"><span class="whitespace-nowrap">{{ dateOnly(row.service_start) }} - {{ dateOnly(row.service_end) }}</span></template>
          <template #cell-created_at="{ row }">{{ dateOnly(row.created_at || row.paid_at) }}</template>
		  <template #cell-actions="{ row }">
			<button v-if="isEditableEntry(row)" type="button" class="btn btn-secondary btn-sm min-h-11" @click="emit('edit-entry', row)">{{ t('common.edit') }}</button>
		  </template>
        </DataTable>
      </div>
      <div class="space-y-3 p-3 md:hidden">
        <article v-for="row in entries" :key="row.id" class="rounded-md border border-gray-200 p-3 text-sm dark:border-dark-700">
          <div class="flex min-w-0 items-start justify-between gap-3"><button type="button" class="min-h-11 min-w-0 truncate text-left font-semibold text-primary-600 dark:text-primary-400" :title="row.account_name" @click="emit('trace-account', row.account_id)">{{ row.account_name }}</button><span class="shrink-0 font-medium tabular-nums">{{ formatMoney(Number(row.original_amount), row.currency) }}</span></div>
          <StatusBadge class="mt-2" :status="availabilityPresentation(row.availability_status, row.account_status).badge" :label="t(`admin.accounts.status.${availabilityPresentation(row.availability_status, row.account_status).key}`)" />
          <div class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t(`admin.sharedPool.entryTypes.${row.entry_type}`) }}</span><span>{{ row.purchase_source || '-' }}</span><span>{{ row.payer_email || '-' }}</span><span>{{ dateOnly(row.paid_at) }}</span>
          </div>
		  <button v-if="isEditableEntry(row)" type="button" class="mt-3 min-h-11 w-full border-t border-gray-100 pt-2 text-sm font-medium text-primary-600 dark:border-dark-700 dark:text-primary-400" @click="emit('edit-entry', row)">{{ t('common.edit') }}</button>
        </article>
        <EmptyState v-if="!loading && !entries.length" :title="t('admin.sharedPool.ledger.emptyEntries')" />
      </div>
      <Pagination v-if="entryPagination.total" :page="entryPagination.page" :page-size="entryPagination.page_size" :total="entryPagination.total" @update:page="changeEntryPage" @update:page-size="changeEntryPageSize" />
    </div>
  </section>

  <BaseDialog :show="showBatchDialog" :title="t('admin.sharedPool.ledger.batchTitle')" width="extra-wide" @close="closeBatchDialog">
    <div class="mb-5 grid grid-cols-4 gap-1" aria-label="Batch cost steps">
      <div v-for="(label, index) in stepLabels" :key="label" class="min-w-0">
        <div class="h-1 rounded-full" :class="batchStep >= index + 1 ? 'bg-primary-500' : 'bg-gray-200 dark:bg-dark-600'"></div>
        <p class="mt-1 truncate text-center text-xs" :class="batchStep === index + 1 ? 'font-medium text-primary-600 dark:text-primary-400' : 'text-gray-500 dark:text-gray-400'">{{ label }}</p>
      </div>
    </div>

    <div v-if="batchErrors.length" class="mb-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300" role="alert" tabindex="-1">
      <p v-for="error in batchErrors" :key="error">{{ batchErrorLabel(error) }}</p>
    </div>

    <div v-if="batchStep === 1" class="space-y-4">
      <SearchInput v-model="accountSearch" :placeholder="t('admin.sharedPool.ledger.searchAccounts')" @search="loadBatchAccounts" />
      <div class="flex items-center justify-between text-sm">
        <span>{{ t('admin.sharedPool.ledger.selectedCount', { count: selectedAccounts.length }) }}</span>
        <button type="button" class="min-h-11 text-primary-600 dark:text-primary-400" @click="toggleAllVisible">{{ allVisibleSelected ? t('admin.sharedPool.ledger.clearVisible') : t('admin.sharedPool.ledger.selectVisible') }}</button>
      </div>
      <div class="max-h-80 space-y-1 overflow-y-auto rounded-md border border-gray-200 p-2 dark:border-dark-700">
        <label v-for="account in batchAccounts" :key="account.id" class="flex min-h-11 cursor-pointer items-center gap-3 rounded px-2 hover:bg-gray-50 dark:hover:bg-dark-700">
          <input type="checkbox" class="h-5 w-5 rounded border-gray-300 text-primary-600 focus:ring-primary-500" :checked="isAccountSelected(account.id)" @change="toggleAccount(account)" />
          <span class="min-w-0"><span class="block truncate text-sm font-medium">{{ account.name }}</span><span class="block truncate text-xs text-gray-500 dark:text-gray-400">#{{ account.id }}</span></span>
        </label>
      </div>
    </div>

    <form v-else-if="batchStep === 2" class="grid grid-cols-1 gap-4 md:grid-cols-2" @submit.prevent="nextBatchStep">
      <div class="md:col-span-2">
        <span class="input-label">{{ t('admin.sharedPool.ledger.amountMode') }}</span>
        <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <button v-for="mode in amountModeOptions" :key="mode.value" type="button" class="min-h-11 rounded-md border px-3 text-left text-sm" :class="batchForm.amount_mode === mode.value ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 dark:border-dark-600'" :aria-pressed="batchForm.amount_mode === mode.value" @click="setAmountMode(mode.value)">
            <span class="block font-medium">{{ mode.label }}</span><span class="mt-0.5 block text-xs opacity-75">{{ mode.hint }}</span>
          </button>
        </div>
      </div>
      <div v-if="batchForm.amount_mode === 'order_total'" class="md:col-span-2">
        <span class="input-label">{{ t('admin.sharedPool.ledger.allocationMode') }}</span>
        <div class="inline-flex rounded-md border border-gray-200 p-0.5 dark:border-dark-600">
          <button v-for="mode in allocationModeOptions" :key="mode.value" type="button" class="min-h-11 px-4 text-sm" :class="batchForm.allocation_mode === mode.value ? 'rounded bg-primary-50 font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-300' : ''" @click="batchForm.allocation_mode = mode.value">{{ mode.label }}</button>
        </div>
      </div>
      <div><label for="ledger-payer" class="input-label">{{ t('admin.sharedPool.ledger.payer') }} *</label><Select id="ledger-payer" v-model="batchForm.payer_user_id" :options="userOptions" searchable /></div>
      <div><label for="ledger-source" class="input-label">{{ t('admin.sharedPool.columns.source') }} *</label><Select id="ledger-source" v-model="batchForm.purchase_source_id" :options="sourceOptions" searchable /></div>
      <div><label for="ledger-entry-type" class="input-label">{{ t('admin.sharedPool.form.entryType') }} *</label><Select id="ledger-entry-type" v-model="batchForm.entry_type" :options="entryTypeOptions" /></div>
      <div><label for="ledger-amount" class="input-label">{{ batchForm.amount_mode === 'per_account' ? t('admin.sharedPool.ledger.perAccountAmount') : t('admin.sharedPool.ledger.orderTotal') }} *</label><input id="ledger-amount" v-model.number="batchForm.amount" class="input" type="number" inputmode="decimal" min="0.01" step="0.01" required /></div>
      <div><label for="ledger-expected" class="input-label">{{ t('admin.sharedPool.ledger.expectedTokens') }} *</label><input id="ledger-expected" v-model.number="batchExpectedTokenMillions" class="input" type="number" inputmode="decimal" min="0.1" step="0.1" required /></div>
      <div><label for="ledger-currency" class="input-label">{{ t('admin.sharedPool.form.currency') }}</label><Select id="ledger-currency" v-model="batchForm.currency" :options="currencyOptions" /></div>
      <div><label for="ledger-service-start" class="input-label">{{ t('admin.sharedPool.form.serviceStart') }} *</label><input id="ledger-service-start" v-model="batchForm.service_start" class="input" type="date" required /></div>
      <div><label for="ledger-service-end" class="input-label">{{ t('admin.sharedPool.form.serviceEnd') }} *</label><input id="ledger-service-end" v-model="batchForm.service_end" class="input" type="date" :min="batchForm.service_start" required /></div>
      <div><label for="ledger-warranty" class="input-label">{{ t('admin.sharedPool.form.warrantyEnd') }}</label><input id="ledger-warranty" v-model="batchForm.warranty_end" class="input" type="date" /></div>
      <div><label for="ledger-order" class="input-label">{{ t('admin.sharedPool.form.orderNo') }}</label><input id="ledger-order" v-model.trim="batchForm.order_no" class="input" /></div>
      <div><label for="ledger-paid" class="input-label">{{ t('admin.sharedPool.ledger.paidAt') }}</label><input id="ledger-paid" v-model="batchForm.paid_at" class="input" type="datetime-local" :aria-invalid="batchPaidAtFuture" aria-describedby="ledger-paid-hint" /><p id="ledger-paid-hint" class="input-hint" :class="batchPaidAtFuture ? 'text-red-600 dark:text-red-400' : ''">{{ t(batchPaidAtFuture ? 'admin.sharedPool.ledger.errors.future_paid_at' : 'admin.sharedPool.ledger.paidAtHint') }}</p></div>
      <div class="md:col-span-2 rounded-md bg-gray-50 p-3 text-sm dark:bg-dark-700/60">
        {{ amountPreviewText }}
      </div>
      <div class="md:col-span-2"><label for="ledger-url" class="input-label">{{ t('admin.sharedPool.form.purchaseUrl') }}</label><input id="ledger-url" v-model.trim="batchForm.purchase_url" class="input" type="url" /></div>
      <div class="md:col-span-2"><label for="ledger-notes" class="input-label">{{ t('admin.sharedPool.form.notes') }}</label><textarea id="ledger-notes" v-model.trim="batchForm.notes" class="input min-h-20" rows="3"></textarea></div>
    </form>

    <div v-else-if="batchStep === 3" class="space-y-3">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.overrideHint') }}</p>
      <div class="space-y-2">
        <div v-for="account in selectedAccounts" :key="account.id" class="grid grid-cols-1 gap-3 rounded-md border border-gray-200 p-3 dark:border-dark-700 md:grid-cols-[minmax(0,1fr)_180px_220px] md:items-end">
          <div class="min-w-0"><p class="truncate text-sm font-medium">{{ account.name }}</p><p class="text-xs text-gray-500 dark:text-gray-400">#{{ account.id }}</p></div>
          <div><label :for="`ledger-override-amount-${account.id}`" class="input-label">{{ t('admin.sharedPool.ledger.accountAmount') }}</label><input :id="`ledger-override-amount-${account.id}`" v-model.number="overrides[account.id].originalAmount" class="input" type="number" inputmode="decimal" min="0.01" step="0.01" :placeholder="overrideAmountPlaceholder(account.id)" /></div>
          <div><label :for="`ledger-override-token-${account.id}`" class="input-label">{{ t('admin.sharedPool.ledger.expectedTokens') }}</label><input :id="`ledger-override-token-${account.id}`" :value="overrideTokenMillions(account.id)" class="input" type="number" inputmode="decimal" min="0.1" step="0.1" :placeholder="String(batchExpectedTokenMillions || '')" @input="setOverrideTokenMillions(account.id, $event)" /></div>
        </div>
      </div>
    </div>

    <div v-else class="space-y-4">
      <div class="grid grid-cols-2 gap-3 rounded-md bg-gray-50 p-4 text-sm dark:bg-dark-700/60 sm:grid-cols-4">
        <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.accountCount') }}</p><p class="mt-1 font-semibold tabular-nums">{{ selectedAccounts.length }}</p></div>
        <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.amountMode') }}</p><p class="mt-1 font-semibold">{{ amountModeOptions.find((item) => item.value === batchForm.amount_mode)?.label }}</p></div>
        <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.orderTotal') }}</p><p class="mt-1 font-semibold tabular-nums">{{ formatMoney(allocationResult.totalAmount, batchForm.currency) }}</p></div>
        <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.payer') }}</p><p class="mt-1 truncate font-semibold">{{ selectedOptionLabel(userOptions, batchForm.payer_user_id) }}</p></div>
      </div>
      <div class="max-h-80 overflow-y-auto rounded-md border border-gray-200 dark:border-dark-700">
        <div v-for="allocation in allocationResult.allocations" :key="allocation.accountId" class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 border-b border-gray-100 px-3 py-2 text-sm last:border-b-0 dark:border-dark-700">
          <span class="truncate">{{ selectedAccounts.find((item) => item.id === allocation.accountId)?.name }}</span>
          <span class="text-right tabular-nums">{{ formatMoney(allocation.originalAmount, batchForm.currency) }} · {{ formatTokens(allocation.expectedTokenCount) }}</span>
        </div>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.duplicateHint') }}</p>
    </div>

    <template #footer>
      <div class="flex w-full flex-wrap justify-between gap-2">
        <button type="button" class="btn btn-secondary min-h-11" :disabled="submitting" @click="batchStep === 1 ? closeBatchDialog() : previousBatchStep()">{{ batchStep === 1 ? t('common.cancel') : t('admin.sharedPool.ledger.previous') }}</button>
        <button v-if="batchStep < 4" type="button" class="btn btn-primary min-h-11" @click="nextBatchStep">{{ t('admin.sharedPool.ledger.next') }}</button>
        <button v-else type="button" class="btn btn-primary min-h-11" :disabled="submitting" @click="submitBatch">
          <LoadingSpinner v-if="submitting" size="sm" /><Icon v-else name="check" size="sm" />{{ t('admin.sharedPool.ledger.submitBatch') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'
import type {
  BatchSharedPoolCostRequest,
  PoolAvailabilityStatus,
  PoolCostEntryType,
  SharedPoolBatchAmountMode,
  SharedPoolCostSummary,
  SharedPoolCostUploaderSummary,
  SharedPoolLedgerEntry,
  SharedPoolPurchaseSource
} from '@/api/admin/sharedPool'
import { BaseDialog, DataTable, EmptyState, LoadingSpinner, Pagination } from '@/components/common'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores'
import { accountStatusPresentation, formatPoolMoney, isPoolPaidAtFuture, poolPaidAtToISOString } from '@/utils/sharedPool'
import {
  calculateBatchCostAllocations,
  DEFAULT_EXPECTED_TOKEN_COUNT,
  millionsToTokens,
  tokensToMillions,
  type BatchAllocationMode,
  type BatchCostOverride
} from '@/utils/sharedPoolLedger'
import { formatDateLocalInput } from '@/utils/format'

type LedgerView = 'summary' | 'entries'
type AccountChoice = Pick<Account, 'id' | 'name'>
type UploaderAccountState = { items: SharedPoolCostSummary[]; page: number; pageSize: number; total: number; loading: boolean }
const props = withDefaults(defineProps<{
  initialAccountId?: number
  initialPurchaseSourceId?: number
  initialUploaderUserId?: number
}>(), {
  initialAccountId: 0,
  initialPurchaseSourceId: 0,
  initialUploaderUserId: 0
})
const emit = defineEmits<{
  (event: 'edit-entry', entry: SharedPoolLedgerEntry): void
  (event: 'open-account', accountId: number): void
  (event: 'trace-account', accountId: number): void
}>()

const { t, te, locale } = useI18n()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()
const queryText = (key: string) => {
  const value = route.query[key]
  return String(Array.isArray(value) ? value[0] || '' : value || '')
}
const queryID = (key: string) => {
  const value = Number(queryText(key))
  return Number.isSafeInteger(value) && value > 0 ? value : 0
}
const queryPage = (key: string, fallback: number) => queryID(key) || fallback
const activeView = ref<LedgerView>(queryText('ledger_view') === 'entries' ? 'entries' : 'summary')
const loading = ref(false)
const submitting = ref(false)
const uploaderSummaries = ref<SharedPoolCostUploaderSummary[]>([])
const entries = ref<SharedPoolLedgerEntry[]>([])
const users = ref<Array<{ id: number; label: string; active: boolean }>>([])
const sources = ref<SharedPoolPurchaseSource[]>([])
const referencedSources = ref<SharedPoolPurchaseSource[]>([])
const accountChoices = ref<AccountChoice[]>([])
const batchAccounts = ref<AccountChoice[]>([])
const selectedAccounts = ref<AccountChoice[]>([])
const expandedUploaderGroups = ref(new Set<string>())
const uploaderAccountStates = reactive<Record<string, UploaderAccountState>>({})
const accountSearch = ref('')
const showBatchDialog = ref(false)
const batchStep = ref(1)
const batchErrors = ref<string[]>([])
const overrides = reactive<Record<number, BatchCostOverride>>({})

const summaryPagination = reactive({ page: queryPage('ledger_summary_page', 1), page_size: queryPage('ledger_summary_page_size', 20), total: 0 })
const entryPagination = reactive({ page: queryPage('ledger_entry_page', 1), page_size: queryPage('ledger_entry_page_size', 20), total: 0 })
const summaryFilters = reactive({
  search: queryText('ledger_summary_search'),
  uploader_user_id: queryID('uploader_user_id') || props.initialUploaderUserId,
  payer_user_id: queryID('payer_user_id'),
  purchase_source_id: queryID('purchase_source_id') || props.initialPurchaseSourceId,
  availability_status: queryText('ledger_availability_status') as PoolAvailabilityStatus | '',
  lifecycle_status: queryText('ledger_lifecycle_status'),
  has_cost: queryText('ledger_has_cost')
})
const entryFilters = reactive<{ search: string; account_id: number; uploader_user_id: number; payer_user_id: number; purchase_source_id: number; entry_type: PoolCostEntryType | ''; start_date: string; end_date: string }>({
  search: queryText('ledger_entry_search'),
  account_id: queryID('account_id') || props.initialAccountId,
  uploader_user_id: queryID('uploader_user_id') || props.initialUploaderUserId,
  payer_user_id: queryID('payer_user_id'),
  purchase_source_id: queryID('purchase_source_id') || props.initialPurchaseSourceId,
  entry_type: queryText('ledger_entry_type') as PoolCostEntryType | '',
  start_date: queryText('ledger_entry_start'),
  end_date: queryText('ledger_entry_end')
})

const today = () => formatDateLocalInput(new Date())
const nextMonth = () => {
  const date = new Date()
  date.setMonth(date.getMonth() + 1)
  return formatDateLocalInput(date)
}
const emptyBatchForm = () => ({
  amount_mode: 'per_account' as SharedPoolBatchAmountMode,
  allocation_mode: 'equal' as BatchAllocationMode,
  payer_user_id: 0,
  purchase_source_id: 0,
  entry_type: 'purchase' as PoolCostEntryType,
  amount: 0,
  expected_token_count: DEFAULT_EXPECTED_TOKEN_COUNT,
  currency: 'CNY',
  service_start: today(),
  service_end: nextMonth(),
  warranty_end: '',
  paid_at: '',
  order_no: '',
  purchase_url: '',
  notes: ''
})
const batchForm = reactive(emptyBatchForm())
const batchPaidAtFuture = computed(() => isPoolPaidAtFuture(batchForm.paid_at))
const batchExpectedTokenMillions = computed({
  get: () => tokensToMillions(batchForm.expected_token_count),
  set: (value: number) => { batchForm.expected_token_count = millionsToTokens(Number(value)) }
})

const ledgerViews = computed(() => [
  { value: 'summary' as const, label: t('admin.sharedPool.ledger.summaryView') },
  { value: 'entries' as const, label: t('admin.sharedPool.ledger.entriesView') }
])
const stepLabels = computed(() => [t('admin.sharedPool.ledger.steps.accounts'), t('admin.sharedPool.ledger.steps.common'), t('admin.sharedPool.ledger.steps.overrides'), t('admin.sharedPool.ledger.steps.preview')])
const amountModeOptions = computed(() => [
  { value: 'per_account' as const, label: t('admin.sharedPool.ledger.perAccountAmount'), hint: t('admin.sharedPool.ledger.perAccountHint') },
  { value: 'order_total' as const, label: t('admin.sharedPool.ledger.orderTotal'), hint: t('admin.sharedPool.ledger.orderTotalHint') }
])
const allocationModeOptions = computed(() => [
  { value: 'equal' as const, label: t('admin.sharedPool.ledger.equalAllocation') },
  { value: 'manual' as const, label: t('admin.sharedPool.ledger.manualAllocation') }
])
const entryTypeOptions = computed(() => ['purchase', 'renewal', 'topup', 'price_version', 'adjustment'].map((value) => ({ value, label: t(`admin.sharedPool.entryTypes.${value}`) })))
const entryTypeFilterOptions = computed(() => [{ value: '', label: t('admin.sharedPool.ledger.allEntryTypes') }, ...['purchase', 'renewal', 'topup', 'price_version', 'refund', 'adjustment', 'replacement_in', 'replacement_out', 'write_off'].map((value) => ({ value, label: t(`admin.sharedPool.entryTypes.${value}`) }))])
const currencyOptions = [{ value: 'CNY', label: 'CNY' }]
const allUserOptions = computed(() => users.value.map((item) => ({ value: item.id, label: item.label })))
const userOptions = computed(() => users.value.filter((item) => item.active).map((item) => ({ value: item.id, label: item.label })))
const uploaderFilterOptions = computed(() => [{ value: 0, label: t('admin.sharedPool.ledger.allUploaders') }, ...allUserOptions.value])
const payerFilterOptions = computed(() => [{ value: 0, label: t('admin.sharedPool.ledger.allPayers') }, ...allUserOptions.value])
const sourceOptions = computed(() => sources.value.filter((item) => item.active).map((item) => ({ value: item.id, label: item.name })))
const sourceFilterOptions = computed(() => [{ value: 0, label: t('admin.sharedPool.ledger.allSources') }, ...referencedSources.value.map((item) => ({ value: item.id, label: item.name }))])
const accountFilterOptions = computed(() => [{ value: 0, label: t('admin.sharedPool.ledger.allAccounts') }, ...accountChoices.value.map((item) => ({ value: item.id, label: item.name }))])
const statusFilterOptions = computed(() => [
  { value: '', label: t('admin.sharedPool.ledger.allStatuses') },
  { value: 'active', label: t('admin.sharedPool.status.active') },
  { value: 'banned_confirmed', label: t('admin.sharedPool.status.banned') },
  { value: 'retired', label: t('admin.sharedPool.status.inactive') }
])
const availabilityFilterOptions = computed(() => [
  { value: '', label: t('admin.sharedPool.ledger.allStatuses') },
  { value: 'normal', label: t('admin.accounts.status.active') },
  { value: 'error', label: t('admin.accounts.status.error') },
  { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') },
  { value: 'overloaded', label: t('admin.accounts.status.overloaded') },
  { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'manual_unschedulable', label: t('admin.accounts.status.unschedulable') }
])
const hasCostFilterOptions = computed(() => [
  { value: '', label: t('admin.sharedPool.ledger.allCostStates') },
  { value: 'true', label: t('admin.sharedPool.ledger.withCost') },
  { value: 'false', label: t('admin.sharedPool.ledger.withoutCost') }
])

const entryColumns = computed<Column[]>(() => [
  { key: 'account_name', label: t('admin.sharedPool.columns.account') },
  { key: 'entry_type', label: t('admin.sharedPool.columns.costType') },
  { key: 'availability_status', label: t('admin.accounts.columns.status') },
  { key: 'payer_email', label: t('admin.sharedPool.ledger.payer') },
  { key: 'purchase_source', label: t('admin.sharedPool.columns.source') },
  { key: 'original_amount', label: t('admin.sharedPool.ledger.originalAmount') },
  { key: 'cny_amount_minor', label: t('admin.sharedPool.ledger.cnyAmount') },
  { key: 'service_start', label: t('admin.sharedPool.columns.servicePeriod') },
  { key: 'order_no', label: t('admin.sharedPool.form.orderNo') },
  { key: 'created_at', label: t('admin.sharedPool.ledger.recordedAt') },
	{ key: 'actions', label: t('admin.sharedPool.columns.actions') }
])

const uploaderGroupKey = (group: SharedPoolCostUploaderSummary) => group.uploader_user_id ? String(group.uploader_user_id) : 'unassigned'
const uploaderGroupName = (group: SharedPoolCostUploaderSummary) => group.uploader_username || group.uploader_email || t('admin.sharedPool.ledger.unassignedUploader')
const uploaderGroupState = (group: SharedPoolCostUploaderSummary) => {
  const key = uploaderGroupKey(group)
  return uploaderAccountStates[key] ||= { items: [], page: 1, pageSize: 10, total: 0, loading: false }
}
async function toggleUploaderGroup(group: SharedPoolCostUploaderSummary) {
  const key = uploaderGroupKey(group)
  const next = new Set(expandedUploaderGroups.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
    if (!uploaderGroupState(group).items.length) void loadUploaderAccounts(group, 1)
  }
  expandedUploaderGroups.value = next
}

const isEditableEntry = (entry: SharedPoolLedgerEntry) => ['purchase', 'renewal', 'topup', 'price_version', 'adjustment'].includes(entry.entry_type)

const allocationResult = computed(() => calculateBatchCostAllocations({
  accountIds: selectedAccounts.value.map((item) => item.id),
  amountMode: batchForm.amount_mode,
  allocationMode: batchForm.allocation_mode,
  commonAmount: Number(batchForm.amount),
  commonExpectedTokenCount: Number(batchForm.expected_token_count),
  overrides
}))
const amountPreviewText = computed(() => batchForm.amount_mode === 'per_account'
  ? t('admin.sharedPool.ledger.perAccountPreview', { price: formatMoney(batchForm.amount, batchForm.currency), count: selectedAccounts.value.length, total: formatMoney(batchForm.amount * selectedAccounts.value.length, batchForm.currency) })
  : t('admin.sharedPool.ledger.orderTotalPreview', { total: formatMoney(batchForm.amount, batchForm.currency), count: selectedAccounts.value.length }))
const allVisibleSelected = computed(() => batchAccounts.value.length > 0 && batchAccounts.value.every((account) => isAccountSelected(account.id)))

const formatMoney = (amount: number, currency = 'CNY') => formatPoolMoney(Number.isFinite(amount) ? amount : 0, currency, locale.value)
const formatMinor = (minor: number) => formatMoney((minor || 0) / 100)
const formatTokens = (tokens: number) => new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(tokens || 0)
const dateOnly = (value?: string | null) => value ? value.slice(0, 10) : '-'
const statusPresentation = (status: string) => accountStatusPresentation(status === 'banned_confirmed' ? 'banned' : status === 'retired' || status === 'replaced' ? 'inactive' : status === 'refund' ? 'warning' : 'active')
const availabilityPresentation = (availability?: PoolAvailabilityStatus | null, accountStatus?: string | null) => {
  const value = availability || (accountStatus === 'error' ? 'error' : accountStatus === 'active' ? 'normal' : 'inactive')
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
const selectedOptionLabel = (options: Array<{ value: number; label: string }>, value: number) => options.find((item) => item.value === value)?.label || '-'
const batchErrorLabel = (error: string) => {
  const key = `admin.sharedPool.ledger.errors.${error}`
  return te(key) ? t(key) : error
}

async function loadReferences() {
  const [userResponse, sourceResponse, referencedSourceResponse, accountResponse] = await Promise.all([
    adminAPI.users.list(1, 200, { sort_by: 'email', sort_order: 'asc' }),
    adminAPI.sharedPool.listPurchaseSources(),
    adminAPI.sharedPool.listPurchaseSources({ referenced: true }),
    adminAPI.accounts.list(1, 200, { lite: 'true', sort_by: 'name', sort_order: 'asc' })
  ])
  users.value = userResponse.items.map((user) => ({ id: user.id, label: user.username || user.email, active: user.status === 'active' }))
  sources.value = sourceResponse
  referencedSources.value = referencedSourceResponse
  accountChoices.value = accountResponse.items.map((account) => ({ id: account.id, name: account.name }))
}

async function loadSummary() {
  loading.value = true
  try {
    const response = await adminAPI.sharedPool.listCostUploaderSummaries({
      page: summaryPagination.page,
      page_size: summaryPagination.page_size,
      search: summaryFilters.search.trim() || undefined,
      uploader_user_id: summaryFilters.uploader_user_id || undefined,
      payer_user_id: summaryFilters.payer_user_id || undefined,
      purchase_source_id: summaryFilters.purchase_source_id || undefined,
      availability_status: summaryFilters.availability_status || undefined,
      lifecycle_status: summaryFilters.lifecycle_status || undefined,
      has_cost: summaryFilters.has_cost === '' ? undefined : summaryFilters.has_cost === 'true'
    })
    uploaderSummaries.value = response.items || []
    summaryPagination.total = response.total || 0
    expandedUploaderGroups.value = new Set()
    Object.keys(uploaderAccountStates).forEach((key) => delete uploaderAccountStates[key])
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.ledger.errors.load'))
  } finally { loading.value = false }
}

async function loadUploaderAccounts(group: SharedPoolCostUploaderSummary, page: number) {
  const state = uploaderGroupState(group)
  state.loading = true
  try {
    const response = await adminAPI.sharedPool.listCostSummaries({
      page,
      page_size: state.pageSize,
      search: summaryFilters.search.trim() || undefined,
      uploader_user_id: group.uploader_user_id || undefined,
      uploader_unassigned: !group.uploader_user_id || undefined,
      payer_user_id: summaryFilters.payer_user_id || undefined,
      purchase_source_id: summaryFilters.purchase_source_id || undefined,
      availability_status: summaryFilters.availability_status || undefined,
      lifecycle_status: summaryFilters.lifecycle_status || undefined,
      has_cost: summaryFilters.has_cost === '' ? undefined : summaryFilters.has_cost === 'true'
    })
    state.items = response.items || []
    state.page = page
    state.total = response.total || 0
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.ledger.errors.load'))
  } finally {
    state.loading = false
  }
}

async function loadEntries() {
  loading.value = true
  try {
    const response = await adminAPI.sharedPool.listLedgerEntries({
      page: entryPagination.page,
      page_size: entryPagination.page_size,
      search: entryFilters.search.trim() || undefined,
      account_id: entryFilters.account_id || undefined,
      uploader_user_id: entryFilters.uploader_user_id || undefined,
      payer_user_id: entryFilters.payer_user_id || undefined,
      purchase_source_id: entryFilters.purchase_source_id || undefined,
      entry_type: entryFilters.entry_type || undefined,
      start_date: entryFilters.start_date || undefined,
      end_date: entryFilters.end_date || undefined
    })
    entries.value = response.items || []
    entryPagination.total = response.total || 0
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.ledger.errors.load'))
  } finally { loading.value = false }
}

const loadActiveView = () => activeView.value === 'summary' ? loadSummary() : loadEntries()
async function replaceLedgerQuery(values: Record<string, string | number | undefined>) {
  const query = { ...route.query }
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== '') query[key] = String(value)
    else delete query[key]
  }
  await router.replace({ query })
}
const syncSummaryQuery = () => replaceLedgerQuery({
  ledger_view: 'summary',
  ledger_summary_page: summaryPagination.page,
  ledger_summary_page_size: summaryPagination.page_size,
  ledger_summary_search: summaryFilters.search.trim() || undefined,
  uploader_user_id: summaryFilters.uploader_user_id || undefined,
  payer_user_id: summaryFilters.payer_user_id || undefined,
  purchase_source_id: summaryFilters.purchase_source_id || undefined,
  ledger_availability_status: summaryFilters.availability_status || undefined,
  ledger_lifecycle_status: summaryFilters.lifecycle_status || undefined,
  ledger_has_cost: summaryFilters.has_cost || undefined
})
const syncEntryQuery = () => replaceLedgerQuery({
  ledger_view: 'entries',
  ledger_entry_page: entryPagination.page,
  ledger_entry_page_size: entryPagination.page_size,
  ledger_entry_search: entryFilters.search.trim() || undefined,
  account_id: entryFilters.account_id || undefined,
  uploader_user_id: entryFilters.uploader_user_id || undefined,
  payer_user_id: entryFilters.payer_user_id || undefined,
  purchase_source_id: entryFilters.purchase_source_id || undefined,
  ledger_entry_type: entryFilters.entry_type || undefined,
  ledger_entry_start: entryFilters.start_date || undefined,
  ledger_entry_end: entryFilters.end_date || undefined
})
async function switchView(view: LedgerView) { activeView.value = view; await (view === 'summary' ? syncSummaryQuery() : syncEntryQuery()); await loadActiveView() }
async function applySummaryFilters() { summaryPagination.page = 1; await syncSummaryQuery(); await loadSummary() }
async function applyEntryFilters() { entryPagination.page = 1; await syncEntryQuery(); await loadEntries() }
async function changeSummaryPage(page: number) { summaryPagination.page = page; await syncSummaryQuery(); await loadSummary() }
async function changeSummaryPageSize(size: number) { summaryPagination.page_size = size; summaryPagination.page = 1; await syncSummaryQuery(); await loadSummary() }
async function changeEntryPage(page: number) { entryPagination.page = page; await syncEntryQuery(); await loadEntries() }
async function changeEntryPageSize(size: number) { entryPagination.page_size = size; entryPagination.page = 1; await syncEntryQuery(); await loadEntries() }
async function showAccountEntries(accountId: number) { entryFilters.account_id = accountId; entryPagination.page = 1; activeView.value = 'entries'; await syncEntryQuery(); await loadEntries() }

const ledgerRouteSignature = computed(() => JSON.stringify([
  'ledger_view',
  'ledger_summary_page', 'ledger_summary_page_size', 'ledger_summary_search',
  'ledger_availability_status', 'ledger_lifecycle_status', 'ledger_has_cost',
  'ledger_entry_page', 'ledger_entry_page_size', 'ledger_entry_search',
  'ledger_entry_type', 'ledger_entry_start', 'ledger_entry_end',
  'account_id', 'uploader_user_id', 'payer_user_id', 'purchase_source_id'
].map(queryText)))

watch(ledgerRouteSignature, () => {
  const nextView: LedgerView = queryText('ledger_view') === 'entries' || (queryID('account_id') > 0 && !queryText('ledger_view'))
    ? 'entries'
    : 'summary'
  const nextSummary = {
    page: queryPage('ledger_summary_page', 1),
    page_size: queryPage('ledger_summary_page_size', 20),
    search: queryText('ledger_summary_search'),
    uploader_user_id: queryID('uploader_user_id'),
    payer_user_id: queryID('payer_user_id'),
    purchase_source_id: queryID('purchase_source_id'),
    availability_status: queryText('ledger_availability_status') as PoolAvailabilityStatus | '',
    lifecycle_status: queryText('ledger_lifecycle_status'),
    has_cost: queryText('ledger_has_cost')
  }
  const nextEntries = {
    page: queryPage('ledger_entry_page', 1),
    page_size: queryPage('ledger_entry_page_size', 20),
    search: queryText('ledger_entry_search'),
    account_id: queryID('account_id'),
    uploader_user_id: queryID('uploader_user_id'),
    payer_user_id: queryID('payer_user_id'),
    purchase_source_id: queryID('purchase_source_id'),
    entry_type: queryText('ledger_entry_type') as PoolCostEntryType | '',
    start_date: queryText('ledger_entry_start'),
    end_date: queryText('ledger_entry_end')
  }
  const activeChanged = nextView !== activeView.value || (nextView === 'summary'
    ? summaryPagination.page !== nextSummary.page ||
      summaryPagination.page_size !== nextSummary.page_size ||
      summaryFilters.search !== nextSummary.search ||
      summaryFilters.uploader_user_id !== nextSummary.uploader_user_id ||
      summaryFilters.payer_user_id !== nextSummary.payer_user_id ||
      summaryFilters.purchase_source_id !== nextSummary.purchase_source_id ||
      summaryFilters.availability_status !== nextSummary.availability_status ||
      summaryFilters.lifecycle_status !== nextSummary.lifecycle_status ||
      summaryFilters.has_cost !== nextSummary.has_cost
    : entryPagination.page !== nextEntries.page ||
      entryPagination.page_size !== nextEntries.page_size ||
      entryFilters.search !== nextEntries.search ||
      entryFilters.account_id !== nextEntries.account_id ||
      entryFilters.uploader_user_id !== nextEntries.uploader_user_id ||
      entryFilters.payer_user_id !== nextEntries.payer_user_id ||
      entryFilters.purchase_source_id !== nextEntries.purchase_source_id ||
      entryFilters.entry_type !== nextEntries.entry_type ||
      entryFilters.start_date !== nextEntries.start_date ||
      entryFilters.end_date !== nextEntries.end_date)

  activeView.value = nextView
  Object.assign(summaryPagination, { page: nextSummary.page, page_size: nextSummary.page_size })
  Object.assign(summaryFilters, {
    search: nextSummary.search,
    uploader_user_id: nextSummary.uploader_user_id,
    payer_user_id: nextSummary.payer_user_id,
    purchase_source_id: nextSummary.purchase_source_id,
    availability_status: nextSummary.availability_status,
    lifecycle_status: nextSummary.lifecycle_status,
    has_cost: nextSummary.has_cost
  })
  Object.assign(entryPagination, { page: nextEntries.page, page_size: nextEntries.page_size })
  Object.assign(entryFilters, {
    search: nextEntries.search,
    account_id: nextEntries.account_id,
    uploader_user_id: nextEntries.uploader_user_id,
    payer_user_id: nextEntries.payer_user_id,
    purchase_source_id: nextEntries.purchase_source_id,
    entry_type: nextEntries.entry_type,
    start_date: nextEntries.start_date,
    end_date: nextEntries.end_date
  })
  if (activeChanged) void loadActiveView()
})

async function loadBatchAccounts(search = accountSearch.value) {
  try {
    const response = await adminAPI.accounts.list(1, 100, { search: search.trim() || undefined, lite: 'true', sort_by: 'name', sort_order: 'asc' })
    batchAccounts.value = response.items.map((account) => ({ id: account.id, name: account.name }))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.sharedPool.ledger.errors.accounts'))
  }
}
function isAccountSelected(accountId: number) { return selectedAccounts.value.some((account) => account.id === accountId) }
function toggleAccount(account: AccountChoice) {
  selectedAccounts.value = isAccountSelected(account.id) ? selectedAccounts.value.filter((item) => item.id !== account.id) : [...selectedAccounts.value, account]
  if (!overrides[account.id]) overrides[account.id] = { originalAmount: null, expectedTokenCount: null }
}
function toggleAllVisible() {
  if (allVisibleSelected.value) {
    const visible = new Set(batchAccounts.value.map((item) => item.id))
    selectedAccounts.value = selectedAccounts.value.filter((item) => !visible.has(item.id))
  } else {
    const selected = new Set(selectedAccounts.value.map((item) => item.id))
    selectedAccounts.value = [...selectedAccounts.value, ...batchAccounts.value.filter((item) => !selected.has(item.id))]
    batchAccounts.value.forEach((account) => { if (!overrides[account.id]) overrides[account.id] = { originalAmount: null, expectedTokenCount: null } })
  }
}
function openBatchDialog() {
  Object.assign(batchForm, emptyBatchForm())
  Object.keys(overrides).forEach((key) => delete overrides[Number(key)])
  selectedAccounts.value = []
  accountSearch.value = ''
  batchErrors.value = []
  batchStep.value = 1
  showBatchDialog.value = true
  void loadBatchAccounts('')
}
function closeBatchDialog() { if (!submitting.value) showBatchDialog.value = false }
function previousBatchStep() { batchErrors.value = []; batchStep.value = Math.max(1, batchStep.value - 1) }
function setAmountMode(mode: SharedPoolBatchAmountMode) {
  batchForm.amount_mode = mode
  batchForm.allocation_mode = 'equal'
  Object.values(overrides).forEach((item) => { item.originalAmount = null })
}
function overrideTokenMillions(accountId: number) {
  const value = overrides[accountId]?.expectedTokenCount
  return typeof value === 'number' ? tokensToMillions(value) : ''
}
function setOverrideTokenMillions(accountId: number, event: Event) {
  const value = (event.target as HTMLInputElement).value
  overrides[accountId].expectedTokenCount = value === '' ? null : millionsToTokens(Number(value))
}
function validateStep(step: number): string[] {
  if (step === 1) return selectedAccounts.value.length ? [] : ['accounts_required']
  if (step === 2) {
    const errors: string[] = []
    if (!batchForm.payer_user_id) errors.push('payer_required')
    if (!batchForm.purchase_source_id) errors.push('source_required')
    if (!(batchForm.amount > 0)) errors.push('amount_invalid')
    if (!Number.isSafeInteger(batchForm.expected_token_count) || batchForm.expected_token_count <= 0) errors.push('expected_tokens_invalid')
    if (!batchForm.service_start || !batchForm.service_end || batchForm.service_end <= batchForm.service_start) errors.push('period_invalid')
    if (batchPaidAtFuture.value) errors.push('future_paid_at')
    return errors
  }
  return allocationResult.value.errors
}
function nextBatchStep() {
  batchErrors.value = validateStep(batchStep.value)
  if (batchErrors.value.length) return
  batchStep.value = Math.min(4, batchStep.value + 1)
}
function overrideAmountPlaceholder(accountId: number) {
  if (batchForm.amount_mode === 'per_account') return String(batchForm.amount || '')
  const allocation = allocationResult.value.allocations.find((item) => item.accountId === accountId)
  return allocation ? allocation.originalAmount.toFixed(2) : ''
}

async function submitBatch() {
  batchErrors.value = validateStep(3)
  if (batchErrors.value.length) { batchStep.value = 3; return }
  const request: BatchSharedPoolCostRequest = {
    amount_mode: batchForm.amount_mode,
    common: {
      payer_user_id: batchForm.payer_user_id,
      purchase_source_id: batchForm.purchase_source_id || undefined,
      entry_type: batchForm.entry_type,
      original_amount: Number(batchForm.amount).toFixed(2),
      currency: batchForm.currency,
      service_start: batchForm.service_start,
      service_end: batchForm.service_end,
      warranty_end: batchForm.warranty_end || undefined,
      paid_at: poolPaidAtToISOString(batchForm.paid_at),
      order_no: batchForm.order_no || undefined,
      purchase_url: batchForm.purchase_url || undefined,
      notes: batchForm.notes || undefined,
      expected_token_count: batchForm.expected_token_count
    },
    accounts: allocationResult.value.allocations.map((item) => ({
      account_id: item.accountId,
      original_amount: item.originalAmount.toFixed(2),
      expected_token_count: item.expectedTokenCount
    }))
  }
  submitting.value = true
  try {
    const result = await adminAPI.sharedPool.createBatchCosts(request)
    appStore.showSuccess(t('admin.sharedPool.ledger.batchSaved', { count: result.account_count, total: formatMoney(Number(result.total_original_amount), batchForm.currency) }))
    showBatchDialog.value = false
    await Promise.all([loadSummary(), activeView.value === 'entries' ? loadEntries() : Promise.resolve()])
  } catch (error: any) {
    batchErrors.value = [error?.message || t('admin.sharedPool.ledger.errors.submit')]
  } finally { submitting.value = false }
}

onMounted(async () => {
  if (props.initialAccountId && !queryText('ledger_view')) activeView.value = 'entries'
  try { await loadReferences() } catch (error: any) { appStore.showError(error?.message || t('admin.sharedPool.ledger.errors.options')) }
  await loadActiveView()
})

defineExpose({ reload: loadActiveView })
</script>
