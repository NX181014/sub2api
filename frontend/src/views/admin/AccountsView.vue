<template>
  <component
    :is="embedded ? 'div' : AppLayout"
    :class="embedded ? 'accounts-workbench-root min-w-0 max-w-full overflow-x-clip' : ''"
  >
    <TablePageLayout>
      <template #filters>
        <div class="account-operation-band">
          <AccountTableFilters
            class="account-operation-filters"
            :search-query="params.search"
            :filters="params"
            :groups="groups"
            :uploaders="uploaderOptions"
            :result-account-count="visibleResultAccountCount"
            :result-batch-count="visibleResultBatchCount"
            @update:filters="handleFiltersChange"
            @update:search-query="handleSearchChange"
            @clear="clearFilters"
          />
          <AccountTableActions
            class="account-operation-actions"
            :loading="loading"
            @refresh="handleManualRefresh"
            @create="handleCreateRequest"
          >
            <template #after>
              <button
                v-if="hasEffectiveFilters && selIds.length === 0"
                type="button"
                data-test="edit-filtered"
                class="btn btn-secondary min-h-11 px-3"
                @click="openBulkEditFiltered"
              >
                <Icon name="edit" size="sm" />
                <span class="hidden lg:inline">{{ t('admin.accounts.bulkActions.editFiltered', { count: pagination.total }) }}</span>
                <span class="lg:hidden">{{ pagination.total }}</span>
              </button>
              <button
                type="button"
                class="btn btn-secondary relative px-2 md:px-3"
                :title="t('admin.sharedPool.approval.title')"
                :aria-label="t('admin.sharedPool.approval.title')"
                @click="openApprovalCenter"
              >
                <Icon name="shield" size="sm" />
                <span class="hidden md:inline">{{ t('admin.sharedPool.approval.title') }}</span>
                <span
                  v-if="pendingApprovalCount"
                  class="inline-flex min-w-5 items-center justify-center rounded-full bg-amber-100 px-1.5 text-xs font-semibold tabular-nums text-amber-700 dark:bg-amber-900/40 dark:text-amber-300"
                >
                  {{ pendingApprovalCount > 99 ? '99+' : pendingApprovalCount }}
                </span>
              </button>

              <!-- Auto Refresh Dropdown -->
              <div class="relative" ref="autoRefreshDropdownRef">
                <button
                  @click="
                    showAutoRefreshDropdown = !showAutoRefreshDropdown;
                    showAccountToolsDialog = false
                  "
                  class="btn btn-secondary px-2 md:px-3"
                  :title="t('admin.accounts.autoRefresh')"
                >
                  <Icon name="refresh" size="sm" :class="[autoRefreshEnabled ? 'animate-spin' : '']" />
                  <span class="hidden md:inline">
                    {{
                      autoRefreshEnabled
                        ? t('admin.accounts.autoRefreshCountdown', { seconds: autoRefreshCountdown })
                        : t('admin.accounts.autoRefresh')
                    }}
                  </span>
                </button>
                <div
                  v-if="showAutoRefreshDropdown"
                  class="absolute right-0 z-50 mt-2 w-56 origin-top-right rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-700 dark:bg-dark-800"
                >
                  <div class="p-2">
                    <button
                      @click="setAutoRefreshEnabled(!autoRefreshEnabled)"
                      class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700"
                    >
                      <span>{{ t('admin.accounts.enableAutoRefresh') }}</span>
                      <Icon v-if="autoRefreshEnabled" name="check" size="sm" class="text-primary-500" />
                    </button>
                    <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>
                    <button
                      v-for="sec in autoRefreshIntervals"
                      :key="sec"
                      @click="setAutoRefreshInterval(sec)"
                      class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700"
                    >
                      <span>{{ autoRefreshIntervalLabel(sec) }}</span>
                      <Icon v-if="autoRefreshIntervalSeconds === sec" name="check" size="sm" class="text-primary-500" />
                    </button>
                  </div>
                </div>
              </div>

              <!-- More Tools Dropdown -->
              <div>
                <button
                  @click="openAccountToolsDialog"
                  class="btn btn-secondary px-2 md:px-3"
                  :title="t('admin.accounts.moreActions')"
                  :aria-expanded="showAccountToolsDialog"
                >
                  <Icon name="more" size="sm" class="md:mr-1.5" />
                  <span class="hidden md:inline">{{ t('admin.accounts.moreActions') }}</span>
                </button>
              </div>
            </template>
          </AccountTableActions>
        </div>
        <div
          v-if="hasPendingListSync"
          class="mt-2 flex items-center justify-between rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200"
        >
          <span>{{ t('admin.accounts.listPendingSyncHint') }}</span>
          <button
            class="btn btn-secondary px-2 py-1 text-xs"
            @click="syncPendingListChanges"
          >
            {{ t('admin.accounts.listPendingSyncAction') }}
          </button>
        </div>
      </template>
      <template #table>
        <div :class="embedded ? 'workbench-layout min-h-0 flex-1' : 'contents'">
          <button
            v-if="embedded && mobileWorkbenchNavigatorExpanded"
            type="button"
            class="workbench-navigator-backdrop"
            :aria-label="t('common.close')"
            @click="mobileWorkbenchNavigatorExpanded = false"
          ></button>
          <aside
            v-if="embedded"
            id="workbench-navigator"
            ref="workbenchSidebarRef"
            class="workbench-sidebar shrink-0 border-b border-gray-200 bg-gray-50/60 dark:border-dark-700 dark:bg-dark-900/30"
            :class="{ 'workbench-mobile-collapsed': !mobileWorkbenchNavigatorExpanded }"
            aria-labelledby="workbench-navigator-title"
            :role="mobileWorkbenchNavigatorExpanded ? 'dialog' : undefined"
            :aria-modal="mobileWorkbenchNavigatorExpanded ? 'true' : undefined"
            tabindex="-1"
          >
            <div data-test="workbench-sidebar-header" class="workbench-sidebar-header">
              <div class="mb-2 flex items-center justify-between gap-3">
                <h2 id="workbench-navigator-title" class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accounts.workbenchNavigatorTitle') }}</h2>
                <span class="flex items-center gap-2">
                  <span class="text-xs tabular-nums text-gray-500 dark:text-gray-400">
                    {{ t('admin.accounts.workbenchBatchCount', { count: workbenchBatchTotal }) }}
                  </span>
                  <button
                    type="button"
                    data-test="workbench-mobile-toggle"
                    class="workbench-mobile-toggle inline-flex min-h-11 min-w-11 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700"
                    :aria-expanded="mobileWorkbenchNavigatorExpanded"
                    :aria-label="t('admin.accounts.workbenchNavigatorTitle')"
                    @click="mobileWorkbenchNavigatorExpanded ? closeMobileWorkbenchNavigator() : openMobileWorkbenchNavigator()"
                  >
                    <Icon :name="mobileWorkbenchNavigatorExpanded ? 'chevronDown' : 'chevronRight'" size="sm" />
                  </button>
                </span>
              </div>
              <label class="workbench-navigator-search relative block">
                <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  v-model.trim="workbenchNavigatorSearch"
                  data-test="workbench-sidebar-search"
                  type="search"
                  :placeholder="t('admin.accounts.workbenchNavigatorSearch')"
                  :aria-label="t('admin.accounts.workbenchNavigatorSearch')"
                  class="h-11 w-full rounded-lg border border-gray-200 bg-white pl-9 pr-3 text-sm text-gray-900 outline-none transition focus:border-primary-400 focus:ring-2 focus:ring-primary-100 dark:border-dark-600 dark:bg-dark-800 dark:text-white dark:focus:border-primary-600 dark:focus:ring-primary-900/40"
                />
              </label>
            </div>
            <div class="workbench-scope-list">
              <button type="button" data-test="workbench-all" :aria-current="workbenchScope === 'all' ? 'page' : undefined" :class="['workbench-nav-item', workbenchScope === 'all' ? 'workbench-nav-item-active' : '']" @click="selectWorkbenchScope('all')">
                <span class="truncate font-medium">{{ t('admin.sharedPool.ledger.allAccounts') }}</span>
                <span class="workbench-nav-count">{{ workbenchAccountTotal }}</span>
              </button>
              <button type="button" data-test="workbench-standalone" :aria-current="workbenchScope === 'standalone' ? 'page' : undefined" :class="['workbench-nav-item', workbenchScope === 'standalone' ? 'workbench-nav-item-active' : '']" @click="selectWorkbenchScope('standalone')">
                <span class="truncate font-medium">{{ t('admin.accounts.standaloneImport') }}</span>
                <span class="workbench-nav-count">{{ workbenchStandaloneCount }}</span>
              </button>
            </div>
            <div class="workbench-navigator-modes" :aria-label="t('admin.accounts.workbenchNavigatorTitle')">
              <button type="button" data-test="workbench-mode-uploader" :aria-pressed="workbenchNavigatorMode === 'uploader'" :class="['workbench-navigator-mode', workbenchNavigatorMode === 'uploader' ? 'workbench-navigator-mode-active' : '']" @click="setWorkbenchNavigatorMode('uploader')">
                {{ t('admin.accounts.workbenchBrowseByUploader') }}
              </button>
              <button type="button" data-test="workbench-mode-batch" :aria-pressed="workbenchNavigatorMode === 'batch'" :class="['workbench-navigator-mode', workbenchNavigatorMode === 'batch' ? 'workbench-navigator-mode-active' : '']" @click="setWorkbenchNavigatorMode('batch')">
                {{ t('admin.accounts.workbenchBrowseByBatch') }}
              </button>
            </div>
            <div class="workbench-navigator-list">
              <template v-if="workbenchNavigatorMode === 'uploader'">
                <button
                  v-for="group in filteredWorkbenchUploaderGroups"
                  :key="String(group.id)"
                  type="button"
                  data-test="workbench-uploader"
                  :aria-current="workbenchScope === 'uploader' && String(selectedWorkbenchUploaderID) === String(group.id) ? 'page' : undefined"
                  :class="['workbench-nav-item', workbenchScope === 'uploader' && String(selectedWorkbenchUploaderID) === String(group.id) ? 'workbench-nav-item-active' : '']"
                  @click="selectWorkbenchUploader(group)"
                >
                  <Icon name="user" size="sm" class="shrink-0 text-gray-400" />
                  <span class="min-w-0 flex-1 truncate font-medium" :title="group.label">{{ group.label }}</span>
                  <span class="workbench-nav-count">{{ group.count }}</span>
                </button>
              </template>
              <template v-else>
                <div v-if="selectedWorkbenchUploader" class="workbench-batch-context">
                  <span class="min-w-0 truncate" :title="selectedWorkbenchUploader.label">
                    {{ selectedWorkbenchUploader.label }}
                  </span>
                  <button type="button" class="workbench-context-clear" @click="clearWorkbenchUploaderBatchFilter">
                    {{ t('admin.accounts.workbenchViewAllBatches') }}
                  </button>
                </div>
                <div v-for="batch in filteredWorkbenchBatches" :key="batch.id" :class="['workbench-batch-item', selectedWorkbenchBatchID === batch.id ? 'workbench-batch-item-active' : '']">
                  <button type="button" data-test="workbench-batch" class="workbench-batch-main" :aria-current="selectedWorkbenchBatchID === batch.id ? 'page' : undefined" @click="selectWorkbenchBatch(batch)">
                    <span class="min-w-0 flex-1">
                      <span class="block truncate font-medium text-gray-800 dark:text-gray-100" :title="batch.names.join('、')">{{ batch.names.join('、') || t('admin.accounts.importBatchGroup') }}</span>
                      <span class="workbench-batch-meta" :title="`${batch.uploader_username || batch.uploader_email || t('admin.sharedPool.ledger.unassignedUploader')} · ${formatDateTime(batch.created_at)}`">
                        {{ batch.uploader_username || batch.uploader_email || t('admin.sharedPool.ledger.unassignedUploader') }} · {{ formatDateTime(batch.created_at) }}
                      </span>
                    </span>
                    <span class="workbench-nav-count">{{ batch.matched_count }}</span>
                  </button>
                </div>
              </template>
              <p v-if="!workbenchNavigatorLoading && workbenchNavigatorResultCount === 0" class="workbench-navigator-empty">
                {{ t('admin.accounts.workbenchNavigatorEmpty') }}
              </p>
              <button
                v-if="workbenchBatchPage < workbenchBatchPages"
                type="button"
                data-test="workbench-load-more"
                class="workbench-view-all"
                :disabled="workbenchNavigatorLoadingMore"
                @click="loadMoreWorkbenchBatches"
              >
                <LoadingSpinner v-if="workbenchNavigatorLoadingMore" size="sm" />
                <template v-else>{{ t('admin.accounts.workbenchLoadMore') }}</template>
              </button>
              <LoadingSpinner v-if="workbenchNavigatorLoading" class="m-3" />
            </div>
          </aside>

          <section class="flex min-h-0 min-w-0 flex-1 flex-col">
            <header v-if="embedded" class="workbench-header flex flex-wrap items-start justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-700">
              <div class="flex min-w-0 items-start gap-2">
                <button
                  type="button"
                  class="workbench-navigator-open min-h-11 min-w-11 items-center justify-center rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-dark-700 dark:text-gray-300 dark:hover:bg-dark-800"
                  :aria-label="t('admin.accounts.workbenchNavigatorTitle')"
                  aria-controls="workbench-navigator"
                  :aria-expanded="mobileWorkbenchNavigatorExpanded"
                  @click="openMobileWorkbenchNavigator"
                >
                  <Icon name="grid" size="sm" />
                </button>
                <div v-if="!embedded || workbenchView !== 'finance'" class="min-w-0">
                  <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white" :title="workbenchTitle">{{ workbenchTitle }}</h2>
                  <p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400" :title="workbenchSubtitle">{{ workbenchSubtitle }}</p>
                </div>
              </div>
              <div class="flex max-w-full flex-wrap items-center justify-end gap-2">
                <label class="workbench-sort-control">
                  <Icon name="sort" size="sm" aria-hidden="true" />
                  <span class="sr-only">{{ t('admin.accounts.workbenchSortLabel') }}</span>
                  <select v-model="workbenchSortValue" :aria-label="t('admin.accounts.workbenchSortLabel')">
                    <option value="created_at:desc">{{ t('admin.accounts.workbenchSortNewest') }}</option>
                    <option value="name:asc">{{ t('admin.accounts.workbenchSortName') }}</option>
                    <option value="status:asc">{{ t('admin.accounts.workbenchSortStatus') }}</option>
                  </select>
                </label>
                <div class="workbench-view-switch" :aria-label="t('admin.accounts.display.view')">
                  <button
                    v-for="view in workbenchViews"
                    :key="view"
                    type="button"
                    :class="['workbench-view-option', workbenchView === view ? 'workbench-view-option-active' : '']"
                    @click="setWorkbenchView(view)"
                  >
                    {{ t(`admin.accounts.display.${view}`) }}
                  </button>
                </div>
                <template v-if="selectedWorkbenchBatch">
                <button
                  type="button"
                  class="inline-flex min-h-11 min-w-11 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:bg-dark-700"
                  :title="t('common.copy')"
                  :aria-label="t('common.copy')"
                  @click="copyWorkbenchBatchID(selectedWorkbenchBatch.id)"
                >
                  <Icon name="copy" size="sm" />
                </button>
                <span
                  v-for="item in batchStatusItems(selectedWorkbenchBatch.status)"
                  :key="item.key"
                  :class="['inline-flex items-center rounded px-2 py-1 text-xs font-medium', item.className]"
                >
                  {{ item.label }} {{ item.count }}
                </span>
                <span class="inline-flex items-center rounded bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  {{ t('admin.accounts.importBatchSchedulable', { current: selectedWorkbenchBatch.schedulable_count, total: selectedWorkbenchBatch.matched_count }) }}
                </span>
                </template>
              </div>
            </header>
        <AccountBulkActionsBar
          v-if="embedded || selIds.length > 0 || hasEffectiveFilters"
          :selected-ids="selIds"
          :selected-batch-count="selectedBatchCount"
          :filtered-count="pagination.total"
          :has-active-filters="hasEffectiveFilters"
          :hidden-selected-count="hiddenSelectedCount"
          :all-page-selected="allVisibleSelected"
          :page-selected-count="visibleSelectedCount"
          :busy="loading || pageBatchLoading || bulkActionInProgress"
          @delete="handleBulkDelete"
          @reset-status="handleBulkResetStatus"
          @refresh-token="handleBulkRefreshToken"
          @probe-upstream-billing="handleBulkProbeUpstreamBilling"
          @edit-selected="openBulkEditSelected"
          @edit-filtered="openBulkEditFiltered"
          @clear="clearSelection"
          @toggle-page="togglePageSelection"
          @toggle-schedulable="handleBulkToggleSchedulable"
        />
        <div
          ref="accountTableRef"
          class="account-workbench-table flex min-h-0 flex-1 flex-col overflow-hidden"
          :class="[
            { 'is-embedded': embedded, 'is-compact': embedded && workbenchDensity === 'compact', 'is-finance-first': embedded && workbenchModuleOrder === 'finance' },
            embedded ? `workbench-view-${workbenchView}` : ''
          ]"
        >
        <DataTable
          ref="dataTableRef"
          :columns="cols"
          :data="accountTableRows"
          :loading="loading"
          row-key="id"
          :server-side-sort="true"
          @sort="handleSort"
          default-sort-key="created_at"
          default-sort-order="desc"
          :sort-storage-key="ACCOUNT_SORT_STORAGE_KEY"
          :sticky-first-column="!embedded"
          :sticky-actions-column="!embedded"
          :estimate-row-height="168"
          :overscan="5"
          :virtualize-threshold="50"
          :mobile-column-keys="embedded
            ? ['select', 'name', 'status', 'usage', 'actions']
            : ['select', 'uploader', 'usage', 'pool_record', 'status']"
        >
          <template #header-select>
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allVisibleSelected"
              :indeterminate="someVisibleSelected"
              :disabled="pageBatchLoading"
              :aria-label="allVisibleSelected ? t('admin.accounts.bulkActions.clearCurrentPage') : t('admin.accounts.bulkActions.selectCurrentPage')"
              @click.stop
              @change="toggleSelectAllVisible($event)"
            />
          </template>
          <template #cell-select="{ row }">
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isImportBatchRow(row) ? isImportBatchSelected(row) : isSelected(row.id)"
              :indeterminate="isImportBatchRow(row) && isImportBatchPartiallySelected(row)"
              :disabled="isImportBatchRow(row) && isImportBatchLoading(row.batchID)"
              :aria-label="isImportBatchRow(row) ? t('admin.accounts.selectImportBatch', { count: row.matchedCount }) : t('common.selectOption')"
              @change="isImportBatchRow(row) ? toggleImportBatchSelection(row, ($event.target as HTMLInputElement).checked) : toggleSel(row.id)"
            />
          </template>
          <template #cell-id="{ row, value }">
            <span v-if="isImportBatchRow(row)" class="text-xs tabular-nums text-gray-500 dark:text-gray-400">
              {{ row.matchedCount === row.totalCount ? t('admin.accounts.importBatchCount', { count: row.totalCount }) : `${row.matchedCount}/${row.totalCount}` }}
            </span>
            <span v-else class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ value }}</span>
          </template>
          <template #cell-name="{ row, value }">
            <button
              v-if="isImportBatchRow(row)"
              type="button"
              class="flex min-h-11 max-w-80 items-center gap-2 text-left"
              :disabled="isImportBatchLoading(row.batchID)"
              :aria-expanded="expandedImportBatches.has(row.batchID)"
              @click="toggleImportBatch(row.batchID)"
            >
              <Icon :name="expandedImportBatches.has(row.batchID) ? 'chevronDown' : 'chevronRight'" size="sm" class="shrink-0 text-gray-500 dark:text-gray-400" />
              <span class="min-w-0">
                <span class="block font-semibold text-gray-900 dark:text-white">{{ t('admin.accounts.importBatchGroup') }}</span>
                <span class="block max-w-64 truncate text-xs text-gray-500 dark:text-gray-400" :title="row.names">{{ row.names }}</span>
              </span>
            </button>
            <div
              v-else
              data-test="account-workbench-identity"
              :data-account-id="embedded ? row.id : undefined"
              class="flex min-w-0 flex-col"
              :class="isImportBatchChild(row) ? 'border-l-2 border-gray-200 pl-4 dark:border-dark-600' : ''"
            >
              <button
                v-if="embedded"
                type="button"
                data-test="account-workbench-card"
                class="min-h-11 w-full max-w-[220px] truncate text-left font-semibold text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
                :title="value"
                @click="emit('trace-account', row.id)"
              >
                {{ value }}
              </button>
              <HelpTooltip
                v-else-if="accountHomepageUrl(row)"
                :content="accountHomepageUrl(row)"
                width-class="w-max max-w-sm break-all"
                class="-ml-1 self-start"
              >
                <template #trigger>
                  <a
                    :href="accountHomepageUrl(row)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="border-b border-dotted border-gray-300 font-medium text-gray-900 dark:border-dark-600 dark:text-white"
                  >
                    {{ value }}
                  </a>
                </template>
              </HelpTooltip>
              <span v-else class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <span
                class="max-w-[220px] truncate text-xs text-gray-500 dark:text-gray-400"
                :title="accountIdentitySubtitle(row)"
              >
                {{ accountIdentitySubtitle(row) }}
              </span>
              <div v-if="embedded" class="mt-2 flex max-w-[220px] flex-wrap items-center gap-1">
                <PlatformTypeBadge :platform="row.platform" :type="row.type"
                  :auth-mode="getOpenAIAuthMode(row)"
                  :plan-type="getAccountPlanType(row)"
                  :privacy-mode="row.extra?.privacy_mode || row.parent_privacy_mode"
                  :subscription-expires-at="row.credentials?.subscription_expires_at || row.parent_subscription_expires_at" />
                <span class="font-mono text-[11px] text-gray-400 dark:text-gray-500">#{{ row.id }}</span>
              </div>
            </div>
          </template>
          <template #cell-uploader="{ row }">
            <button
              v-if="isImportBatchRow(row)"
              type="button"
              class="flex min-h-11 max-w-64 items-center gap-2 text-left"
              :disabled="isImportBatchLoading(row.batchID)"
              :aria-expanded="expandedImportBatches.has(row.batchID)"
              @click="toggleImportBatch(row.batchID)"
            >
              <Icon :name="expandedImportBatches.has(row.batchID) ? 'chevronDown' : 'chevronRight'" size="sm" class="shrink-0 text-gray-500 md:hidden dark:text-gray-400" />
              <span class="min-w-0">
                <span class="block truncate font-semibold text-gray-900 dark:text-white" :title="row.uploader">{{ row.uploader }}</span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.importBatchSummary', { count: row.matchedCount, time: formatDateTime(row.createdAt) }) }}</span>
                <span class="mt-1 block max-w-56 truncate text-xs text-gray-500 dark:text-gray-400 md:hidden" :title="row.names">{{ row.names }}</span>
              </span>
            </button>
            <div v-else-if="isImportBatchChild(row)" class="min-w-0 border-l-2 border-gray-200 pl-3 dark:border-dark-600 md:border-0 md:pl-0">
              <p class="max-w-52 truncate font-medium text-gray-900 dark:text-white md:hidden" :title="row.name">
                {{ row.name }} · #{{ row.id }}
              </p>
              <span class="hidden text-xs text-gray-400 dark:text-gray-500 md:inline">—</span>
            </div>
            <div v-else class="min-w-0">
              <p class="max-w-52 truncate font-medium text-gray-900 dark:text-white" :title="accountUploader(row)">{{ accountUploader(row) }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.standaloneImport') }}</p>
            </div>
          </template>
          <template #cell-notes="{ row, value }">
            <template v-if="!isImportBatchRow(row)">
              <span v-if="value" :title="value" class="block max-w-xs truncate text-sm text-gray-600 dark:text-gray-300">{{ value }}</span>
              <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
            </template>
          </template>
          <template #cell-platform_type="{ row }">
            <div v-if="!isImportBatchRow(row)" class="flex min-w-0 flex-col gap-1">
              <div class="flex flex-wrap items-center gap-1">
                <PlatformTypeBadge :platform="row.platform" :type="row.type"
                  :auth-mode="getOpenAIAuthMode(row)"
                  :plan-type="getAccountPlanType(row)"
                  :privacy-mode="row.extra?.privacy_mode || row.parent_privacy_mode"
                  :subscription-expires-at="row.credentials?.subscription_expires_at || row.parent_subscription_expires_at" />
                <span
                  v-if="getAntigravityTierLabel(row)"
                  :class="['inline-block rounded px-1.5 py-0.5 text-[10px] font-medium', getAntigravityTierClass(row)]"
                >
                  {{ getAntigravityTierLabel(row) }}
                </span>
              </div>
              <div
                v-if="getOpenAICompactMeta(row)"
                :class="[
                  'inline-flex items-center gap-1.5 pl-0.5 text-[11px] font-medium leading-4',
                  getOpenAICompactMeta(row)?.className
                ]"
                :title="getOpenAICompactTitle(row)"
              >
                <span :class="['h-1.5 w-1.5 rounded-full', getOpenAICompactMeta(row)?.dotClass]" />
                <span>{{ getOpenAICompactMeta(row)?.label }}</span>
              </div>
            </div>
          </template>
          <template #cell-capacity="{ row }">
            <AccountCapacityCell v-if="!isImportBatchRow(row)" :account="row" />
          </template>
          <template #cell-status="{ row }">
            <div v-if="isImportBatchRow(row)" class="flex max-w-64 flex-wrap gap-1">
              <span
                v-for="item in importBatchStatusItems(row)"
                :key="item.key"
                :class="['inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium', item.className]"
              >
                {{ item.label }} {{ item.count }}
              </span>
              <span v-if="!importBatchStatusItems(row).length" class="inline-flex rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ t('admin.accounts.status.notChecked') }}
              </span>
            </div>
            <div v-else-if="embedded" data-test="account-workbench-runtime" class="flex min-w-0 flex-col gap-2">
              <AccountStatusIndicator :account="row" @show-temp-unsched="handleShowTempUnsched" />
              <div class="flex flex-wrap items-center gap-3">
                <button
                  @click="handleToggleSchedulable(row)"
                  :disabled="togglingSchedulable === row.id"
                  class="inline-flex min-h-11 items-center gap-2 rounded-md px-1 text-xs font-medium text-gray-600 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-300"
                  :title="row.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')"
                >
                  <span class="relative inline-flex h-5 w-9 rounded-full border-2 border-transparent transition-colors" :class="row.schedulable ? 'bg-primary-500' : 'bg-gray-200 dark:bg-dark-600'">
                    <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition" :class="row.schedulable ? 'translate-x-4' : 'translate-x-0'" />
                  </span>
                  <span>{{ t('admin.accounts.columns.schedulable') }}</span>
                </button>
                <AccountCapacityCell :account="row" />
              </div>
            </div>
            <div v-else class="flex items-center gap-1.5">
              <AccountStatusIndicator :account="row" @show-temp-unsched="handleShowTempUnsched" />
            </div>
          </template>
          <template #cell-schedulable="{ row }">
            <span v-if="isImportBatchRow(row)" class="text-xs font-medium tabular-nums text-gray-600 dark:text-gray-300">
              {{ t('admin.accounts.importBatchSchedulable', { current: row.schedulableCount, total: row.matchedCount }) }}
            </span>
            <button v-else @click="handleToggleSchedulable(row)" :disabled="togglingSchedulable === row.id" class="inline-flex h-11 w-11 flex-shrink-0 cursor-pointer items-center justify-center rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800" :title="row.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')">
              <span class="relative inline-flex h-5 w-9 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out" :class="[row.schedulable ? 'bg-primary-500 hover:bg-primary-600' : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500']">
                <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out" :class="[row.schedulable ? 'translate-x-4' : 'translate-x-0']" />
              </span>
            </button>
          </template>
          <template #cell-today_stats="{ row }">
            <AccountTodayStatsCell
              v-if="!isImportBatchRow(row)"
              :stats="todayStatsByAccountId[String(row.id)] ?? null"
              :loading="todayStatsLoading"
              :error="todayStatsError"
            />
          </template>
          <template #cell-groups="{ row }">
            <AccountGroupsCell v-if="!isImportBatchRow(row)" :groups="row.groups" :max-display="4" />
          </template>
          <template #header-usage="{ column }">
            <div class="flex items-center">
              <span>{{ column.label }}</span>
              <HelpTooltip :content="t('admin.accounts.usageWindowsHint')" width-class="w-72" />
            </div>
          </template>
          <template #cell-usage="{ row }">
            <div
              v-if="!isImportBatchRow(row)"
              data-test="account-workbench-usage"
              :class="embedded ? 'w-full min-w-0' : 'account-usage-card'"
            >
              <div :class="embedded ? 'account-usage-finance' : ''">
                <div class="min-w-0">
                  <div v-if="embedded && workbenchView !== 'finance'" class="mb-2 flex items-center gap-1 text-xs font-semibold text-gray-500 dark:text-gray-400">
                    <span>{{ t('admin.accounts.columns.usageWindows') }}</span>
                    <HelpTooltip :content="t('admin.accounts.usageWindowsHint')" width-class="w-72" />
                  </div>
                  <AccountUsageCell
                    :account="row"
                    :today-stats="todayStatsByAccountId[String(row.id)] ?? null"
                    :today-stats-loading="todayStatsLoading"
                    :manual-refresh-token="usageManualRefreshToken"
                  />
                  <time
                    v-if="embedded"
                    data-test="account-workbench-last-used"
                    class="mt-2 block text-xs text-gray-500 dark:text-gray-400"
                    :datetime="row.last_used_at || undefined"
                    :title="row.last_used_at ? formatDateTime(row.last_used_at) : undefined"
                  >
                    {{ t('admin.accounts.columns.lastUsed') }}: {{ formatRelativeTime(row.last_used_at) }}
                  </time>
                </div>

                <button
                  v-if="embedded && workbenchView !== 'runtime'"
                  type="button"
                  data-test="account-workbench-finance"
                  class="account-finance-summary min-h-11 min-w-0 text-left"
                  :title="t('admin.sharedPool.actions.poolRecord')"
                  @click="emit('pool-record', row)"
                >
                  <div class="flex min-w-0 items-start justify-between gap-2">
                    <div class="min-w-0">
                      <p class="truncate text-xs font-semibold text-gray-800 dark:text-gray-100" :title="poolWorkbenchRecordFor(row).sourceName">
                        {{ poolWorkbenchRecordFor(row).sourceName || t('admin.sharedPool.intake.pending') }}
                      </p>
                      <p v-if="poolWorkbenchRecordFor(row).sourceCount > 1" class="text-[10px] text-gray-500 dark:text-gray-400">
                        {{ t('admin.sharedPool.workbench.sourceCount', { count: poolWorkbenchRecordFor(row).sourceCount }) }}
                      </p>
                    </div>
                    <StatusBadge
                      class="shrink-0"
                      :status="poolTraceBadgeStatus(poolWorkbenchRecordFor(row).traceQuality)"
                      :label="t(`admin.sharedPool.workbench.dataQuality.${poolWorkbenchRecordFor(row).traceQuality}`)"
                    />
                  </div>

                  <dl class="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-[11px]">
                    <div>
                      <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.workbench.grossCost') }}</dt>
                      <dd class="font-medium tabular-nums text-gray-800 dark:text-gray-100">{{ formatPoolCost(poolWorkbenchRecordFor(row).grossCostMinor) }}</dd>
                    </div>
                    <div>
                      <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.ledger.netCost') }}</dt>
                      <dd class="font-medium tabular-nums text-gray-800 dark:text-gray-100">{{ formatPoolCost(poolWorkbenchRecordFor(row).netCostMinor) }}</dd>
                    </div>
                    <div class="col-span-2 flex min-w-0 items-center justify-between gap-2">
                      <dt class="shrink-0 text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.workbench.latestPurchase') }}</dt>
                      <dd class="truncate text-right font-medium text-gray-700 dark:text-gray-200" :title="poolWorkbenchRecordFor(row).latestPurchaseAt ? formatDateTime(poolWorkbenchRecordFor(row).latestPurchaseAt) : '-'">
                        {{ poolWorkbenchRecordFor(row).latestPurchaseAt ? formatDateTime(poolWorkbenchRecordFor(row).latestPurchaseAt) : '-' }}
                      </dd>
                    </div>
                  </dl>

                  <div class="mt-2">
                    <div class="mb-1 flex items-center justify-between gap-2 text-[10px]">
                      <span class="text-gray-500 dark:text-gray-400">
                        {{ t('admin.sharedPool.workbench.recovery') }} {{ formatPoolCost(poolWorkbenchRecordFor(row).recognizedCostMinor) }}
                      </span>
                      <span class="font-medium tabular-nums text-gray-700 dark:text-gray-200">{{ (poolWorkbenchRecordFor(row).progress * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                      <div
                        class="h-full bg-emerald-500 transition-all duration-300"
                        :style="{ width: `${Math.min(1, poolWorkbenchRecordFor(row).progress) * 100}%` }"
                      ></div>
                    </div>
                  </div>

                  <div class="mt-2 flex items-center justify-between gap-2 text-[11px]">
                    <span class="text-gray-500 dark:text-gray-400">
                      {{ t(poolWorkbenchRecordFor(row).remainingCostMinor > 0 ? 'admin.sharedPool.columns.remaining' : 'admin.sharedPool.columns.netProfit') }}
                    </span>
                    <span
                      class="font-semibold tabular-nums"
                      :class="poolWorkbenchRecordFor(row).remainingCostMinor > 0
                        ? 'text-amber-600 dark:text-amber-400'
                        : poolWorkbenchRecordFor(row).netGainMinor >= 0
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : 'text-red-600 dark:text-red-400'"
                    >
                      {{ formatPoolCost(poolWorkbenchRecordFor(row).remainingCostMinor > 0 ? poolWorkbenchRecordFor(row).remainingCostMinor : poolWorkbenchRecordFor(row).netGainMinor) }}
                    </span>
                  </div>
                </button>
              </div>
              <div
                v-if="embedded && showWorkbenchSecondary"
                data-test="account-workbench-secondary"
                class="mt-3 flex min-w-0 flex-wrap items-center gap-x-4 gap-y-2 border-t border-gray-200 pt-3 text-[11px] text-gray-500 dark:border-dark-700 dark:text-gray-400"
              >
                <span class="min-w-0 truncate" :title="accountUploader(row)">
                  {{ t('admin.sharedPool.columns.uploader') }}: {{ accountUploader(row) }}
                </span>
                <span v-if="importBatchID(row)" class="min-w-0 truncate" :title="importBatchID(row)">
                  {{ t('admin.accounts.importBatchGroup') }}: {{ importBatchID(row) }}
                </span>
                <AccountGroupsCell v-if="!authStore.isSimpleMode" :groups="row.groups" :max-display="2" />
              </div>
            </div>
          </template>
          <template #cell-pool_record="{ row }">
            <button
              v-if="!isImportBatchRow(row)"
              type="button"
              class="min-h-11 rounded-lg px-2 py-1 text-left transition-colors hover:bg-emerald-50 dark:hover:bg-emerald-900/20"
              @click="emit('pool-record', row)"
            >
              <template v-if="hasPoolCost(row)">
                <span class="block max-w-36 truncate text-xs font-medium text-emerald-700 dark:text-emerald-300">
                  {{ poolRecordFor(row.id)?.purchase_source_name || t('admin.sharedPool.actions.poolRecord') }}
                </span>
                <span class="block text-xs tabular-nums text-gray-500 dark:text-gray-400">
                  {{ formatPoolCost(row.pool_net_cost_minor) }} · {{ poolRecoveryLabel(row) }}
                </span>
              </template>
              <span v-else class="text-xs font-medium text-amber-600 dark:text-amber-300">
                {{ t('admin.sharedPool.intake.pending') }}
              </span>
            </button>
          </template>
          <template #cell-proxy="{ row }">
            <div v-if="!isImportBatchRow(row)" class="flex flex-col gap-1">
              <div v-if="row.proxy" class="flex items-center gap-2">
                <span class="text-sm text-gray-700 dark:text-gray-300">{{ row.proxy.name }}</span>
                <span v-if="row.proxy.country_code" class="text-xs text-gray-500 dark:text-gray-400">
                  ({{ row.proxy.country_code }})
                </span>
              </div>
              <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
              <div v-if="row.proxy && row.proxy.expires_at" class="flex items-center gap-2 text-xs">
                <span class="text-gray-600 dark:text-gray-300">{{ formatDateTime(row.proxy.expires_at) }}</span>
                <span :class="proxyExpiryBadge(row.proxy)">{{ proxyExpiryText(row.proxy) }}</span>
              </div>
              <div v-if="row.proxy_fallback_origin_id" class="flex items-center gap-1">
                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" :title="t('admin.accounts.fallbackActiveTip', { origin: row.proxy_fallback_origin_name })">
                  {{ t('admin.accounts.fallbackActive') }}
                </span>
                <button class="text-xs px-1.5 py-0.5 rounded border border-gray-300 dark:border-dark-600 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-dark-700" @click="onRevertFallback(row)">{{ t('admin.accounts.revertProxy') }}</button>
              </div>
            </div>
          </template>
          <template #cell-rate_multiplier="{ row }">
            <span v-if="!isImportBatchRow(row)" class="text-sm font-mono text-gray-700 dark:text-gray-300">
              {{ (row.rate_multiplier ?? 1).toFixed(2) }}x
            </span>
          </template>
          <template #header-upstream_billing_rate="{ column }">
            <div class="flex flex-wrap items-center justify-end gap-1">
              <span>{{ column.label }}</span>
              <span @click.stop>
                <HelpTooltip :content="t('admin.accounts.upstreamBilling.trustWarning')" width-class="w-80" />
              </span>
            </div>
          </template>
          <template #cell-upstream_billing_rate="{ row }">
            <UpstreamBillingRateCell
              v-if="!isImportBatchRow(row)"
              :account="row"
              :global-probe-enabled="upstreamBillingProbeGloballyEnabled"
              :now="upstreamBillingNow"
              :probing="probingUpstreamBilling.has(row.id)"
              @probe="handleProbeUpstreamBilling(row)"
            />
          </template>
          <template #cell-priority="{ row, value }">
            <span v-if="!isImportBatchRow(row)" class="text-sm text-gray-700 dark:text-gray-300">{{ value }}</span>
          </template>
          <template #header-scheduler_score="{ column }">
            <div class="flex items-center">
              <span>{{ column.label }}</span>
              <HelpTooltip :content="t('admin.accounts.schedulerScore.hint')" width-class="w-80" />
            </div>
          </template>
          <template #cell-scheduler_score="{ row }">
            <div v-if="!isImportBatchRow(row) && getSchedulerScoreRows(row).length" class="flex min-w-[7rem] flex-col gap-0.5 font-mono text-[11px] leading-4">
              <div
                v-for="score in getSchedulerScoreRows(row)"
                :key="String(score.group_id)"
                class="flex items-center gap-1 whitespace-nowrap text-gray-700 dark:text-gray-300"
                :title="`${formatSchedulerScoreGroup(score)} / ${formatSchedulerScore(score.base_score)} / ${formatStickySchedulerScore(score)}`"
              >
                <span class="max-w-[4.75rem] truncate text-gray-500 dark:text-dark-400">{{ formatSchedulerScoreGroup(score) }}</span>
                <span class="text-gray-300 dark:text-gray-600">/</span>
                <span>{{ formatSchedulerScore(score.base_score) }}</span>
                <span class="text-gray-300 dark:text-gray-600">/</span>
                <span class="text-primary-700 dark:text-primary-300">{{ formatStickySchedulerScore(score) }}</span>
              </div>
            </div>
            <span v-else-if="!isImportBatchRow(row)" class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>
          <template #cell-last_used_at="{ row, value }">
            <span v-if="!isImportBatchRow(row)" class="text-sm text-gray-500 dark:text-dark-400">{{ formatRelativeTime(value) }}</span>
          </template>
          <template #cell-created_at="{ row, value }">
            <span v-if="!isImportBatchRow(row)" class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-expires_at="{ row, value }">
            <div v-if="!isImportBatchRow(row)" class="flex flex-col items-start gap-1">
              <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatExpiresAt(value) }}</span>
              <div v-if="isExpired(value) || (row.auto_pause_on_expired && value)" class="flex items-center gap-1">
                <span
                  v-if="isExpired(value)"
                  class="inline-flex items-center rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                >
                  {{ t('admin.accounts.expired') }}
                </span>
                <span
                  v-if="row.auto_pause_on_expired && value"
                  class="inline-flex items-center rounded-md bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                >
                  {{ t('admin.accounts.autoPauseOnExpired') }}
                </span>
              </div>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <button
              v-if="isImportBatchRow(row)"
              type="button"
              class="btn btn-secondary min-h-11 px-3 text-xs"
              :disabled="isImportBatchLoading(row.batchID)"
              :aria-expanded="expandedImportBatches.has(row.batchID)"
              @click="toggleImportBatch(row.batchID)"
            >
              <Icon :name="expandedImportBatches.has(row.batchID) ? 'chevronDown' : 'chevronRight'" size="sm" />
              {{ expandedImportBatches.has(row.batchID) ? t('admin.accounts.collapseImportBatch') : t('admin.accounts.expandImportBatch') }}
            </button>
            <div v-else data-test="account-workbench-actions" class="flex items-center gap-1">
              <button @click="handleEdit(row)" class="flex min-h-11 min-w-11 flex-col items-center justify-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400">
                <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" /></svg>
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <button @click="openMenu(row)" class="flex min-h-11 min-w-11 flex-col items-center justify-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:hover:bg-dark-700 dark:hover:text-white">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 12a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM12.75 12a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM18.75 12a.75.75 0 11-1.5 0 .75.75 0 011.5 0z" /></svg>
                <span class="text-xs">{{ t('common.more') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
        </div>
          </section>
        </div>
      </template>
      <template #pagination><Pagination v-if="pagination.total > 0" :page="pagination.page" :total="pagination.total" :page-size="pagination.page_size" @update:page="handlePageChange" @update:pageSize="handlePageSizeChange" /></template>
    </TablePageLayout>
    <CreateAccountModal :show="showCreate" :proxies="proxies" :groups="groups" @close="showCreate = false" @created="handleAccountCreated" />
    <EditAccountModal :show="showEdit" :account="edAcc" :proxies="proxies" :groups="groups" @close="showEdit = false" @updated="handleAccountUpdated" @submitted="handleApprovalSubmitted" />
    <ReAuthAccountModal :show="showReAuth" :account="reAuthAcc" @close="closeReAuthModal" @reauthorized="handleAccountUpdated" />
    <AccountTestModal :show="showTest" :account="testingAcc" @close="closeTestModal" />
    <AccountStatsModal :show="showStats" :account="statsAcc" @close="closeStatsModal" />
    <ScheduledTestsPanel :show="showSchedulePanel" :account-id="scheduleAcc?.id ?? null" :model-options="scheduleModelOptions" @close="closeSchedulePanel" />
    <AccountActionMenu :show="menu.show" :account="menu.acc" @close="menu.show = false" @edit="handleEdit" @test="handleTest" @stats="handleViewStats" @schedule="handleSchedule" @duplicate="handleDuplicateAccount" @reauth="handleReAuth" @refresh-token="handleRefresh" @recover-state="handleRecoverState" @reset-quota="handleResetQuota" @set-privacy="handleSetPrivacy" @create-spark-shadow="handleCreateSparkShadow" @credential="openCredentialRequest" @pool-record="emit('pool-record', $event)" @delete="handleDelete" />
    <BaseDialog
      :show="showAccountToolsDialog"
      :title="t('admin.accounts.toolsDialog.title')"
      width="wide"
      :close-on-click-outside="true"
      @close="showAccountToolsDialog = false"
    >
      <div class="space-y-6">
        <section>
          <h4 class="dialog-section-title">{{ t('admin.accounts.dataActions') }}</h4>
          <div class="dialog-action-grid">
            <button class="dialog-action-button" @click="openSyncFromCrs"><Icon name="sync" size="md" class="text-blue-600" />{{ t('admin.accounts.syncFromCrs') }}</button>
            <button class="dialog-action-button" @click="openImportData"><Icon name="upload" size="md" class="text-emerald-600" />{{ t('admin.accounts.dataImport') }}</button>
            <button class="dialog-action-button" @click="openExportDataDialogFromMenu">
              <Icon name="download" size="md" class="text-violet-600" />
              {{ selIds.length ? t('admin.accounts.dataExportSelected') : t('admin.accounts.dataExport') }}
              <span v-if="selIds.length" class="ml-auto rounded-full bg-primary-100 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">{{ selIds.length }}</span>
            </button>
          </div>
        </section>
        <section>
          <h4 class="dialog-section-title">{{ t('admin.accounts.toolActions') }}</h4>
          <div class="dialog-action-grid">
            <button class="dialog-action-button" @click="openErrorPassthrough"><Icon name="shield" size="md" class="text-amber-600" />{{ t('admin.errorPassthrough.title') }}</button>
            <button class="dialog-action-button" @click="openTLSFingerprintProfiles"><Icon name="lock" size="md" class="text-gray-600" />{{ t('admin.tlsFingerprintProfiles.title') }}</button>
          </div>
        </section>
        <section v-if="embedded" class="space-y-4">
          <h4 class="dialog-section-title">{{ t('admin.accounts.display.title') }}</h4>
          <div>
            <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('admin.accounts.display.view') }}</p>
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <button v-for="view in workbenchViews" :key="view" type="button" :class="['display-choice', workbenchView === view ? 'display-choice-active' : '']" @click="setWorkbenchView(view)">
                {{ t(`admin.accounts.display.${view}`) }}
              </button>
            </div>
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="display-setting-row">
              <span>{{ t('admin.accounts.display.secondary') }}</span>
              <input v-model="showWorkbenchSecondary" type="checkbox" class="h-5 w-5 rounded border-gray-300 text-primary-600 focus:ring-primary-500" @change="saveWorkbenchDisplay" />
            </label>
            <label class="display-setting-row">
              <span>{{ t('admin.accounts.display.density') }}</span>
              <select v-model="workbenchDensity" class="input min-h-11 w-36" @change="saveWorkbenchDisplay">
                <option value="comfortable">{{ t('admin.accounts.display.comfortable') }}</option>
                <option value="compact">{{ t('admin.accounts.display.compact') }}</option>
              </select>
            </label>
            <label class="display-setting-row sm:col-span-2">
              <span>{{ t('admin.accounts.display.order') }}</span>
              <select v-model="workbenchModuleOrder" class="input min-h-11 w-44" @change="saveWorkbenchDisplay">
                <option value="usage">{{ t('admin.accounts.display.usageFirst') }}</option>
                <option value="finance">{{ t('admin.accounts.display.financeFirst') }}</option>
              </select>
            </label>
          </div>
          <button type="button" class="btn btn-secondary min-h-11" @click="resetWorkbenchDisplay">{{ t('admin.accounts.display.reset') }}</button>
        </section>
        <section v-else>
          <h4 class="dialog-section-title">{{ t('admin.accounts.viewColumns') }}</h4>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <button v-for="col in toggleableColumns" :key="col.key" type="button" class="display-choice flex items-center justify-between" @click="toggleColumn(col.key)">
              <span class="truncate">{{ col.label }}</span>
              <Icon v-if="isColumnVisible(col.key)" name="check" size="sm" class="text-primary-500" />
            </button>
          </div>
        </section>
      </div>
    </BaseDialog>
    <SyncFromCrsModal :show="showSync" @close="showSync = false" @synced="reload" />
    <ImportDataModal
      :show="showImportData"
      :proxies="proxies"
      :groups="groups"
      @close="showImportData = false"
      @imported="handleDataImported"
    />
    <BulkEditAccountModal
      :show="showBulkEdit"
      :account-ids="selIds"
      :selected-platforms="selPlatforms"
      :selected-types="selTypes"
      :target="bulkEditTarget ?? undefined"
      :proxies="proxies"
      :groups="groups"
      @close="showBulkEdit = false"
      @updated="handleBulkUpdated"
    />
    <TempUnschedStatusModal :show="showTempUnsched" :account="tempUnschedAcc" @close="showTempUnsched = false" @reset="handleTempUnschedReset" />
    <BaseDialog :show="showDeleteDialog" :title="t('admin.accounts.deleteAccount')" width="normal" @close="closeDeleteDialog">
      <form id="account-delete-form" class="space-y-4" @submit.prevent="confirmDelete">
        <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.accounts.deleteConfirm', { name: deletingAcc?.name }) }}</p>
        <p class="rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">{{ t('admin.sharedPool.delete.hardDeleteHint') }}</p>
      </form>
      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:justify-end sm:gap-3">
          <button type="button" class="btn btn-secondary" :disabled="deleteSubmitting" @click="closeDeleteDialog">{{ t('common.cancel') }}</button>
          <button type="submit" form="account-delete-form" class="btn btn-danger" :disabled="deleteSubmitting">
            <LoadingSpinner v-if="deleteSubmitting" size="sm" />
            {{ t('common.delete') }}
          </button>
        </div>
      </template>
    </BaseDialog>
    <ConfirmDialog :show="showCreateShadowDialog" :title="t('admin.accounts.createSparkShadow')" :message="t('admin.accounts.createSparkShadowConfirm', { name: creatingShadowAcc?.name })" @confirm="confirmCreateSparkShadow" @cancel="showCreateShadowDialog = false" />
    <ConfirmDialog :show="showExportDataDialog" :title="t('admin.accounts.dataExport')" :message="t('admin.accounts.dataExportConfirmMessage')" :confirm-text="t('admin.accounts.dataExportConfirm')" :cancel-text="t('common.cancel')" @confirm="handleExportData" @cancel="showExportDataDialog = false">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.dataExportIncludeProxies') }}</p>
    </ConfirmDialog>
    <ErrorPassthroughRulesModal :show="showErrorPassthrough" @close="showErrorPassthrough = false" />
    <TLSFingerprintProfilesModal :show="showTLSFingerprintProfiles" @close="showTLSFingerprintProfiles = false" />
    <TotpStepUpDialog :controller="accountExportStepUp" />
    <TotpStepUpDialog :controller="credentialStepUp" />

    <BaseDialog
      :show="showApprovalCenter"
      :title="t('admin.sharedPool.approval.title')"
      width="extra-wide"
      @close="closeApprovalCenter"
    >
      <div class="space-y-4">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.subtitle') }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.sharedPool.approval.queueSummary', { pending: pendingApprovalCount, highRisk: highRiskApprovalCount }) }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="flex rounded-lg border border-gray-200 p-1 dark:border-dark-700" role="tablist">
              <button
                v-for="scope in approvalScopes"
                :key="scope"
                type="button"
                class="min-h-9 rounded-md px-3 text-xs font-medium transition-colors"
                :class="approvalScope === scope ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
                role="tab"
                :aria-selected="approvalScope === scope"
                @click="changeApprovalScope(scope)"
              >
                {{ t(`admin.sharedPool.approval.scopes.${scope}`) }}
                <span v-if="scope === 'reviewable' && pendingApprovalCount" class="ml-1">{{ pendingApprovalCount }}</span>
              </button>
            </div>
            <label for="approval-status-filter" class="sr-only">{{ t('admin.sharedPool.approval.status') }}</label>
            <select v-if="approvalScope !== 'reviewable'" id="approval-status-filter" v-model="approvalStatusFilter" class="input min-h-11 w-36" @change="changeApprovalFilter">
              <option value="pending">{{ t('admin.sharedPool.approval.pending') }}</option>
              <option value="approved">{{ t('admin.sharedPool.approval.approved') }}</option>
              <option value="rejected">{{ t('admin.sharedPool.approval.rejected') }}</option>
              <option value="">{{ t('common.all') }}</option>
            </select>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="approvalsLoading" @click="loadApprovals">
              <Icon name="refresh" size="sm" :class="approvalsLoading ? 'animate-spin' : ''" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </div>
        </div>

        <DataTable :columns="approvalColumns" :data="approvals" row-key="id" :loading="approvalsLoading">
          <template #cell-action_type="{ row }">
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ approvalActionLabel(row.action_type) }}
            </span>
          </template>
          <template #cell-account_name="{ row }">
            <div class="min-w-0">
              <p class="max-w-48 truncate font-medium text-gray-900 dark:text-white" :title="row.account_name">
                {{ row.account_name || `#${row.account_id}` }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">#{{ row.account_id }}</p>
            </div>
          </template>
          <template #cell-requested_by_email="{ row }">
            <span class="text-sm text-gray-600 dark:text-gray-300">{{ row.requested_by_email || `#${row.requested_by_user_id}` }}</span>
          </template>
          <template #cell-status="{ row }">
            <StatusBadge :status="approvalStatusBadge(row.status)" :label="approvalStatusLabel(row.status)" />
          </template>
          <template #cell-requested_at="{ row }">
            <span class="whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ formatDateTime(row.requested_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex flex-wrap justify-end gap-2">
              <button type="button" class="btn btn-secondary min-h-11 px-3 text-xs" @click="selectApproval(row)">
                {{ t('admin.sharedPool.approval.details') }}
              </button>
              <button
                v-if="canRevealApproval(row)"
                type="button"
                class="btn btn-primary min-h-11 px-3 text-xs"
                :disabled="approvalActionID === row.id"
                @click="revealCredential(row)"
              >
                <LoadingSpinner v-if="approvalActionID === row.id" size="sm" />
                {{ t('admin.sharedPool.approval.revealOnce') }}
              </button>
            </div>
          </template>
          <template #empty>
            <div class="py-8 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.empty') }}</div>
          </template>
        </DataTable>
        <Pagination
          v-if="approvalPagination.total > approvalPagination.page_size"
          :page="approvalPagination.page"
          :total="approvalPagination.total"
          :page-size="approvalPagination.page_size"
          @update:page="handleApprovalPageChange"
        />

        <section v-if="selectedApproval" class="rounded-xl border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-900/40">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h4 class="font-semibold text-gray-900 dark:text-white">
                {{ approvalActionLabel(selectedApproval.action_type) }} · {{ selectedApproval.account_name || `#${selectedApproval.account_id}` }}
              </h4>
              <p class="mt-1 text-sm font-medium text-gray-700 dark:text-gray-300">{{ approvalBusinessSummary }}</p>
            </div>
            <StatusBadge :status="approvalStatusBadge(selectedApproval.status)" :label="approvalStatusLabel(selectedApproval.status)" />
          </div>

          <dl class="mt-3 grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <dt class="font-medium text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.requester') }}</dt>
              <dd class="mt-0.5 break-all text-gray-900 dark:text-white">{{ selectedApproval.requested_by_email || `#${selectedApproval.requested_by_user_id}` }}</dd>
            </div>
            <div>
              <dt class="font-medium text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.requestedAt') }}</dt>
              <dd class="mt-0.5 text-gray-900 dark:text-white">{{ formatDateTime(selectedApproval.requested_at) }}</dd>
            </div>
            <div>
              <dt class="font-medium text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.triggerReason') }}</dt>
              <dd class="mt-0.5 text-gray-900 dark:text-white">{{ approvalTriggerReason(selectedApproval.action_type) }}</dd>
            </div>
            <div>
              <dt class="font-medium text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.requestNote') }}</dt>
              <dd class="mt-0.5 break-words text-gray-900 dark:text-white">{{ approvalRequestReason(selectedApproval.reason) }}</dd>
            </div>
          </dl>

          <div v-if="approvalBusinessGroups.length" class="mt-4 grid gap-3 lg:grid-cols-2">
            <article v-for="group in approvalBusinessGroups" :key="group.key" class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800/60">
              <h5 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ approvalGroupLabel(group.key) }} · {{ group.items.length }}
              </h5>
              <div class="mt-2 space-y-2">
                <div v-for="change in group.items.slice(0, 3)" :key="change.key" class="rounded-md bg-gray-50 px-3 py-2 text-sm dark:bg-dark-900/50">
                  <p class="font-medium text-gray-800 dark:text-gray-100">{{ approvalFieldLabel(change.key) }}</p>
                  <p class="mt-0.5 break-all text-gray-600 dark:text-gray-300">
                    {{ approvalBusinessChangeValue(change, 'before') }} → {{ approvalBusinessChangeValue(change, 'after') }}
                  </p>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ approvalBusinessChangeImpact(change) }}</p>
                </div>
                <details v-if="group.items.length > 3" class="text-sm">
                  <summary class="cursor-pointer py-1 font-medium text-primary-600 dark:text-primary-400">
                    {{ t('admin.sharedPool.approval.showRemaining', { count: group.items.length - 3 }) }}
                  </summary>
                  <div class="mt-2 space-y-2">
                    <div v-for="change in group.items.slice(3)" :key="change.key" class="rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-900/50">
                      <p class="font-medium text-gray-800 dark:text-gray-100">{{ approvalFieldLabel(change.key) }}</p>
                      <p class="mt-0.5 break-all text-gray-600 dark:text-gray-300">{{ approvalBusinessChangeValue(change, 'before') }} → {{ approvalBusinessChangeValue(change, 'after') }}</p>
                      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ approvalBusinessChangeImpact(change) }}</p>
                    </div>
                  </div>
                </details>
              </div>
            </article>
          </div>

          <div v-if="approvalBusinessImpacts.length" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900/60 dark:bg-red-950/20">
            <p class="text-sm font-semibold text-red-800 dark:text-red-300">{{ t('admin.sharedPool.approval.deleteImpactTitle') }}</p>
            <dl class="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-4">
              <div v-for="impact in approvalBusinessImpacts" :key="impact.key" class="rounded-md bg-white/70 px-2 py-2 dark:bg-dark-900/50">
                <dt class="text-xs text-gray-500 dark:text-gray-400">{{ approvalImpactLabel(impact.key) }}</dt>
                <dd class="mt-0.5 font-semibold tabular-nums text-gray-900 dark:text-white">{{ impact.count }}</dd>
              </div>
            </dl>
          </div>

          <details v-if="approvalDiffRows.length" class="mt-4 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
            <summary class="cursor-pointer bg-gray-100 px-3 py-2 text-sm font-medium text-gray-700 dark:bg-dark-800 dark:text-gray-200">
              {{ t('admin.sharedPool.approval.technicalDetails') }} · {{ approvalDiffRows.length }}
            </summary>
            <div>
            <div class="hidden grid-cols-[minmax(8rem,0.8fr)_minmax(0,1fr)_minmax(0,1fr)] bg-gray-100 px-3 py-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-gray-400 sm:grid">
              <span>{{ t('admin.sharedPool.approval.field') }}</span>
              <span>{{ t('admin.sharedPool.approval.before') }}</span>
              <span>{{ t('admin.sharedPool.approval.after') }}</span>
            </div>
            <div
              v-for="row in approvalDiffRows"
              :key="row.field"
              class="grid grid-cols-1 gap-2 border-t border-gray-200 px-3 py-3 text-sm dark:border-dark-700 sm:grid-cols-[minmax(8rem,0.8fr)_minmax(0,1fr)_minmax(0,1fr)] sm:gap-3 sm:py-2"
            >
              <span class="break-all text-sm font-medium text-gray-700 dark:text-gray-300"><span class="font-semibold sm:hidden">{{ t('admin.sharedPool.approval.field') }}: </span>{{ row.field }}</span>
              <span class="break-all whitespace-pre-wrap text-gray-500 dark:text-gray-400"><span class="font-semibold sm:hidden">{{ t('admin.sharedPool.approval.before') }}: </span>{{ row.before }}</span>
              <span class="break-all whitespace-pre-wrap text-gray-900 dark:text-white"><span class="font-semibold sm:hidden">{{ t('admin.sharedPool.approval.after') }}: </span>{{ row.after }}</span>
            </div>
            </div>
          </details>

          <div v-if="canReviewApproval(selectedApproval)" class="sticky bottom-0 z-10 -mx-4 mt-4 space-y-3 border-t border-gray-200 bg-white px-4 pb-1 pt-4 shadow-[0_-8px_16px_-16px_rgba(0,0,0,0.5)] dark:border-dark-700 dark:bg-dark-900">
            <label for="approval-decision-reason" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.sharedPool.approval.decisionReason') }}
            </label>
            <textarea
              id="approval-decision-reason"
              v-model.trim="approvalDecisionReason"
              class="input min-h-24 w-full resize-y"
              :placeholder="t('admin.sharedPool.approval.decisionReasonHint')"
            ></textarea>
            <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <button type="button" class="btn btn-danger min-h-11 px-4" :disabled="approvalActionID !== null || !approvalDecisionReason.trim()" @click="decideApproval('reject')">
                <LoadingSpinner v-if="approvalDecisionInProgress === 'reject'" size="sm" />
                {{ t('admin.sharedPool.approval.reject') }}
              </button>
              <button type="button" class="btn btn-primary min-h-11 px-4" :disabled="approvalActionID !== null" @click="decideApproval('approve')">
                <LoadingSpinner v-if="approvalDecisionInProgress === 'approve'" size="sm" />
                {{ t('admin.sharedPool.approval.approve') }}
              </button>
            </div>
          </div>
          <p v-else-if="normalizeApprovalStatus(selectedApproval.status) === 'pending'" class="mt-4 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
            {{ t('admin.sharedPool.approval.selfReviewBlocked') }}
          </p>
        </section>
      </div>
    </BaseDialog>

    <BaseDialog
      :show="showCredentialDialog"
      :title="t('admin.sharedPool.approval.credentialTitle', { name: credentialAccount?.name || '' })"
      width="normal"
      @close="closeCredentialDialog"
    >
      <form id="credential-access-form" class="space-y-4" @submit.prevent="submitCredentialRequest">
        <template v-if="credentialReveal">
          <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200">
            {{ t('admin.sharedPool.approval.revealWarning') }}
          </div>
          <pre class="max-h-80 overflow-auto rounded-lg bg-gray-950 p-4 text-xs leading-5 text-gray-100">{{ credentialRevealJSON }}</pre>
        </template>
        <template v-else>
          <p v-if="!authStore.user?.is_primary_admin" class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.sharedPool.approval.credentialHint') }}</p>
          <label v-if="!authStore.user?.is_primary_admin" for="credential-purpose" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.sharedPool.approval.purpose') }}
          </label>
          <textarea
            v-if="!authStore.user?.is_primary_admin"
            id="credential-purpose"
            v-model.trim="credentialPurpose"
            class="input min-h-24 w-full resize-y"
            :placeholder="t('admin.sharedPool.approval.purposeHint')"
          ></textarea>
        </template>
      </form>
      <template #footer>
        <FormDialogActions
          v-if="!credentialReveal"
          form="credential-access-form"
          :submitting="credentialSubmitting"
          :disabled="!authStore.user?.is_primary_admin && !credentialPurpose.trim()"
          :submit-text="authStore.user?.is_primary_admin ? t('admin.sharedPool.approval.verifyAndReveal') : t('admin.sharedPool.approval.submit')"
          :cancel-text="t('common.close')"
          @cancel="closeCredentialDialog"
        />
        <div v-else class="flex justify-end">
          <button type="button" class="btn btn-secondary min-h-11 px-4" @click="closeCredentialDialog">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>
  </component>
</template>

<script setup lang="ts">
import { ref, reactive, computed, nextTick, onMounted, onUnmounted, toRaw, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { adminAPI } from '@/api/admin'
import { useTableLoader } from '@/composables/useTableLoader'
import { useSwipeSelect, type SwipeSelectVirtualContext } from '@/composables/useSwipeSelect'
import { useTableSelection } from '@/composables/useTableSelection'
import { useStepUp, isStepUpBlocked, isStepUpCancelled, stepUpBlockReason } from '@/composables/useStepUp'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import FormDialogActions from '@/components/common/FormDialogActions.vue'
import { CreateAccountModal, EditAccountModal, BulkEditAccountModal, SyncFromCrsModal, TempUnschedStatusModal } from '@/components/account'
import AccountTableActions from '@/components/admin/account/AccountTableActions.vue'
import AccountTableFilters from '@/components/admin/account/AccountTableFilters.vue'
import AccountBulkActionsBar from '@/components/admin/account/AccountBulkActionsBar.vue'
import AccountActionMenu from '@/components/admin/account/AccountActionMenu.vue'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import ReAuthAccountModal from '@/components/admin/account/ReAuthAccountModal.vue'
import AccountTestModal from '@/components/admin/account/AccountTestModal.vue'
import AccountStatsModal from '@/components/admin/account/AccountStatsModal.vue'
import ScheduledTestsPanel from '@/components/admin/account/ScheduledTestsPanel.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import AccountTodayStatsCell from '@/components/account/AccountTodayStatsCell.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import UpstreamBillingRateCell from '@/components/account/UpstreamBillingRateCell.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import ErrorPassthroughRulesModal from '@/components/admin/ErrorPassthroughRulesModal.vue'
import TLSFingerprintProfilesModal from '@/components/admin/TLSFingerprintProfilesModal.vue'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import { proxyExpiryBadgeClass, proxyExpiryLabelKey } from '@/utils/proxyExpiry'
import { extractApiErrorMessage } from '@/utils/apiError'
import { sanitizeUrl } from '@/utils/url'
import type { Account, AccountPlatform, AccountSchedulerGroupScore, AccountType, Proxy as AccountProxy, AdminGroup, WindowStats, ClaudeModel } from '@/types'
import type { AccountBatchStatusSummary, AccountImportBatchSummary, AccountListRow } from '@/api/admin/accounts'
import type { PoolApproval, PoolApprovalBusinessChange, PoolApprovalBusinessSummary, PoolApprovalScope, PoolApprovalStatus, PoolCredentialReveal, SharedPoolAccountCost } from '@/api/admin/sharedPool'

type AccountWorkbenchScope = 'all' | 'standalone' | 'uploader' | 'batch'
type AccountWorkbenchContext = {
  scope: AccountWorkbenchScope
  import_batch_id?: string
  uploader_user_id?: number | string
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
const props = withDefaults(defineProps<{
  embedded?: boolean
  poolRecords?: Record<number, SharedPoolAccountCost>
  initialWorkbenchScope?: 'all' | 'standalone' | 'uploader' | 'batch'
  initialImportBatchId?: string
  initialUploaderUserId?: number | string
  initialWorkbenchContext?: Partial<AccountWorkbenchContext>
}>(), {
  embedded: false,
  poolRecords: () => ({}),
  initialWorkbenchScope: 'all',
  initialImportBatchId: '',
  initialUploaderUserId: '',
  initialWorkbenchContext: () => ({})
})
const embedded = props.embedded
const emit = defineEmits<{
  'pool-record': [account: Account]
  'trace-account': [accountId: number]
  'workbench-context': [context: AccountWorkbenchContext]
  refreshed: []
  'pool-create-request': []
  'pool-import-request': []
  'pool-created': [accounts: Array<Pick<Account, 'id' | 'name'>>]
  'pool-imported': [accounts: Array<{ id: number; name: string }>]
}>()

const { t, te } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const proxies = ref<AccountProxy[]>([])
const groups = ref<AdminGroup[]>([])
const uploaderOptions = ref<Array<{ value: number; label: string }>>([])
const expandedImportBatches = ref(new Set<string>())
const workbenchScope = ref<AccountWorkbenchScope>(
  props.initialWorkbenchContext.import_batch_id || props.initialImportBatchId
    ? 'batch'
    : props.initialWorkbenchContext.scope || props.initialWorkbenchScope
)
const selectedWorkbenchBatchID = ref(props.initialWorkbenchContext.import_batch_id || props.initialImportBatchId)
const selectedWorkbenchUploaderID = ref<number | string>(
  props.initialWorkbenchContext.uploader_user_id ?? props.initialUploaderUserId
)
type WorkbenchNavigatorMode = 'uploader' | 'batch'
const workbenchNavigatorMode = ref<WorkbenchNavigatorMode>(workbenchScope.value === 'batch' ? 'batch' : 'uploader')
const workbenchBatches = ref<AccountImportBatchSummary[]>([])
const workbenchNavigatorSearch = ref('')
const mobileWorkbenchNavigatorExpanded = ref(false)
const workbenchSidebarRef = ref<HTMLElement | null>(null)
const workbenchNavigatorLoading = ref(false)
const workbenchNavigatorLoadingMore = ref(false)
const workbenchAccountTotal = ref(0)
const workbenchStandaloneCount = ref(0)
const workbenchBatchTotal = ref(0)
const workbenchBatchPage = ref(1)
const workbenchBatchPages = ref(1)
let workbenchNavigatorRevision = 0
let workbenchNavigatorPreviousFocus: HTMLElement | null = null
let workbenchNavigatorBodyLocked = false
let workbenchNavigatorMobileMedia: MediaQueryList | null = null
const accountTableRef = ref<HTMLElement | null>(null)
const dataTableRef = ref<InstanceType<typeof DataTable> | null>(null)
type AccountBulkEditTarget =
  | {
      mode: 'selected'
      accountIds: number[]
      selectedPlatforms: AccountPlatform[]
      selectedTypes: AccountType[]
    }
  | {
      mode: 'filtered'
      filters: {
        platform?: string
        type?: string
        status?: string
        group?: string
        search?: string
        privacy_mode?: string
        uploader_user_id?: number
        uploader_unassigned?: boolean
        import_batch_id?: string
        import_batch_scope?: 'standalone' | 'batched'
        sort_by?: string
        sort_order?: AccountSortOrder
      }
      previewCount: number
      selectedPlatforms: AccountPlatform[]
      selectedTypes: AccountType[]
    }
const knownAccountsByID = new Map<number, Account>()
const selPlatforms = computed<AccountPlatform[]>(() => {
  const platforms = new Set(
    selIds.value
      .map(id => knownAccountsByID.get(id))
      .filter((account): account is Account => Boolean(account))
      .map(a => a.platform)
  )
  return [...platforms]
})
const selTypes = computed<AccountType[]>(() => {
  const types = new Set(
    selIds.value
      .map(id => knownAccountsByID.get(id))
      .filter((account): account is Account => Boolean(account))
      .map(a => a.type)
  )
  return [...types]
})
const showCreate = ref(false)
const showEdit = ref(false)
const showSync = ref(false)
const showImportData = ref(false)
const showExportDataDialog = ref(false)
const showBulkEdit = ref(false)
const bulkEditTarget = ref<AccountBulkEditTarget | null>(null)
const showTempUnsched = ref(false)
const showDeleteDialog = ref(false)
const showCreateShadowDialog = ref(false)
const showReAuth = ref(false)
const showTest = ref(false)
const showStats = ref(false)
const showErrorPassthrough = ref(false)
const showTLSFingerprintProfiles = ref(false)
const showApprovalCenter = ref(false)
const approvalsLoading = ref(false)
const approvals = ref<PoolApproval[]>([])
const approvalScope = ref<PoolApprovalScope>('reviewable')
const approvalScopes: PoolApprovalScope[] = ['reviewable', 'mine', 'processed']
const approvalStatusFilter = ref<PoolApprovalStatus | ''>('pending')
const approvalPagination = reactive({ page: 1, page_size: 20, total: 0 })
const pendingApprovalCount = ref(0)
const highRiskApprovalCount = ref(0)
const selectedApproval = ref<PoolApproval | null>(null)
const approvalDecisionReason = ref('')
const approvalActionID = ref<number | null>(null)
const approvalDecisionInProgress = ref<'approve' | 'reject' | null>(null)
const showCredentialDialog = ref(false)
const credentialAccount = ref<{ id: number; name: string } | null>(null)
const credentialPurpose = ref('')
const credentialReveal = ref<PoolCredentialReveal | null>(null)
const credentialSubmitting = ref(false)
const credentialStepUp = useStepUp()
let credentialClearTimer: ReturnType<typeof setTimeout> | null = null
const edAcc = ref<Account | null>(null)
const tempUnschedAcc = ref<Account | null>(null)
const deletingAcc = ref<Account | null>(null)
const deleteSubmitting = ref(false)
const creatingShadowAcc = ref<Account | null>(null)
const reAuthAcc = ref<Account | null>(null)
const testingAcc = ref<Account | null>(null)
const statsAcc = ref<Account | null>(null)
const showSchedulePanel = ref(false)
const scheduleAcc = ref<Account | null>(null)
const scheduleModelOptions = ref<SelectOption[]>([])
const togglingSchedulable = ref<number | null>(null)
const menu = reactive<{ show: boolean; acc: Account | null }>({ show: false, acc: null })
const exportingData = ref(false)
const probingUpstreamBilling = reactive(new Set<number>())
const upstreamBillingProbeGloballyEnabled = ref<boolean | undefined>(undefined)
const upstreamBillingNow = ref(Date.now())
let lastUpstreamBillingSortRefreshMinute = -1
useIntervalFn(() => { upstreamBillingNow.value = Date.now() }, 60_000)

const approvalColumns = computed(() => [
  { key: 'action_type', label: t('admin.sharedPool.approval.type') },
  { key: 'account_name', label: t('admin.sharedPool.columns.account') },
  { key: 'requested_by_email', label: t('admin.sharedPool.approval.requester') },
  { key: 'status', label: t('admin.sharedPool.approval.status') },
  { key: 'requested_at', label: t('admin.sharedPool.approval.requestedAt') },
  { key: 'actions', label: t('admin.sharedPool.columns.actions'), class: 'text-right' }
])

const credentialRevealJSON = computed(() => JSON.stringify(credentialReveal.value?.credentials ?? {}, null, 2))
const SENSITIVE_APPROVAL_FIELD = /(credential|token|secret|password|api[_-]?key|cookie|authorization)/i
const APPROVAL_FIELD_LABELS: Record<string, string> = {
  name: 'name', notes: 'notes', type: 'type', proxy_id: 'proxy', concurrency: 'concurrency', priority: 'priority',
  rate_multiplier: 'rateMultiplier', load_factor: 'loadFactor', status: 'status', group_ids: 'groups', expires_at: 'expiresAt',
  auto_pause_on_expired: 'autoPauseOnExpired', provider_identity: 'providerIdentity', contributor_user_id: 'contributor',
  created_by_user_id: 'uploader', cost_sharing_enabled: 'costSharingEnabled', credential_keys: 'credentialKeys',
  extra_keys: 'extraKeys', delete_account: 'deleteAccount', cost_entry_id: 'costEntry', cost_payer_user_id: 'payer',
  cost_purchase_source_id: 'purchaseSource', cost_entry_type: 'costType', cost_original_amount: 'costAmount',
  cost_currency: 'currency', cost_fx_rate: 'fxRate', cost_service_start: 'serviceStart', cost_service_end: 'serviceEnd',
  cost_warranty_end: 'warrantyEnd', cost_paid_at: 'paidAt', cost_order_no: 'orderNo', cost_purchase_url: 'purchaseUrl',
  cost_note: 'note', cost_expected_token_count: 'expectedTokens'
}
const APPROVAL_ACCOUNT_TYPE_KEYS: Record<string, string> = {
  oauth: 'admin.accounts.types.oauth',
  'setup-token': 'admin.accounts.setupToken',
  apikey: 'admin.accounts.apiKey',
  bedrock: 'admin.accounts.bedrockLabel'
}
const approvalFieldLabel = (path: string) => {
  const field = path.replace(/^fields\./, '')
  const key = APPROVAL_FIELD_LABELS[field]
  return key ? t(`admin.sharedPool.approval.fieldLabels.${key}`) : field
}
const approvalValuesEqual = (before: unknown, after: unknown) => {
  if (Object.is(before, after)) return true
  if (typeof before !== 'object' || typeof after !== 'object' || before === null || after === null) return false
  return JSON.stringify(before) === JSON.stringify(after)
}
const formatApprovalValue = (value: unknown, field: string): string => {
  if (value === undefined || value === null || value === '') return '-'
  if (field.endsWith('_keys') && Array.isArray(value)) return value.join(', ') || '-'
  if (SENSITIVE_APPROVAL_FIELD.test(field)) return t('admin.sharedPool.approval.sensitiveValue')
  if (typeof value === 'boolean') return t(value ? 'common.yes' : 'common.no')
  const normalizedField = field.replace(/^fields\./, '')
  if (normalizedField === 'status' && typeof value === 'string') {
    const key = `admin.accounts.status.${value}`
    return te(key) ? t(key) : value
  }
  if (normalizedField === 'type' && typeof value === 'string') {
    const key = APPROVAL_ACCOUNT_TYPE_KEYS[value]
    return key && te(key) ? t(key) : value
  }
  if (normalizedField === 'cost_entry_type' && typeof value === 'string') {
    const key = `admin.sharedPool.entryTypes.${value}`
    return te(key) ? t(key) : value
  }
  if ((normalizedField.endsWith('_at') || normalizedField.endsWith('_start') || normalizedField.endsWith('_end')) && (typeof value === 'string' || typeof value === 'number')) {
    return formatDateTime(typeof value === 'number' ? new Date(value * 1000) : value)
  }
  if (typeof value !== 'object') return String(value)
  return JSON.stringify(value, (key, nested) => SENSITIVE_APPROVAL_FIELD.test(key) ? t('admin.sharedPool.approval.sensitiveValue') : nested, 2)
}

const approvalDiffRows = computed(() => {
  const changes = selectedApproval.value?.changes
  if (!changes || typeof changes !== 'object') return []

  const rows: Array<{ field: string; before: string; after: string }> = []
  const before = changes.before
  const after = changes.after
  if (before && after && typeof before === 'object' && typeof after === 'object') {
    const beforeRecord = before as Record<string, unknown>
    const afterRecord = after as Record<string, unknown>
    for (const field of new Set([...Object.keys(beforeRecord), ...Object.keys(afterRecord)])) {
      if (approvalValuesEqual(beforeRecord[field], afterRecord[field])) continue
      rows.push({
        field: approvalFieldLabel(field),
        before: formatApprovalValue(beforeRecord[field], field),
        after: formatApprovalValue(afterRecord[field], field)
      })
    }
    return rows
  }

  for (const [section, rawSection] of Object.entries(changes)) {
    if (section === 'business') continue
    if (rawSection && typeof rawSection === 'object' && !Array.isArray(rawSection)) {
      for (const [field, rawChange] of Object.entries(rawSection as Record<string, unknown>)) {
        const path = `${section}.${field}`
        if (rawChange && typeof rawChange === 'object' && !Array.isArray(rawChange)) {
          const change = rawChange as Record<string, unknown>
          const hasPair = 'before' in change || 'after' in change || 'old' in change || 'new' in change
          if (hasPair) {
            const previousValue = change.before ?? change.old
            const nextValue = change.after ?? change.new
            if (approvalValuesEqual(previousValue, nextValue)) continue
            rows.push({
              field: approvalFieldLabel(path),
              before: formatApprovalValue(previousValue, path),
              after: formatApprovalValue(nextValue, path)
            })
            continue
          }
        }
        rows.push({ field: approvalFieldLabel(path), before: '-', after: formatApprovalValue(rawChange, path) })
      }
      continue
    }
    rows.push({ field: approvalFieldLabel(section), before: '-', after: formatApprovalValue(rawSection, section) })
  }
  return rows
})

const approvalBusiness = computed<PoolApprovalBusinessSummary | null>(() => selectedApproval.value?.changes?.business ?? null)
const approvalBusinessGroups = computed(() => approvalBusiness.value?.groups ?? [])
const approvalBusinessImpacts = computed(() => approvalBusiness.value?.impacts ?? [])
const approvalGroupLabel = (key: string) => t(`admin.sharedPool.approval.groups.${key}`)
const approvalImpactLabel = (key: string) => t(`admin.sharedPool.approval.impacts.${key}`)
const approvalBusinessChangeValue = (change: PoolApprovalBusinessChange, side: 'before' | 'after') => (
  change.sensitive
    ? (side === 'after' ? t('admin.sharedPool.approval.updated') : t('admin.sharedPool.approval.sensitiveValue'))
    : formatApprovalValue(change[side], change.key)
)
const approvalBusinessChangeImpact = (change: PoolApprovalBusinessChange) => t(`admin.sharedPool.approval.effects.${change.impact}`)

const showAccountToolsDialog = ref(false)
type WorkbenchView = 'runtime' | 'finance' | 'full'
type WorkbenchDensity = 'comfortable' | 'compact'
type WorkbenchModuleOrder = 'usage' | 'finance'
const WORKBENCH_DISPLAY_STORAGE_KEY = 'account-workbench-display-v1'
const workbenchViews: WorkbenchView[] = ['runtime', 'finance', 'full']
const workbenchView = ref<WorkbenchView>('runtime')
const workbenchDensity = ref<WorkbenchDensity>('compact')
const workbenchModuleOrder = ref<WorkbenchModuleOrder>('usage')
const showWorkbenchSecondary = ref(false)

const saveWorkbenchDisplay = () => {
  try {
    localStorage.setItem(WORKBENCH_DISPLAY_STORAGE_KEY, JSON.stringify({
      view: workbenchView.value,
      density: workbenchDensity.value,
      moduleOrder: workbenchModuleOrder.value,
      showSecondary: showWorkbenchSecondary.value
    }))
  } catch (error) {
    console.error('Failed to save account workbench display:', error)
  }
}

const loadWorkbenchDisplay = () => {
  try {
    const saved = localStorage.getItem(WORKBENCH_DISPLAY_STORAGE_KEY)
    if (!saved) return
    const value = JSON.parse(saved) as { view?: string; density?: string; moduleOrder?: string; showSecondary?: boolean }
    if (workbenchViews.includes(value.view as WorkbenchView)) workbenchView.value = value.view as WorkbenchView
    if (value.density === 'comfortable' || value.density === 'compact') workbenchDensity.value = value.density
    if (value.moduleOrder === 'usage' || value.moduleOrder === 'finance') workbenchModuleOrder.value = value.moduleOrder
    if (typeof value.showSecondary === 'boolean') showWorkbenchSecondary.value = value.showSecondary
  } catch (error) {
    console.error('Failed to load account workbench display:', error)
  }
}

const setWorkbenchView = (view: WorkbenchView) => {
  workbenchView.value = view
  saveWorkbenchDisplay()
}

const resetWorkbenchDisplay = () => {
  workbenchView.value = 'runtime'
  workbenchDensity.value = 'compact'
  workbenchModuleOrder.value = 'usage'
  showWorkbenchSecondary.value = false
  saveWorkbenchDisplay()
}
const hiddenColumns = reactive<Set<string>>(new Set())
const DEFAULT_HIDDEN_COLUMNS = embedded
  ? [
      'id', 'uploader', 'today_stats', 'pool_record', 'proxy', 'notes', 'priority', 'scheduler_score',
      'rate_multiplier', 'upstream_billing_rate', 'last_used_at', 'created_at', 'expires_at'
    ]
  : [
      'id', 'today_stats', 'groups', 'proxy', 'notes', 'priority', 'scheduler_score',
      'rate_multiplier', 'upstream_billing_rate', 'last_used_at', 'expires_at'
    ]
const HIDDEN_COLUMNS_KEY = 'account-hidden-columns'
const HIDDEN_COLUMNS_VERSION_KEY = 'account-hidden-columns-version'
const HIDDEN_COLUMNS_CURRENT_VERSION = embedded ? 'shared-pool-workbench-v2' : 'scheduler-score-hidden-by-default'

// Sorting settings
const ACCOUNT_SORT_STORAGE_KEY = 'account-table-sort-v2'
type AccountSortOrder = 'asc' | 'desc'
type AccountSortState = {
  sort_by: string
  sort_order: AccountSortOrder
}
const ACCOUNT_SORTABLE_KEYS = new Set([
  'name',
  'status',
  'schedulable',
  'created_at'
])
const loadInitialAccountSortState = (): AccountSortState => {
  const initialSortKey = props.initialWorkbenchContext.sort_by || 'created_at'
  const fallback: AccountSortState = {
    sort_by: ACCOUNT_SORTABLE_KEYS.has(initialSortKey) ? initialSortKey : 'created_at',
    sort_order: props.initialWorkbenchContext.sort_order === 'asc' ? 'asc' : 'desc'
  }
  if (props.initialWorkbenchContext.sort_by) return fallback
  try {
    const raw = localStorage.getItem(ACCOUNT_SORT_STORAGE_KEY)
    if (!raw) return fallback
    const parsed = JSON.parse(raw) as { key?: string; order?: string }
    const key = typeof parsed.key === 'string' ? parsed.key : ''
    if (!ACCOUNT_SORTABLE_KEYS.has(key)) return fallback
    return {
      sort_by: key,
      sort_order: parsed.order === 'desc' ? 'desc' : 'asc'
    }
  } catch {
    return fallback
  }
}
const sortState = reactive<AccountSortState>(loadInitialAccountSortState())

// Auto refresh settings
const showAutoRefreshDropdown = ref(false)
const autoRefreshDropdownRef = ref<HTMLElement | null>(null)
const AUTO_REFRESH_STORAGE_KEY = 'account-auto-refresh'
const autoRefreshIntervals = [5, 10, 15, 30] as const
const autoRefreshEnabled = ref(false)
const autoRefreshIntervalSeconds = ref<(typeof autoRefreshIntervals)[number]>(30)
const autoRefreshCountdown = ref(0)
const autoRefreshFetching = ref(false)
let autoRefreshRequestRevision = 0
const AUTO_REFRESH_SILENT_WINDOW_MS = 15000
const autoRefreshSilentUntil = ref(0)
const hasPendingListSync = ref(false)
const todayStatsByAccountId = ref<Record<string, WindowStats>>({})
const todayStatsLoading = ref(false)
const todayStatsError = ref<string | null>(null)
const todayStatsReqSeq = ref(0)
const pendingTodayStatsRefresh = ref(false)
const usageManualRefreshToken = ref(0)

const buildDefaultTodayStats = (): WindowStats => ({
  requests: 0,
  tokens: 0,
  cost: 0,
  standard_cost: 0,
  user_cost: 0
})

const refreshTodayStatsBatch = async () => {
  // Why this checks both columns:
  // - today_stats column shows dedicated today's metrics.
  // - usage column also embeds today's stats for Key/Bedrock rows.
  // So we only skip fetching when BOTH columns are hidden.
  if (hiddenColumns.has('today_stats') && hiddenColumns.has('usage')) {
    todayStatsLoading.value = false
    todayStatsError.value = null
    return
  }

  const accountIDs = accounts.value.map(account => account.id)
  const reqSeq = ++todayStatsReqSeq.value
  if (accountIDs.length === 0) {
    todayStatsByAccountId.value = {}
    todayStatsError.value = null
    todayStatsLoading.value = false
    return
  }

  todayStatsLoading.value = true
  todayStatsError.value = null

  try {
    const result = await adminAPI.accounts.getBatchTodayStats(accountIDs)
    if (reqSeq !== todayStatsReqSeq.value) return
    const serverStats = result.stats ?? {}
    const nextStats: Record<string, WindowStats> = {}
    for (const accountID of accountIDs) {
      const key = String(accountID)
      nextStats[key] = serverStats[key] ?? buildDefaultTodayStats()
    }
    todayStatsByAccountId.value = nextStats
  } catch (error) {
    if (reqSeq !== todayStatsReqSeq.value) return
    todayStatsError.value = 'Failed'
    console.error('Failed to load account today stats:', error)
  } finally {
    if (reqSeq === todayStatsReqSeq.value) {
      todayStatsLoading.value = false
    }
  }
}

const autoRefreshIntervalLabel = (sec: number) => {
  if (sec === 5) return t('admin.accounts.refreshInterval5s')
  if (sec === 10) return t('admin.accounts.refreshInterval10s')
  if (sec === 15) return t('admin.accounts.refreshInterval15s')
  if (sec === 30) return t('admin.accounts.refreshInterval30s')
  return `${sec}s`
}

const formatSchedulerScore = (value: unknown): string => {
  const num = Number(value)
  if (!Number.isFinite(num)) return '-'
  return num.toFixed(6).replace(/\.?0+$/, '')
}

const formatStickySchedulerScore = (score: AccountSchedulerGroupScore): string => {
  if (!score) return '-'
  if (score.sticky_score_infinity) return '+∞'
  return formatSchedulerScore(score.sticky_score)
}

const getSchedulerScoreRows = (account: Account): AccountSchedulerGroupScore[] => {
  const groupRows = Array.isArray(account.scheduler_scores)
    ? account.scheduler_scores.filter(score => score.group_id != null)
    : []
  if (groupRows.length) return groupRows
  // 未分组账号没有分组维度分数，回退展示后端返回的基础分
  if (account.scheduler_score) {
    return [{ group_id: null, ...account.scheduler_score }]
  }
  return []
}

const formatSchedulerScoreGroup = (score: AccountSchedulerGroupScore): string => {
  if ('group_name' in score && score.group_name) return score.group_name
  if ('group_id' in score && score.group_id != null) return `#${score.group_id}`
  return t('admin.accounts.schedulerScore.ungrouped')
}

const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      parsed.forEach(key => {
        hiddenColumns.add(key)
      })
      if (localStorage.getItem(HIDDEN_COLUMNS_VERSION_KEY) !== HIDDEN_COLUMNS_CURRENT_VERSION) {
        hiddenColumns.add('scheduler_score')
        if (embedded) {
          DEFAULT_HIDDEN_COLUMNS.forEach(key => hiddenColumns.add(key))
          hiddenColumns.delete('capacity')
          hiddenColumns.delete('platform_type')
          hiddenColumns.delete('status')
          hiddenColumns.delete('schedulable')
          hiddenColumns.delete('groups')
          hiddenColumns.delete('usage')
        }
        localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
        localStorage.setItem(HIDDEN_COLUMNS_VERSION_KEY, HIDDEN_COLUMNS_CURRENT_VERSION)
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach(key => {
        hiddenColumns.add(key)
      })
      localStorage.setItem(HIDDEN_COLUMNS_VERSION_KEY, HIDDEN_COLUMNS_CURRENT_VERSION)
    }
  } catch (e) {
    console.error('Failed to load saved columns:', e)
    DEFAULT_HIDDEN_COLUMNS.forEach(key => {
      hiddenColumns.add(key)
    })
  }
}

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
    localStorage.setItem(HIDDEN_COLUMNS_VERSION_KEY, HIDDEN_COLUMNS_CURRENT_VERSION)
  } catch (e) {
    console.error('Failed to save columns:', e)
  }
}

const loadSavedAutoRefresh = () => {
  try {
    const saved = localStorage.getItem(AUTO_REFRESH_STORAGE_KEY)
    if (!saved) return
    const parsed = JSON.parse(saved) as { enabled?: boolean; interval_seconds?: number }
    autoRefreshEnabled.value = parsed.enabled === true
    const interval = Number(parsed.interval_seconds)
    if (autoRefreshIntervals.includes(interval as any)) {
      autoRefreshIntervalSeconds.value = interval as any
    }
  } catch (e) {
    console.error('Failed to load saved auto refresh settings:', e)
  }
}

const saveAutoRefreshToStorage = () => {
  try {
    localStorage.setItem(
      AUTO_REFRESH_STORAGE_KEY,
      JSON.stringify({
        enabled: autoRefreshEnabled.value,
        interval_seconds: autoRefreshIntervalSeconds.value
      })
    )
  } catch (e) {
    console.error('Failed to save auto refresh settings:', e)
  }
}

if (typeof window !== 'undefined') {
  loadSavedColumns()
  loadSavedAutoRefresh()
  loadWorkbenchDisplay()
}

const setAutoRefreshEnabled = (enabled: boolean) => {
  autoRefreshEnabled.value = enabled
  saveAutoRefreshToStorage()
  if (enabled) {
    autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
    resumeAutoRefresh()
  } else {
    pauseAutoRefresh()
    autoRefreshCountdown.value = 0
  }
}

const setAutoRefreshInterval = (seconds: (typeof autoRefreshIntervals)[number]) => {
  autoRefreshIntervalSeconds.value = seconds
  saveAutoRefreshToStorage()
  if (autoRefreshEnabled.value) {
    autoRefreshCountdown.value = seconds
  }
}

const toggleColumn = (key: string) => {
  const wasHidden = hiddenColumns.has(key)
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
  if ((key === 'today_stats' || key === 'usage') && wasHidden) {
    refreshTodayStatsBatch().catch((error) => {
      console.error('Failed to load account today stats after showing column:', error)
    })
  }
  if (key === 'scheduler_score') {
    // The server only returns scheduler scores when this column is visible, so reload the current page immediately.
    syncAccountListDerivedParams()
    load().catch((error) => {
      console.error('Failed to reload accounts after toggling scheduler score column:', error)
    })
  }
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)
const shouldIncludeSchedulerScore = () => isColumnVisible('scheduler_score')
const syncAccountListDerivedParams = () => {
  // Keep every load path, including auto-refresh and sorting, aligned with the current column visibility.
  const requestParams = params as any
  requestParams.include_scheduler_score = shouldIncludeSchedulerScore() ? '1' : '0'
}

const workbenchRequestFilters = (raw: Record<string, unknown>) => {
  const request = { ...raw } as Record<string, unknown>
  delete request.import_batch_id
  delete request.import_batch_scope
  if (workbenchScope.value === 'uploader') {
    request.uploader_user_id = selectedWorkbenchUploaderID.value
    request.import_batch_scope = 'batched'
  }
  return request
}

const fetchAccountWorkbenchRows = async (
  page: number,
  pageSize: number,
  rawFilters: Record<string, unknown>,
  options?: { signal?: AbortSignal }
) => {
  if (!embedded) return adminAPI.accounts.listRows(page, pageSize, rawFilters, options)

  const filters = workbenchRequestFilters(rawFilters)
  if (workbenchScope.value === 'batch' && selectedWorkbenchBatchID.value) {
    const result = await adminAPI.accounts.listImportBatch(
      selectedWorkbenchBatchID.value,
      page,
      pageSize,
      filters
    )
    return { ...result, items: result.items.map(account => ({ kind: 'account' as const, account })) }
  }
  if (workbenchScope.value === 'standalone') filters.import_batch_scope = 'standalone'
  const result = await adminAPI.accounts.list(page, pageSize, filters, options)
  return { ...result, items: result.items.map(account => ({ kind: 'account' as const, account })) }
}

const {
  items: accountListRows,
  loading,
  params,
  pagination,
  load: baseLoad,
  reload: baseReload,
  debouncedReload: baseDebouncedReload,
  handlePageChange: baseHandlePageChange,
  handlePageSizeChange: baseHandlePageSizeChange
} = useTableLoader<AccountListRow, any>({
  fetchFn: fetchAccountWorkbenchRows,
  initialParams: {
    platform: props.initialWorkbenchContext.platform || '',
    type: props.initialWorkbenchContext.type || '',
    status: props.initialWorkbenchContext.status || '',
    privacy_mode: props.initialWorkbenchContext.privacy_mode || '',
    uploader_user_id: '',
    group: props.initialWorkbenchContext.group || '',
    search: props.initialWorkbenchContext.search || '',
    include_scheduler_score: shouldIncludeSchedulerScore() ? '1' : '0',
    include_pool_metrics: '1',
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order
  }
})
if (embedded) {
  pagination.page = Math.max(1, Number(props.initialWorkbenchContext.page || 1))
  if (props.initialWorkbenchContext.page_size) {
    pagination.page_size = Math.max(1, Number(props.initialWorkbenchContext.page_size))
  }
}

const accounts = ref<Account[]>([])
const visibleResultAccountCount = computed(() => embedded ? pagination.total : accountListRows.value.reduce(
  (total, row) => total + (row.kind === 'account' ? 1 : row.batch.matched_count),
  0
))
const visibleResultBatchCount = computed(() => accountListRows.value.filter(row => row.kind === 'import_batch').length)
const accountUploader = (account: Account): string => account.uploader_username || account.uploader_email || '-'
const importBatchID = (account: Account): string => {
  const value = account.extra?.import_batch_id
  return typeof value === 'string' ? value : ''
}
const completeImportBatchAccounts = ref(new Map<string, Account[]>())
const importBatchLoads = new Map<string, Promise<Account[]>>()
const loadingImportBatchKeys = ref(new Set<string>())
const staleImportBatchRequest = new Error('stale import batch request')
let importBatchFilterRevision = 0
const isImportBatchLoading = (id: string) => loadingImportBatchKeys.value.has(`${importBatchFilterRevision}:${id}`)
const invalidateImportBatchCache = (collapse = false) => {
  importBatchFilterRevision++
  completeImportBatchAccounts.value = new Map()
  if (collapse) expandedImportBatches.value = new Set()
}
const buildImportBatchFilters = () => ({
  platform: String(params.platform || ''),
  type: String(params.type || ''),
  status: String(params.status || ''),
  group: String(params.group || ''),
  search: String(params.search || ''),
  privacy_mode: String(params.privacy_mode || ''),
  uploader_user_id: params.uploader_user_id || undefined,
  include_pool_metrics: '1',
  include_scheduler_score: shouldIncludeSchedulerScore() ? '1' : '0',
  sort_by: String(params.sort_by || ''),
  sort_order: params.sort_order === 'desc' ? 'desc' as const : 'asc' as const
})
const loadCompleteImportBatch = (id: string): Promise<Account[]> => {
  const cached = completeImportBatchAccounts.value.get(id)
  if (cached) return Promise.resolve(cached)
  const revision = importBatchFilterRevision
  const loadKey = `${revision}:${id}`
  const active = importBatchLoads.get(loadKey)
  if (active) return active
  const filters = buildImportBatchFilters()

  const request = (async () => {
    const pageSize = 100
    const members = new Map<number, Account>()
    let page = 1
    let pages = 1
    do {
      if (revision !== importBatchFilterRevision) throw staleImportBatchRequest
      const result = await adminAPI.accounts.listImportBatch(id, page, pageSize, filters)
      if (revision !== importBatchFilterRevision) throw staleImportBatchRequest
      result.items.forEach(account => {
        members.set(account.id, account)
        knownAccountsByID.set(account.id, account)
      })
      pages = Math.max(result.pages || Math.ceil(result.total / pageSize), 1)
      page++
    } while (page <= pages)

    if (revision !== importBatchFilterRevision) throw staleImportBatchRequest
    const batchAccounts = [...members.values()]
    const next = new Map(completeImportBatchAccounts.value)
    next.set(id, batchAccounts)
    completeImportBatchAccounts.value = next
    return batchAccounts
  })()
  importBatchLoads.set(loadKey, request)
  loadingImportBatchKeys.value = new Set(loadingImportBatchKeys.value).add(loadKey)
  void request.finally(() => {
    importBatchLoads.delete(loadKey)
    const next = new Set(loadingImportBatchKeys.value)
    next.delete(loadKey)
    loadingImportBatchKeys.value = next
  }).catch(() => undefined)
  return request
}
type ImportBatchRow = {
  __rowKind: 'import-batch'
  id: string
  batchID: string
  accounts: Account[]
  uploader: string
  createdAt: string
  names: string
  matchedCount: number
  totalCount: number
  schedulableCount: number
  status: AccountBatchStatusSummary
}
type AccountTableRow = Account | ImportBatchRow
const isImportBatchRow = (row: AccountTableRow): row is ImportBatchRow => '__rowKind' in row && row.__rowKind === 'import-batch'
const currentBatchIDs = computed(() => new Set(
  accountListRows.value
    .filter((row): row is Extract<AccountListRow, { kind: 'import_batch' }> => row.kind === 'import_batch')
    .map(row => row.batch.id)
))
const isImportBatchChild = (row: AccountTableRow): row is Account => {
  if (isImportBatchRow(row)) return false
  const id = importBatchID(row)
  return Boolean(id && currentBatchIDs.value.has(id))
}
const accountByID = computed(() => new Map(accounts.value.map(account => [account.id, account])))
const importBatchRow = (batch: AccountImportBatchSummary): ImportBatchRow => ({
  __rowKind: 'import-batch',
  id: `import-batch:${batch.id}`,
  batchID: batch.id,
  accounts: completeImportBatchAccounts.value.get(batch.id) || [],
  uploader: batch.uploader_username || batch.uploader_email || '-',
  createdAt: batch.created_at,
  names: batch.names.join('、'),
  matchedCount: batch.matched_count,
  totalCount: batch.total_count,
  schedulableCount: batch.schedulable_count,
  status: batch.status
})
const accountTableRows = computed<AccountTableRow[]>(() => {
  const rows: AccountTableRow[] = []
  for (const logicalRow of accountListRows.value) {
    if (logicalRow.kind === 'account') {
      rows.push(accountByID.value.get(logicalRow.account.id) || logicalRow.account)
      continue
    }
    const batch = importBatchRow(logicalRow.batch)
    rows.push(batch)
    if (expandedImportBatches.value.has(batch.batchID)) rows.push(...batch.accounts)
  }
  return rows
})
const syncPageAccounts = () => {
  const next = new Map<number, Account>()
  for (const logicalRow of accountListRows.value) {
    if (logicalRow.kind === 'account') {
      next.set(logicalRow.account.id, logicalRow.account)
      continue
    }
    for (const account of completeImportBatchAccounts.value.get(logicalRow.batch.id) || []) {
      next.set(account.id, account)
    }
  }
  accounts.value = [...next.values()]
}
watch([accountListRows, completeImportBatchAccounts], syncPageAccounts, { immediate: true })
const pageBatchLoading = computed(() => [...currentBatchIDs.value].some(isImportBatchLoading))
const batchStatusItems = (status: AccountBatchStatusSummary) => {
  const definitions: Array<{ key: keyof AccountBatchStatusSummary; label: string; className: string }> = [
    { key: 'error', label: t('admin.accounts.status.error'), className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300' },
    { key: 'rate_limited', label: t('admin.accounts.status.rateLimited'), className: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300' },
    { key: 'overloaded', label: t('admin.accounts.status.overloaded'), className: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300' },
    { key: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable'), className: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300' },
    { key: 'inactive', label: t('admin.accounts.status.inactive'), className: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300' },
    { key: 'manual_unschedulable', label: t('admin.accounts.status.unschedulable'), className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' },
    { key: 'normal', label: t('admin.accounts.status.active'), className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' }
  ]
  return definitions
    .map(item => ({ ...item, count: status[item.key] }))
    .filter(item => item.count > 0)
}
const importBatchStatusItems = (batch: ImportBatchRow) => batchStatusItems(batch.status)
type WorkbenchUploaderGroup = {
  id: number | string
  label: string
  count: number
  schedulableCount: number
  status: AccountBatchStatusSummary
  batches: AccountImportBatchSummary[]
}
const sortedWorkbenchBatches = computed(() => [...workbenchBatches.value].sort(
  (left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime()
))
const workbenchBatchMatchesSearch = (batch: AccountImportBatchSummary, needle: string) => !needle || [
  batch.id,
  batch.uploader_username,
  batch.uploader_email,
  ...batch.names
].some(value => String(value || '').toLocaleLowerCase().includes(needle))
const workbenchUploaderGroups = computed<WorkbenchUploaderGroup[]>(() => {
  const groups = new Map<string, WorkbenchUploaderGroup>()
  for (const batch of sortedWorkbenchBatches.value) {
    const id = batch.uploader_user_id ?? 'unassigned'
    const key = String(id)
    const group = groups.get(key) || {
      id,
      label: batch.uploader_username || batch.uploader_email || t('admin.sharedPool.ledger.unassignedUploader'),
      count: 0,
      schedulableCount: 0,
      status: {
        normal: 0,
        error: 0,
        inactive: 0,
        rate_limited: 0,
        overloaded: 0,
        temp_unschedulable: 0,
        manual_unschedulable: 0
      },
      batches: []
    }
    group.count += batch.matched_count
    group.schedulableCount += batch.schedulable_count
    for (const key of Object.keys(group.status) as Array<keyof AccountBatchStatusSummary>) {
      group.status[key] += batch.status[key]
    }
    group.batches.push(batch)
    groups.set(key, group)
  }
  return [...groups.values()]
})
const filteredWorkbenchUploaderGroups = computed<WorkbenchUploaderGroup[]>(() => {
  const needle = workbenchNavigatorSearch.value.trim().toLocaleLowerCase()
  if (!needle) return workbenchUploaderGroups.value
  return workbenchUploaderGroups.value
    .map(group => {
      const groupMatches = group.label.toLocaleLowerCase().includes(needle)
      const batches = groupMatches ? group.batches : group.batches.filter(batch => workbenchBatchMatchesSearch(batch, needle))
      return { ...group, batches }
    })
    .filter(group => group.batches.length > 0)
})
const filteredWorkbenchBatches = computed(() => {
  const needle = workbenchNavigatorSearch.value.trim().toLocaleLowerCase()
  const uploaderID = selectedWorkbenchUploaderID.value
  return sortedWorkbenchBatches.value.filter(batch =>
    (uploaderID === '' || String(batch.uploader_user_id ?? 'unassigned') === String(uploaderID)) &&
    workbenchBatchMatchesSearch(batch, needle)
  )
})
const workbenchNavigatorResultCount = computed(() =>
  workbenchNavigatorMode.value === 'uploader'
    ? filteredWorkbenchUploaderGroups.value.length
    : filteredWorkbenchBatches.value.length
)
const selectedWorkbenchBatch = computed(() =>
  workbenchBatches.value.find(batch => batch.id === selectedWorkbenchBatchID.value)
)
const selectedWorkbenchUploader = computed(() =>
  workbenchUploaderGroups.value.find(group => String(group.id) === String(selectedWorkbenchUploaderID.value))
)
const workbenchTitle = computed(() => {
  if (workbenchScope.value === 'standalone') return t('admin.accounts.standaloneImport')
  if (workbenchScope.value === 'uploader') return selectedWorkbenchUploader.value?.label || t('admin.sharedPool.ledger.allUploaders')
  if (workbenchScope.value === 'batch') return selectedWorkbenchBatch.value?.names.join('、') || t('admin.accounts.importBatchGroup')
  return t('admin.sharedPool.ledger.allAccounts')
})
const workbenchSubtitle = computed(() => {
  const batch = selectedWorkbenchBatch.value
  if (workbenchScope.value === 'batch' && batch) {
    const uploader = batch.uploader_username || batch.uploader_email || t('admin.sharedPool.ledger.unassignedUploader')
    return `${uploader} · ${formatDateTime(batch.created_at)} · ${batch.id}`
  }
  return t('admin.accounts.resultSummary', {
    accounts: pagination.total,
    batches: workbenchScope.value === 'all' ? workbenchBatchTotal.value : 0
  })
})
const workbenchNavigatorFilters = () => ({
  platform: String(params.platform || ''),
  type: String(params.type || ''),
  status: String(params.status || ''),
  group: String(params.group || ''),
  search: workbenchNavigatorSearch.value.trim() || String(params.search || ''),
  privacy_mode: String(params.privacy_mode || ''),
  uploader_user_id: params.uploader_user_id || undefined,
  import_batch_scope: 'batched' as const,
  sort_by: 'created_at',
  sort_order: 'desc' as const
})
const loadWorkbenchNavigator = async () => {
  if (!embedded) return
  const revision = ++workbenchNavigatorRevision
  workbenchNavigatorLoading.value = true
  workbenchNavigatorLoadingMore.value = false
  try {
    const filters = workbenchNavigatorFilters()
    const summaryFilters = { ...filters } as Record<string, unknown>
    delete summaryFilters.import_batch_scope
    summaryFilters.search = String(params.search || '')
    const [result, summary, standalone] = await Promise.all([
      adminAPI.accounts.listRows(1, 100, filters),
      adminAPI.accounts.getSelectionSummary(summaryFilters),
      adminAPI.accounts.getSelectionSummary({ ...summaryFilters, import_batch_scope: 'standalone' })
    ])
    if (revision !== workbenchNavigatorRevision) return
    let batches = result.items
      .filter((row): row is Extract<AccountListRow, { kind: 'import_batch' }> => row.kind === 'import_batch')
      .map(row => row.batch)
    if (selectedWorkbenchBatchID.value && !batches.some(batch => batch.id === selectedWorkbenchBatchID.value)) {
      const selected = await adminAPI.accounts.listRows(1, 1, {
        ...filters,
        import_batch_scope: undefined,
        import_batch_id: selectedWorkbenchBatchID.value
      })
      if (revision !== workbenchNavigatorRevision) return
      const selectedBatch = selected.items.find(
        (row): row is Extract<AccountListRow, { kind: 'import_batch' }> => row.kind === 'import_batch'
      )?.batch
      if (selectedBatch) batches = [selectedBatch, ...batches]
    }
    if (
      workbenchScope.value === 'uploader' &&
      selectedWorkbenchUploaderID.value !== '' &&
      !batches.some(batch => String(batch.uploader_user_id ?? 'unassigned') === String(selectedWorkbenchUploaderID.value))
    ) {
      const selected = await adminAPI.accounts.listRows(1, 100, {
        ...filters,
        uploader_user_id: selectedWorkbenchUploaderID.value
      })
      if (revision !== workbenchNavigatorRevision) return
      const known = new Set(batches.map(batch => batch.id))
      batches = [
        ...selected.items
          .filter((row): row is Extract<AccountListRow, { kind: 'import_batch' }> => row.kind === 'import_batch')
          .map(row => row.batch)
          .filter(batch => !known.has(batch.id)),
        ...batches
      ]
    }
    if (revision !== workbenchNavigatorRevision) return
    workbenchBatches.value = batches
    workbenchAccountTotal.value = summary.total
    workbenchStandaloneCount.value = standalone.total
    workbenchBatchTotal.value = result.total
    workbenchBatchPage.value = result.page
    workbenchBatchPages.value = Math.max(1, result.pages || 1)
  } catch (error) {
    console.error('Failed to load account workbench navigator:', error)
  } finally {
    if (revision === workbenchNavigatorRevision) workbenchNavigatorLoading.value = false
  }
}
watch(workbenchNavigatorSearch, (_value, _oldValue, onCleanup) => {
  const timer = window.setTimeout(() => void loadWorkbenchNavigator(), 250)
  onCleanup(() => window.clearTimeout(timer))
})
const loadMoreWorkbenchBatches = async () => {
  if (workbenchNavigatorLoadingMore.value || workbenchBatchPage.value >= workbenchBatchPages.value) return
  const revision = workbenchNavigatorRevision
  workbenchNavigatorLoadingMore.value = true
  try {
    const result = await adminAPI.accounts.listRows(workbenchBatchPage.value + 1, 100, workbenchNavigatorFilters())
    if (revision !== workbenchNavigatorRevision) return
    const known = new Set(workbenchBatches.value.map(batch => batch.id))
    const additions = result.items
      .filter((row): row is Extract<AccountListRow, { kind: 'import_batch' }> => row.kind === 'import_batch')
      .map(row => row.batch)
      .filter(batch => !known.has(batch.id))
    workbenchBatches.value = [...workbenchBatches.value, ...additions]
    workbenchBatchTotal.value = result.total
    workbenchBatchPage.value = result.page
    workbenchBatchPages.value = Math.max(1, result.pages || 1)
  } catch (error) {
    console.error('Failed to load more account workbench batches:', error)
  } finally {
    if (revision === workbenchNavigatorRevision) workbenchNavigatorLoadingMore.value = false
  }
}
const emitWorkbenchContext = () => emit('workbench-context', {
  scope: workbenchScope.value,
  ...(selectedWorkbenchBatchID.value ? { import_batch_id: selectedWorkbenchBatchID.value } : {}),
  ...(selectedWorkbenchUploaderID.value !== '' ? { uploader_user_id: selectedWorkbenchUploaderID.value } : {}),
  page: pagination.page,
  page_size: pagination.page_size,
  search: String(params.search || ''),
  platform: String(params.platform || ''),
  type: String(params.type || ''),
  status: String(params.status || ''),
  group: String(params.group || ''),
  privacy_mode: String(params.privacy_mode || ''),
  sort_by: sortState.sort_by,
  sort_order: sortState.sort_order
})
const setWorkbenchNavigatorMode = (mode: WorkbenchNavigatorMode) => {
  workbenchNavigatorMode.value = mode
}
const clearWorkbenchUploaderBatchFilter = () => {
  selectedWorkbenchUploaderID.value = ''
  if (workbenchScope.value !== 'uploader') return
  workbenchScope.value = 'all'
  clearSelection()
  pagination.page = 1
  emitWorkbenchContext()
  void reload()
}
const copyWorkbenchBatchID = async (id: string) => {
  try {
    await navigator.clipboard.writeText(id)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}
const workbenchNavigatorFocusable = () => Array.from(workbenchSidebarRef.value?.querySelectorAll<HTMLElement>(
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
) || [])
const releaseWorkbenchNavigatorBodyLock = () => {
  if (!workbenchNavigatorBodyLocked) return
  const count = Math.max(0, Number(document.body.dataset.modalOpenCount || 1) - 1)
  if (count) document.body.dataset.modalOpenCount = String(count)
  else {
    delete document.body.dataset.modalOpenCount
    document.body.classList.remove('modal-open')
  }
  workbenchNavigatorBodyLocked = false
}
const closeMobileWorkbenchNavigator = () => {
  mobileWorkbenchNavigatorExpanded.value = false
}
const openMobileWorkbenchNavigator = () => {
  mobileWorkbenchNavigatorExpanded.value = true
}
const handleWorkbenchNavigatorKeydown = (event: KeyboardEvent) => {
  if (!mobileWorkbenchNavigatorExpanded.value || !workbenchNavigatorMobileMedia?.matches) return
  if (event.key === 'Escape') {
    event.preventDefault()
    closeMobileWorkbenchNavigator()
    return
  }
  if (event.key !== 'Tab') return
  const focusable = workbenchNavigatorFocusable()
  if (!focusable.length) {
    event.preventDefault()
    workbenchSidebarRef.value?.focus()
  } else if (event.shiftKey && document.activeElement === focusable[0]) {
    event.preventDefault()
    focusable[focusable.length - 1].focus()
  } else if (!event.shiftKey && document.activeElement === focusable[focusable.length - 1]) {
    event.preventDefault()
    focusable[0].focus()
  }
}
watch(mobileWorkbenchNavigatorExpanded, async (expanded) => {
  if (expanded && workbenchNavigatorMobileMedia?.matches) {
    workbenchNavigatorPreviousFocus = document.activeElement as HTMLElement
    const count = Number(document.body.dataset.modalOpenCount || 0) + 1
    document.body.dataset.modalOpenCount = String(count)
    document.body.classList.add('modal-open')
    workbenchNavigatorBodyLocked = true
    await nextTick()
    ;(workbenchSidebarRef.value?.querySelector<HTMLElement>('input[type="search"]') || workbenchNavigatorFocusable()[0] || workbenchSidebarRef.value)?.focus()
  } else {
    releaseWorkbenchNavigatorBodyLock()
    workbenchNavigatorPreviousFocus?.focus()
    workbenchNavigatorPreviousFocus = null
  }
})
const handleWorkbenchNavigatorMediaChange = (event: MediaQueryListEvent) => {
  if (!event.matches) closeMobileWorkbenchNavigator()
}
const selectWorkbenchScope = (scope: 'all' | 'standalone') => {
  if (workbenchScope.value === scope) {
    closeMobileWorkbenchNavigator()
    return
  }
  workbenchScope.value = scope
  selectedWorkbenchBatchID.value = ''
  selectedWorkbenchUploaderID.value = ''
  clearSelection()
  pagination.page = 1
  closeMobileWorkbenchNavigator()
  emitWorkbenchContext()
  void reload()
}
const selectWorkbenchUploader = (group: WorkbenchUploaderGroup) => {
  workbenchScope.value = 'uploader'
  selectedWorkbenchBatchID.value = ''
  selectedWorkbenchUploaderID.value = group.id
  clearSelection()
  pagination.page = 1
  closeMobileWorkbenchNavigator()
  emitWorkbenchContext()
  void reload()
}
const selectWorkbenchBatch = (batch: AccountImportBatchSummary) => {
  workbenchScope.value = 'batch'
  selectedWorkbenchBatchID.value = batch.id
  selectedWorkbenchUploaderID.value = batch.uploader_user_id ?? 'unassigned'
  clearSelection()
  pagination.page = 1
  closeMobileWorkbenchNavigator()
  emitWorkbenchContext()
  void reload()
}
const toggleImportBatch = async (id: string) => {
  const next = new Set(expandedImportBatches.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    try {
      await loadCompleteImportBatch(id)
      next.add(id)
      void refreshTodayStatsBatch()
    } catch (error) {
      if (error !== staleImportBatchRequest) appStore.showError(extractApiErrorMessage(error, t('common.error')))
    }
  }
  expandedImportBatches.value = next
}

const {
  selectedIds: selIds,
  isSelected,
  setSelectedIds,
  select,
  deselect,
  toggle: toggleSel,
  clear: clearSelection,
  batchUpdate
} = useTableSelection<Account>({
  rows: accounts,
  getId: (account) => account.id
})
const selectedBatchCount = computed(() => new Set(
  selIds.value.map(id => knownAccountsByID.get(id)).filter((account): account is Account => Boolean(account)).map(importBatchID).filter(Boolean)
).size)
watch(accounts, rows => {
  rows.forEach(account => knownAccountsByID.set(account.id, account))
}, { immediate: true })
const completeBatchMembersForSelection = (id: string, matchedCount: number, loaded: Account[] = []) => {
  const members = loaded.length > 0
    ? loaded
    : [...knownAccountsByID.values()].filter(account => importBatchID(account) === id)
  return members.length === matchedCount ? members : []
}
const allVisibleSelected = computed(() => accountListRows.value.length > 0 && accountListRows.value.every(row => {
  if (row.kind === 'account') return isSelected(row.account.id)
  const members = completeBatchMembersForSelection(
    row.batch.id,
    row.batch.matched_count,
    completeImportBatchAccounts.value.get(row.batch.id)
  )
  return members.length > 0 && members.every(account => isSelected(account.id))
}))
const visibleSelectedCount = computed(() => {
  const visibleIDs = new Set<number>()
  for (const row of accountListRows.value) {
    if (row.kind === 'account') {
      if (isSelected(row.account.id)) visibleIDs.add(row.account.id)
      continue
    }
    const members = completeBatchMembersForSelection(
      row.batch.id,
      row.batch.matched_count,
      completeImportBatchAccounts.value.get(row.batch.id)
    )
    for (const account of members) if (isSelected(account.id)) visibleIDs.add(account.id)
  }
  return visibleIDs.size
})
const someVisibleSelected = computed(() => visibleSelectedCount.value > 0 && !allVisibleSelected.value)
const hiddenSelectedCount = computed(() => selIds.value.length - visibleSelectedCount.value)
const loadCurrentPageAccounts = async () => {
  const pageAccounts = new Map<number, Account>()
  const batchRows = accountListRows.value.filter(
    (row): row is Extract<AccountListRow, { kind: 'import_batch' }> => row.kind === 'import_batch'
  )
  for (const row of accountListRows.value) {
    if (row.kind === 'account') pageAccounts.set(row.account.id, row.account)
  }
  const batchAccounts = await Promise.all(batchRows.map(row => loadCompleteImportBatch(row.batch.id)))
  for (const accounts of batchAccounts) {
    for (const account of accounts) pageAccounts.set(account.id, account)
  }
  return [...pageAccounts.values()]
}
const setCurrentPageSelected = async (checked: boolean) => {
  const pageAccounts = await loadCurrentPageAccounts()
  batchUpdate(selected => {
    for (const account of pageAccounts) checked ? selected.add(account.id) : selected.delete(account.id)
  })
}
const togglePageSelection = async () => {
  if (pageBatchLoading.value) return
  try {
    await setCurrentPageSelected(!allVisibleSelected.value)
  } catch (error) {
    if (error !== staleImportBatchRequest) appStore.showError(extractApiErrorMessage(error, t('common.error')))
  }
}
const isImportBatchSelected = (batch: ImportBatchRow) => {
  const members = completeBatchMembersForSelection(batch.batchID, batch.matchedCount, batch.accounts)
  return members.length > 0 && members.every(account => isSelected(account.id))
}
const isImportBatchPartiallySelected = (batch: ImportBatchRow) => {
  const members = completeBatchMembersForSelection(batch.batchID, batch.matchedCount, batch.accounts)
  const selectedCount = members.filter(account => isSelected(account.id)).length
  return selectedCount > 0 && selectedCount < members.length
}
const toggleImportBatchSelection = async (batch: ImportBatchRow, checked: boolean) => {
  try {
    const batchAccounts = await loadCompleteImportBatch(batch.batchID)
    batchUpdate(selected => {
      for (const account of batchAccounts) checked ? selected.add(account.id) : selected.delete(account.id)
    })
  } catch (error) {
    if (error !== staleImportBatchRequest) appStore.showError(extractApiErrorMessage(error, t('common.error')))
  }
}

const swipeVirtualContext: SwipeSelectVirtualContext = {
  getVirtualizer: () => dataTableRef.value?.virtualizer ?? null,
  getSortedData: () => dataTableRef.value?.sortedData ?? accounts.value,
  getRowId: (row: AccountTableRow) => isImportBatchRow(row) ? null : row.id,
}

useSwipeSelect(accountTableRef, {
  isSelected,
  select,
  deselect,
  batchUpdate
}, swipeVirtualContext)

const resetAutoRefreshCache = () => {
  autoRefreshRequestRevision++
}

const isFirstLoad = ref(true)

function markUpstreamBillingSortRefresh() {
  if (sortState.sort_by === 'upstream_billing_rate') {
    lastUpstreamBillingSortRefreshMinute = Math.floor(Date.now() / 60_000)
  }
}

const load = async () => {
  const requestParams = params as any
  markUpstreamBillingSortRefresh()
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = false
  if (isFirstLoad.value) {
    requestParams.lite = '1'
  }
  await Promise.all([baseLoad(), loadWorkbenchNavigator()])
  if (isFirstLoad.value) {
    isFirstLoad.value = false
    delete requestParams.lite
  }
  await refreshTodayStatsBatch()
}

const reload = async (resetPage = true) => {
  invalidateImportBatchCache(true)
  markUpstreamBillingSortRefresh()
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = false
  await Promise.all([
    resetPage ? baseReload() : baseLoad(),
    loadWorkbenchNavigator()
  ])
  await refreshTodayStatsBatch()
}

const applyWorkbenchContext = async (context: Partial<AccountWorkbenchContext>) => {
  const nextBatchID = context.import_batch_id || ''
  const nextScope = nextBatchID ? 'batch' : context.scope || props.initialWorkbenchScope
  const nextUploaderID = nextScope === 'uploader' ? context.uploader_user_id ?? '' : ''
  const nextPage = Math.max(1, Number(context.page || 1))
  const nextPageSize = Math.max(1, Number(context.page_size || 20))
  const nextSortBy = ACCOUNT_SORTABLE_KEYS.has(context.sort_by || '') ? context.sort_by! : 'created_at'
  const nextSortOrder = context.sort_order === 'asc' ? 'asc' : 'desc'
  const nextFilters = {
    search: context.search || '',
    platform: context.platform || '',
    type: context.type || '',
    status: context.status || '',
    group: context.group || '',
    privacy_mode: context.privacy_mode || ''
  }
  const unchanged =
    workbenchScope.value === nextScope &&
    selectedWorkbenchBatchID.value === nextBatchID &&
    String(selectedWorkbenchUploaderID.value) === String(nextUploaderID) &&
    pagination.page === nextPage &&
    pagination.page_size === nextPageSize &&
    sortState.sort_by === nextSortBy &&
    sortState.sort_order === nextSortOrder &&
    Object.entries(nextFilters).every(([key, value]) => String(params[key] || '') === value)
  if (unchanged) return

  workbenchScope.value = nextScope
  selectedWorkbenchBatchID.value = nextBatchID
  selectedWorkbenchUploaderID.value = nextUploaderID
  if (nextScope === 'batch') workbenchNavigatorMode.value = 'batch'
  if (nextScope === 'uploader') workbenchNavigatorMode.value = 'uploader'
  pagination.page = nextPage
  pagination.page_size = nextPageSize
  sortState.sort_by = nextSortBy
  sortState.sort_order = nextSortOrder
  Object.assign(params, nextFilters, { sort_by: nextSortBy, sort_order: nextSortOrder })
  clearSelection()
  await reload(false)
}

watch(
  () => props.initialWorkbenchContext,
  (context) => {
    if (embedded) void applyWorkbenchContext(context)
  },
  { deep: true }
)

const refreshCurrentPage = () => reload(false)

const handleAccountCreated = async (accounts: Array<Pick<Account, 'id' | 'name'>> = []) => {
  await reload()
  if (embedded) {
    emit('pool-created', accounts)
  }
}

const refreshUpstreamBillingSortedList = async (force = false) => {
  if (sortState.sort_by !== 'upstream_billing_rate') return

  const minute = Math.floor(upstreamBillingNow.value / 60_000)
  if (!force && lastUpstreamBillingSortRefreshMinute === minute) return
  lastUpstreamBillingSortRefreshMinute = minute
  try {
    await refreshCurrentPage()
  } catch (error) {
    console.error('Failed to refresh upstream billing sort:', error)
  }
}

const debouncedReload = () => {
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  baseDebouncedReload()
}

const resetSelectionForFilterChange = () => {
  clearSelection()
  pagination.page = 1
}

const handleSearchChange = (value: string) => {
  params.search = value
  resetSelectionForFilterChange()
  invalidateImportBatchCache(true)
  if (embedded) emitWorkbenchContext()
  debouncedReload()
}

const handleFiltersChange = (filters: Record<string, unknown>) => {
  Object.assign(params, filters)
  resetSelectionForFilterChange()
  expandedImportBatches.value = new Set()
  if (embedded) emitWorkbenchContext()
  void reload()
}

const clearFilters = () => {
  Object.assign(params, {
    platform: '',
    type: '',
    status: '',
    privacy_mode: '',
    uploader_user_id: '',
    group: '',
    search: ''
  })
  resetSelectionForFilterChange()
  expandedImportBatches.value = new Set()
  if (embedded) emitWorkbenchContext()
  void reload()
}

const handlePageChange = (page: number) => {
  invalidateImportBatchCache(true)
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  baseHandlePageChange(page)
  if (embedded) emitWorkbenchContext()
}

const handlePageSizeChange = (size: number) => {
  invalidateImportBatchCache(true)
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  baseHandlePageSizeChange(size)
  if (embedded) emitWorkbenchContext()
}

const handleSort = (key: string, order: AccountSortOrder) => {
  invalidateImportBatchCache(true)
  sortState.sort_by = key
  sortState.sort_order = order
  const requestParams = params as any
  requestParams.sort_by = key
  requestParams.sort_order = order
  syncAccountListDerivedParams()
  pagination.page = 1
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  if (embedded) emitWorkbenchContext()
  load()
}
const workbenchSortValue = computed({
  get: () => `${sortState.sort_by}:${sortState.sort_order}`,
  set: (value: string) => {
    const [key, order] = value.split(':')
    handleSort(key || 'created_at', order === 'asc' ? 'asc' : 'desc')
  }
})

watch(loading, (isLoading, wasLoading) => {
  if (wasLoading && !isLoading) {
    upstreamBillingNow.value = Date.now()
  }
  if (wasLoading && !isLoading && pendingTodayStatsRefresh.value) {
    pendingTodayStatsRefresh.value = false
    refreshTodayStatsBatch().catch((error) => {
      console.error('Failed to refresh account today stats after table load:', error)
    })
  }
})

watch(upstreamBillingNow, () => {
  if (sortState.sort_by !== 'upstream_billing_rate' || loading.value) return
  if (typeof document !== 'undefined' && document.hidden) return
  void refreshUpstreamBillingSortedList()
})

const isAnyModalOpen = computed(() => {
  return (
    showCreate.value ||
    showEdit.value ||
    showSync.value ||
    showImportData.value ||
    showExportDataDialog.value ||
    showBulkEdit.value ||
    showTempUnsched.value ||
    showDeleteDialog.value ||
    showReAuth.value ||
    showTest.value ||
    showStats.value ||
    showSchedulePanel.value ||
    showAccountToolsDialog.value ||
    menu.show ||
    showErrorPassthrough.value ||
    showTLSFingerprintProfiles.value
  )
})

const enterAutoRefreshSilentWindow = () => {
  autoRefreshSilentUntil.value = Date.now() + AUTO_REFRESH_SILENT_WINDOW_MS
  autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
}

const inAutoRefreshSilentWindow = () => {
  return Date.now() < autoRefreshSilentUntil.value
}

const syncAccountRefs = (nextAccount: Account) => {
  if (edAcc.value?.id === nextAccount.id) edAcc.value = nextAccount
  if (reAuthAcc.value?.id === nextAccount.id) reAuthAcc.value = nextAccount
  if (tempUnschedAcc.value?.id === nextAccount.id) tempUnschedAcc.value = nextAccount
  if (deletingAcc.value?.id === nextAccount.id) deletingAcc.value = nextAccount
  if (menu.acc?.id === nextAccount.id) menu.acc = nextAccount
}

const refreshAccountsIncrementally = async () => {
  if (autoRefreshFetching.value) return
  syncAccountListDerivedParams()
  const revision = autoRefreshRequestRevision
  autoRefreshFetching.value = true
  try {
    const result = await fetchAccountWorkbenchRows(
      pagination.page,
      pagination.page_size,
      toRaw(params) as {
        platform?: string
        type?: string
        status?: string
        privacy_mode?: string
        group?: string
        search?: string
        uploader_user_id?: number | string
        include_pool_metrics?: string
        sort_by?: string
        sort_order?: AccountSortOrder
      }
    )
    if (revision !== autoRefreshRequestRevision || pageBatchLoading.value) return
    for (const row of result.items) {
      if (row.kind === 'account') syncAccountRefs(row.account)
    }
    const nextBatchIDs = new Set(result.items
      .filter((row): row is Extract<AccountListRow, { kind: 'import_batch' }> => row.kind === 'import_batch')
      .map(row => row.batch.id))
    const expandedBatchIDs = [...expandedImportBatches.value].filter(id => nextBatchIDs.has(id))
    importBatchFilterRevision++
    completeImportBatchAccounts.value = new Map(
      [...completeImportBatchAccounts.value].filter(([id]) => nextBatchIDs.has(id) && !expandedBatchIDs.includes(id))
    )
    accountListRows.value = result.items || []
    pagination.total = result.total || 0
    pagination.pages = result.pages || 0
    hasPendingListSync.value = false
    markUpstreamBillingSortRefresh()
    upstreamBillingNow.value = Date.now()

    await Promise.all(expandedBatchIDs.map(async id => {
      try {
        await loadCompleteImportBatch(id)
      } catch (error) {
        if (error !== staleImportBatchRequest) console.error('Failed to refresh expanded import batch:', error)
      }
    }))

    await Promise.all([refreshTodayStatsBatch(), loadWorkbenchNavigator()])
    emit('refreshed')
  } catch (error) {
    console.error('Auto refresh failed:', error)
  } finally {
    autoRefreshFetching.value = false
  }
}

const handleManualRefresh = async () => {
  await Promise.all([load(), loadUpstreamBillingProbeGlobalState()])
  // Force usage cells to refetch /usage on explicit user refresh.
  usageManualRefreshToken.value += 1
  emit('refreshed')
}

const loadUpstreamBillingProbeGlobalState = async () => {
  try {
    const settings = await adminAPI.accounts.getUpstreamBillingProbeSettings()
    upstreamBillingProbeGloballyEnabled.value = settings.enabled
  } catch (error) {
    console.error('Failed to load upstream billing probe settings:', error)
  }
}

const closeAccountToolsDialog = () => {
  showAccountToolsDialog.value = false
}

const openAccountToolsDialog = () => {
  showAutoRefreshDropdown.value = false
  showAccountToolsDialog.value = true
}

const openSyncFromCrs = () => {
  closeAccountToolsDialog()
  showSync.value = true
}

const openImportData = () => {
  closeAccountToolsDialog()
  if (embedded) {
    emit('pool-import-request')
    return
  }
  showImportData.value = true
}

const handleCreateRequest = () => {
  if (embedded) {
    emit('pool-create-request')
    return
  }
  showCreate.value = true
}

const continueCreateWithPoolDraft = () => { showCreate.value = true }
const continueImportWithPoolDraft = () => { showImportData.value = true }
defineExpose({ continueCreateWithPoolDraft, continueImportWithPoolDraft, reload })

const openExportDataDialogFromMenu = () => {
  closeAccountToolsDialog()
  openExportDataDialog()
}

const openErrorPassthrough = () => {
  closeAccountToolsDialog()
  showErrorPassthrough.value = true
}

const openTLSFingerprintProfiles = () => {
  closeAccountToolsDialog()
  showTLSFingerprintProfiles.value = true
}

const syncPendingListChanges = async () => {
  hasPendingListSync.value = false
  await load()
  // Keep behavior consistent with manual refresh.
  usageManualRefreshToken.value += 1
}

const { pause: pauseAutoRefresh, resume: resumeAutoRefresh } = useIntervalFn(
  async () => {
    if (!autoRefreshEnabled.value) return
    if (document.hidden) return
    if (loading.value || autoRefreshFetching.value || pageBatchLoading.value) return
    if (isAnyModalOpen.value) return
    if (menu.show || showAccountToolsDialog.value || showAutoRefreshDropdown.value) return
    if (inAutoRefreshSilentWindow()) {
      autoRefreshCountdown.value = Math.max(
        0,
        Math.ceil((autoRefreshSilentUntil.value - Date.now()) / 1000)
      )
      return
    }

    if (autoRefreshCountdown.value <= 0) {
      autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
      await refreshAccountsIncrementally()
      return
    }

    autoRefreshCountdown.value -= 1
  },
  1000,
  { immediate: false }
)

// Fresh billing/quota snapshots are authoritative. Imported credential tiers
// can be stale, so they remain fallbacks together with legacy plan_type fields.
function getAccountPlanType(row: any): string | undefined {
  if (!row) return undefined
  if (row.platform === 'grok') {
    const extra = (row.extra || {}) as Record<string, any>
    const billing = extra.grok_billing_snapshot as Record<string, any> | undefined
    const quota = extra.grok_quota_snapshot as Record<string, any> | undefined
    return (
      billing?.plan ||
      quota?.subscription_tier ||
      row.credentials?.subscription_tier ||
      extra.subscription_tier ||
      row.credentials?.plan_type ||
      row.parent_plan_type ||
      undefined
    )
  }
  return row.credentials?.plan_type || row.parent_plan_type || undefined
}

function getOpenAIAuthMode(row: any): string | undefined {
  if (!row || row.platform !== 'openai' || row.type !== 'oauth') return undefined
  const authMode = row.credentials?.auth_mode
  return typeof authMode === 'string' && authMode.trim() ? authMode : undefined
}

// Antigravity 订阅等级辅助函数
function getAntigravityTierFromRow(row: any): string | null {
  if (row.platform !== 'antigravity') return null
  const extra = row.extra as Record<string, unknown> | undefined
  if (!extra) return null
  const lca = extra.load_code_assist as Record<string, unknown> | undefined
  if (!lca) return null
  const paid = lca.paidTier as Record<string, unknown> | undefined
  if (paid && typeof paid.id === 'string') return paid.id
  const current = lca.currentTier as Record<string, unknown> | undefined
  if (current && typeof current.id === 'string') return current.id
  return null
}

function getAntigravityTierLabel(row: any): string | null {
  const tier = getAntigravityTierFromRow(row)
  switch (tier) {
    case 'free-tier': return t('admin.accounts.tier.free')
    case 'g1-pro-tier': return t('admin.accounts.tier.pro')
    case 'g1-ultra-tier': return t('admin.accounts.tier.ultra')
    default: return null
  }
}

// 账号显示邮箱:优先账号自身(extra/credentials),影子账号回退母账号 parent_email。
// 供名称单元格 v-if/标题/文本三处共用,避免同一回退链在模板里重复三次。
function accountDisplayEmail(row: any): string {
  return row.extra?.email_address || row.extra?.email || row.credentials?.email || row.parent_email || ''
}

function accountIdentitySubtitle(row: Account): string {
  const parts = [`#${row.id}`]
  const email = accountDisplayEmail(row)
  if (email) parts.push(email)
  if (row.parent_chatgpt_account_id) parts.push(String(row.parent_chatgpt_account_id))
  return parts.join(' · ')
}

function accountHomepageUrl(row: Account): string {
  if (row.type !== 'apikey' || typeof row.credentials?.base_url !== 'string') return ''
  const baseUrl = sanitizeUrl(row.credentials.base_url)
  return baseUrl ? new URL(baseUrl).origin : ''
}

type OpenAICompactBadgeState = 'active' | 'blocked' | 'auto'

function getOpenAICompactState(row: any): OpenAICompactBadgeState | null {
  if (row.platform !== 'openai' || (row.type !== 'oauth' && row.type !== 'apikey')) return null
  const extra = row.extra as Record<string, unknown> | undefined
  const mode = typeof extra?.openai_compact_mode === 'string' ? extra.openai_compact_mode : 'auto'
  if (mode === 'force_on') return 'active'
  if (mode === 'force_off') return 'blocked'
  if (typeof extra?.openai_compact_supported === 'boolean') {
    return extra.openai_compact_supported ? 'active' : 'blocked'
  }
  return 'auto'
}

function getOpenAICompactMeta(row: any): { label: string; className: string; dotClass: string } | null {
  const state = getOpenAICompactState(row)
  if (!state) return null
  switch (state) {
    case 'active':
      return {
        label: t('admin.accounts.openai.compactSupported'),
        className: 'text-emerald-600 dark:text-emerald-300',
        dotClass: 'bg-emerald-500 shadow-[0_0_0_2px_rgba(16,185,129,0.14)]'
      }
    case 'blocked':
      return {
        label: t('admin.accounts.openai.compactUnsupported'),
        className: 'text-rose-600 dark:text-rose-300',
        dotClass: 'bg-rose-500 shadow-[0_0_0_2px_rgba(244,63,94,0.14)]'
      }
    case 'auto':
      return {
        label: t('admin.accounts.openai.compactAuto'),
        className: 'text-slate-500 dark:text-slate-400',
        dotClass: 'bg-slate-300 dark:bg-slate-500'
      }
  }
}

function getOpenAICompactTitle(row: any): string {
  const extra = row.extra as Record<string, unknown> | undefined
  const checkedAt = typeof extra?.openai_compact_checked_at === 'string' ? extra.openai_compact_checked_at : ''
  const label = getOpenAICompactMeta(row)?.label || ''
  if (!checkedAt) return label
  return `${label} | ${t('admin.accounts.openai.compactLastChecked')}: ${formatDateTime(new Date(checkedAt))}`
}

function getAntigravityTierClass(row: any): string {
  const tier = getAntigravityTierFromRow(row)
  switch (tier) {
    case 'free-tier': return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
    case 'g1-pro-tier': return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    case 'g1-ultra-tier': return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    default: return ''
  }
}

// All available columns
const allColumns = computed(() => {
  const c = [
    { key: 'select', label: '', sortable: false },
    { key: 'name', label: t('admin.accounts.columns.name'), sortable: true, class: 'min-w-[180px] max-w-[220px]' },
    { key: 'id', label: t('admin.accounts.columns.id'), sortable: false },
    { key: 'status', label: t('admin.accounts.columns.status'), sortable: true, class: 'min-w-[128px]' },
    { key: 'usage', label: t('admin.accounts.columns.usageWindows'), sortable: false, class: 'min-w-[260px]' },
    { key: 'capacity', label: t('admin.accounts.columns.capacity'), sortable: false, class: 'min-w-[100px]' },
    { key: 'schedulable', label: t('admin.accounts.columns.schedulable'), sortable: true, class: 'min-w-[92px]' },
    { key: 'platform_type', label: t('admin.accounts.columns.platformType'), sortable: false }
  ]
  if (!authStore.isSimpleMode) {
    c.push({ key: 'groups', label: t('admin.accounts.columns.groups'), sortable: false })
  }
  c.push(
    { key: 'uploader', label: t('admin.sharedPool.columns.uploader'), sortable: false },
    { key: 'today_stats', label: t('admin.accounts.columns.todayStats'), sortable: false },
    { key: 'pool_record', label: t('admin.sharedPool.actions.poolRecord'), sortable: false },
    { key: 'proxy', label: t('admin.accounts.columns.proxy'), sortable: false },
    { key: 'priority', label: t('admin.accounts.columns.priority'), sortable: false },
    { key: 'scheduler_score', label: t('admin.accounts.columns.schedulerScore'), sortable: false },
    { key: 'rate_multiplier', label: t('admin.accounts.columns.billingRateMultiplier'), sortable: false },
    { key: 'upstream_billing_rate', label: t('admin.accounts.columns.upstreamBillingRate'), sortable: false },
    { key: 'last_used_at', label: t('admin.accounts.columns.lastUsed'), sortable: false },
    { key: 'created_at', label: t('admin.accounts.columns.createdAt'), sortable: true },
    { key: 'expires_at', label: t('admin.accounts.columns.expiresAt'), sortable: false },
    { key: 'notes', label: t('admin.accounts.columns.notes'), sortable: false },
    { key: 'actions', label: t('admin.accounts.columns.actions'), sortable: false }
  )
  return c
})

const toggleableColumns = computed(() =>
  allColumns.value.filter(col => !['select', 'name', 'actions'].includes(col.key))
)

// Filtered columns based on visibility
const cols = computed(() => {
  if (embedded) {
    return ['select', 'name', 'usage', 'status', 'actions'].flatMap(key => {
      const column = allColumns.value.find(col => col.key === key)
      return column ? [column] : []
    })
  }
  return allColumns.value.filter(col =>
    ['select', 'name', 'actions'].includes(col.key) || !hiddenColumns.has(col.key)
  )
})

const handleEdit = (a: Account) => { edAcc.value = a; showEdit.value = true }
const poolRecordFor = (accountID: number) => props.poolRecords[accountID]
const hasPoolCost = (account: Account) => Number(account.pool_purchase_cost_minor || account.pool_net_cost_minor || 0) > 0 || Boolean(poolRecordFor(account.id))
const formatPoolCost = (minor?: number | null) => `¥${(Number(minor || 0) / 100).toFixed(2)}`
type PoolWorkbenchTraceQuality = 'ready' | 'no_cost' | 'missing_expected_tokens' | 'partial_expected_tokens' | 'future_purchase_time' | 'derived' | 'unavailable'
type PoolWorkbenchAccount = Account & {
  pool_latest_purchase_source?: string | null
  pool_purchase_source_count?: number
  pool_purchase_cost_minor?: number
  pool_recognized_cost_minor?: number
  pool_latest_purchased_at?: string | null
  pool_recovery_data_quality?: PoolWorkbenchTraceQuality
}
const poolWorkbenchRecordFor = (account: Account) => {
  const page = account as PoolWorkbenchAccount
  const fallback = poolRecordFor(account.id)
  const fallbackMinor = (amount?: number | null) => Math.round(Number(amount || 0) * 100)
  const sourceName = page.pool_latest_purchase_source?.trim() || fallback?.purchase_source_name?.trim() || ''
  const grossCostMinor = Number(page.pool_purchase_cost_minor ?? fallbackMinor(fallback?.purchase_cost))
  const netCostMinor = Number(page.pool_net_cost_minor ?? fallbackMinor(fallback?.purchase_cost))
  const hasPageMetrics = Boolean(page.pool_recovery_data_quality)
  const legacyRemainingCostMinor = Math.max(0, Number(page.pool_remaining_cost_minor ?? fallbackMinor(fallback?.remaining_cost)))
  const recognizedCostMinor = Math.max(0, Number(
    hasPageMetrics
      ? (page.pool_recognized_cost_minor ?? 0)
      : (fallback ? fallbackMinor(fallback.usage_value) : Math.max(0, netCostMinor - legacyRemainingCostMinor))
  ))
  const remainingCostMinor = hasPageMetrics
    ? Math.max(0, grossCostMinor - recognizedCostMinor)
    : legacyRemainingCostMinor
  const progress = Math.max(0, grossCostMinor > 0
    ? recognizedCostMinor / grossCostMinor
    : Number(page.pool_cost_progress ?? ((fallback?.roi_rate || 0) / 100)))
  return {
    sourceName,
    sourceCount: Math.max(0, Number(page.pool_purchase_source_count ?? (sourceName ? 1 : 0))),
    grossCostMinor,
    netCostMinor,
    recognizedCostMinor,
    latestPurchaseAt: page.pool_latest_purchased_at || fallback?.paid_at || null,
    remainingCostMinor,
    netGainMinor: recognizedCostMinor - grossCostMinor,
    progress,
    traceQuality: page.pool_recovery_data_quality || (fallback ? 'derived' : 'unavailable') as PoolWorkbenchTraceQuality
  }
}
const poolTraceBadgeStatus = (quality: PoolWorkbenchTraceQuality) =>
  quality === 'ready'
    ? 'success'
    : quality === 'future_purchase_time'
      ? 'error'
      : ['derived', 'missing_expected_tokens', 'partial_expected_tokens'].includes(quality) ? 'warning' : 'inactive'
const poolRecoveryLabel = (account: Account) => {
  const progress = Number(account.pool_cost_progress || 0)
  if (progress >= 1) return t('admin.sharedPool.status.recovered')
  return `${t('admin.sharedPool.status.recovering')} ${(Math.max(0, progress) * 100).toFixed(1)}%`
}

const normalizeApprovalStatus = (status: string) => status.toLowerCase()

const approvalActionLabel = (action: PoolApproval['action_type']) => t(
  action === 'VIEW_CREDENTIAL'
    ? 'admin.sharedPool.approval.viewCredential'
		: action === 'DELETE_ACCOUNT'
		  ? 'admin.sharedPool.approval.deleteAccount'
		  : 'admin.sharedPool.approval.updateAccount'
)

const approvalTriggerReason = (action: PoolApproval['action_type']) => t(
  action === 'VIEW_CREDENTIAL'
    ? 'admin.sharedPool.approval.triggerCredential'
    : action === 'DELETE_ACCOUNT'
      ? 'admin.sharedPool.approval.triggerDelete'
      : 'admin.sharedPool.approval.triggerUpdate'
)

const approvalRequestReason = (reason: string) => {
  const defaults: Record<string, string> = {
    'delete account': 'deleteAccountReason',
    'update account information': 'updateAccountReason',
    'reauthorize account credentials': 'reauthorizeReason',
    'update shared pool account metadata': 'updatePoolReason',
    'update shared pool account cost record': 'updateCostReason',
    'primary administrator direct credential access': 'credentialAccessReason'
  }
  const key = defaults[reason.trim().toLowerCase()]
  return key ? t(`admin.sharedPool.approval.reasons.${key}`) : reason || '-'
}

const approvalBusinessSummary = computed(() => {
  const approval = selectedApproval.value
  if (!approval) return ''
  const business = approval.changes?.business
  if (business?.groups?.length) {
    const scope = business.scope?.length ? business.scope : business.groups.map(group => group.key)
    return t('admin.sharedPool.approval.businessGrouped', {
      groups: scope.map(approvalGroupLabel).join('、'),
      count: business.groups.reduce((total, group) => total + group.items.length, 0)
    })
  }
  if (approval.action_type === 'DELETE_ACCOUNT') return t('admin.sharedPool.approval.businessDelete')
  if (approval.action_type === 'VIEW_CREDENTIAL') return t('admin.sharedPool.approval.businessCredential')
  const fields = approvalDiffRows.value.map(row => row.field).join('、')
  return fields
    ? t('admin.sharedPool.approval.businessUpdateFields', { fields })
    : t('admin.sharedPool.approval.businessUpdate')
})

const approvalStatusLabel = (status: string) => {
  const normalized = normalizeApprovalStatus(status)
  const key = ['pending', 'approved', 'rejected', 'expired', 'consumed'].includes(normalized)
    ? normalized
    : 'pending'
  return t(`admin.sharedPool.approval.${key}`)
}

const approvalStatusBadge = (status: string) => {
  switch (normalizeApprovalStatus(status)) {
    case 'approved':
    case 'consumed': return 'success'
    case 'rejected':
    case 'expired': return 'danger'
    default: return 'warning'
  }
}

const canReviewApproval = (approval: PoolApproval) => (
  normalizeApprovalStatus(approval.status) === 'pending' &&
  (authStore.user?.is_primary_admin || approval.requested_by_user_id !== authStore.user?.id)
)

const canRevealApproval = (approval: PoolApproval) => (
  approval.action_type === 'VIEW_CREDENTIAL' &&
  normalizeApprovalStatus(approval.status) === 'approved' &&
  approval.requested_by_user_id === authStore.user?.id &&
  !approval.revealed_at
)

const loadPendingApprovalCount = async () => {
  try {
    const [all, highRisk] = await Promise.all([
      adminAPI.sharedPool.listApprovals({ scope: 'reviewable', status: 'pending', page: 1, page_size: 1 }),
      adminAPI.sharedPool.listApprovals({ scope: 'reviewable', status: 'pending', high_risk: true, page: 1, page_size: 1 })
    ])
    pendingApprovalCount.value = all.total
    highRiskApprovalCount.value = highRisk.total
  } catch {
    pendingApprovalCount.value = 0
    highRiskApprovalCount.value = 0
  }
}

const loadApprovals = async () => {
  approvalsLoading.value = true
  try {
    const result = await adminAPI.sharedPool.listApprovals({
      scope: approvalScope.value,
      status: approvalScope.value === 'reviewable' ? 'pending' : approvalStatusFilter.value || undefined,
      page: approvalPagination.page,
      page_size: approvalPagination.page_size
    })
    approvals.value = result.items
    approvalPagination.total = result.total
    if (selectedApproval.value) {
      selectedApproval.value = result.items.find(item => item.id === selectedApproval.value?.id) ?? null
    }
    await loadPendingApprovalCount()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.sharedPool.approval.loadFailed')))
  } finally {
    approvalsLoading.value = false
  }
}

const openApprovalCenter = () => {
  showApprovalCenter.value = true
  selectedApproval.value = null
  approvalDecisionReason.value = ''
  void loadApprovals()
}

const selectApproval = (approval: PoolApproval) => {
  selectedApproval.value = approval
  approvalDecisionReason.value = ''
}

const changeApprovalFilter = () => {
  approvalPagination.page = 1
  selectedApproval.value = null
  approvalDecisionReason.value = ''
  void loadApprovals()
}

const changeApprovalScope = (scope: PoolApprovalScope) => {
  approvalScope.value = scope
  approvalStatusFilter.value = scope === 'reviewable' ? 'pending' : ''
  changeApprovalFilter()
}

const handleApprovalPageChange = (page: number) => {
  approvalPagination.page = page
  selectedApproval.value = null
  approvalDecisionReason.value = ''
  void loadApprovals()
}

const closeApprovalCenter = () => {
  if (approvalActionID.value !== null) return
  showApprovalCenter.value = false
  selectedApproval.value = null
  approvalDecisionReason.value = ''
}

const decideApproval = async (decision: 'approve' | 'reject') => {
  const approval = selectedApproval.value
  if (!approval || !canReviewApproval(approval)) return
  if (decision === 'reject' && !approvalDecisionReason.value.trim()) {
    appStore.showError(t('admin.sharedPool.approval.rejectReasonRequired'))
    return
  }

  approvalActionID.value = approval.id
  approvalDecisionInProgress.value = decision
  try {
    const updated = decision === 'approve'
      ? await adminAPI.sharedPool.approveApproval(approval.id, approvalDecisionReason.value)
      : await adminAPI.sharedPool.rejectApproval(approval.id, approvalDecisionReason.value)
    appStore.showSuccess(t(`admin.sharedPool.approval.${decision}Success`))
    approvalDecisionReason.value = ''
    selectedApproval.value = updated
    await loadApprovals()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.sharedPool.approval.decisionFailed'), {
      POOL_APPROVAL_STALE: t('admin.sharedPool.approval.errors.stale'),
      APPROVAL_ALREADY_DECIDED: t('admin.sharedPool.approval.errors.decided'),
      POOL_APPROVAL_CONFLICT: t('admin.sharedPool.approval.errors.conflict')
    }))
  } finally {
    approvalActionID.value = null
    approvalDecisionInProgress.value = null
  }
}

const openCredentialRequest = (account: Account) => {
  credentialAccount.value = { id: account.id, name: account.name }
  credentialPurpose.value = ''
  credentialReveal.value = null
  showCredentialDialog.value = true
}

const closeCredentialDialog = () => {
  if (credentialSubmitting.value) return
  if (credentialClearTimer) {
    clearTimeout(credentialClearTimer)
    credentialClearTimer = null
  }
  showCredentialDialog.value = false
  credentialAccount.value = null
  credentialPurpose.value = ''
  credentialReveal.value = null
}

const revealCredential = async (approval: PoolApproval) => {
  if (!approval.account_id) return
  approvalActionID.value = approval.id
  credentialSubmitting.value = true
  try {
    const revealed = await credentialStepUp.run(() => adminAPI.sharedPool.revealApproval(approval.id))
    credentialAccount.value = { id: approval.account_id, name: approval.account_name || `#${approval.account_id}` }
    credentialPurpose.value = approval.reason
    credentialReveal.value = revealed
    showApprovalCenter.value = false
    selectedApproval.value = null
    showCredentialDialog.value = true
    if (credentialClearTimer) clearTimeout(credentialClearTimer)
    credentialClearTimer = setTimeout(() => {
      credentialClearTimer = null
      closeCredentialDialog()
    }, 60_000)
    await loadApprovals()
  } catch (error) {
    if (!isStepUpCancelled(error)) {
      appStore.showError(extractApiErrorMessage(error, t('admin.sharedPool.approval.revealFailed')))
    }
  } finally {
    credentialSubmitting.value = false
    approvalActionID.value = null
  }
}

const submitCredentialRequest = async () => {
  const account = credentialAccount.value
  const purpose = credentialPurpose.value.trim()
  if (!account || (!authStore.user?.is_primary_admin && !purpose)) return

  credentialSubmitting.value = true
  try {
    const approval = await adminAPI.sharedPool.createApproval({
      action_type: 'VIEW_CREDENTIAL',
      account_id: account.id,
      reason: purpose
    })
    await loadPendingApprovalCount()
    if (normalizeApprovalStatus(approval.status) === 'approved') {
      credentialSubmitting.value = false
      await revealCredential(approval)
      return
    }
    appStore.showSuccess(t('admin.sharedPool.approval.credentialSubmitted'))
    credentialSubmitting.value = false
    closeCredentialDialog()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.sharedPool.approval.submitFailed')))
  } finally {
    credentialSubmitting.value = false
  }
}

const handleApprovalSubmitted = () => {
  void loadPendingApprovalCount()
}

const openMenu = (account: Account) => {
  menu.acc = account
  menu.show = true
}
const toggleSelectAllVisible = async (event: Event) => {
  if (pageBatchLoading.value) return
  const target = event.target as HTMLInputElement
  try {
    await setCurrentPageSelected(target.checked)
  } catch (error) {
    if (error !== staleImportBatchRequest) appStore.showError(extractApiErrorMessage(error, t('common.error')))
  }
}
const bulkActionInProgress = ref(false)
const handleBulkDelete = async () => {
  if (bulkActionInProgress.value) return
  const accountIDs = [...selIds.value]
  if (!accountIDs.length || !confirm(t('admin.sharedPool.delete.bulkConfirm', { count: accountIDs.length }))) return
  bulkActionInProgress.value = true

	let remainingIDs = [...accountIDs]
	let activeChunk: number[] = []
	const failedIDs = new Set<number>()
	let deleted = 0
	let approvalRequired = 0
	let failed = 0
  try {
		while (remainingIDs.length) {
			activeChunk = remainingIDs.splice(0, 100)
			const result = await adminAPI.accounts.bulkDelete(activeChunk)
			deleted += result.deleted
			approvalRequired += result.approval_required
			failed += result.failed
			const affectedIDs = new Set<number>()
			for (const item of result.results) {
				if (item.status === 'failed') failedIDs.add(item.account_id)
				for (const id of item.result?.affected_account_ids || []) affectedIDs.add(id)
			}
			if (affectedIDs.size) remainingIDs = remainingIDs.filter(id => !affectedIDs.has(id))
			activeChunk = []
		}
		setSelectedIds([...failedIDs])
    const message = t('admin.sharedPool.delete.bulkSummary', {
			deleted,
			approval: approvalRequired,
			failed
    })
		failed ? appStore.showError(message) : appStore.showSuccess(message)
    await reload()
    void loadPendingApprovalCount()
  } catch (error) {
		setSelectedIds([...new Set([...failedIDs, ...activeChunk, ...remainingIDs])])
    appStore.showError(extractApiErrorMessage(error, t('admin.sharedPool.delete.failed')))
		await reload()
		void loadPendingApprovalCount()
  } finally {
    bulkActionInProgress.value = false
  }
}
const handleBulkResetStatus = async () => {
  if (bulkActionInProgress.value) return
  if (!confirm(t('common.confirm'))) return
  bulkActionInProgress.value = true
  const accountIDs = [...selIds.value]
  try {
    const result = await adminAPI.accounts.batchClearError(accountIDs)
    if (result.failed > 0) {
      const failedIDs = (result.errors || []).map(item => item.account_id)
      setSelectedIds(failedIDs.length ? failedIDs : accountIDs)
      appStore.showError(t('admin.accounts.bulkActions.partialSuccess', { success: result.success, failed: result.failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.bulkActions.resetStatusSuccess', { count: result.success }))
      clearSelection()
    }
    await refreshCurrentPage()
  } catch (error) {
    console.error('Failed to bulk reset status:', error)
    appStore.showError(String(error))
  } finally {
    bulkActionInProgress.value = false
  }
}
const handleBulkRefreshToken = async () => {
  if (bulkActionInProgress.value) return
  if (!confirm(t('common.confirm'))) return
  bulkActionInProgress.value = true
  const accountIDs = [...selIds.value]
  try {
    const result = await adminAPI.accounts.batchRefresh(accountIDs)
    if (result.failed > 0) {
      const failedIDs = (result.errors || []).map(item => item.account_id)
      setSelectedIds(failedIDs.length ? failedIDs : accountIDs)
      appStore.showError(t('admin.accounts.bulkActions.partialSuccess', { success: result.success, failed: result.failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.bulkActions.refreshTokenSuccess', { count: result.success }))
      clearSelection()
    }
    await refreshCurrentPage()
  } catch (error) {
    console.error('Failed to bulk refresh token:', error)
    appStore.showError(String(error))
  } finally {
    bulkActionInProgress.value = false
  }
}
const handleBulkProbeUpstreamBilling = async () => {
  if (bulkActionInProgress.value) return
  const accountIDs = [...selIds.value]
  if (accountIDs.length === 0) {
    appStore.showError(t('admin.accounts.upstreamBilling.noEligibleAccounts'))
    return
  }
  if (accountIDs.length > 20) {
    appStore.showError(t('admin.accounts.upstreamBilling.batchLimit'))
    return
  }
  bulkActionInProgress.value = true
  accountIDs.forEach(id => probingUpstreamBilling.add(id))
  try {
    const results = await adminAPI.accounts.probeUpstreamBillingBatch(accountIDs)
    if (results.some(result => result.snapshot)) await refreshCurrentPage()
    const failed = results.filter(result => result.error).length
    if (failed > 0) {
      appStore.showError(t('admin.accounts.upstreamBilling.batchPartial', { success: results.length - failed, failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.upstreamBilling.batchCompleted', { count: results.length }))
    }
  } catch (error) {
    console.error('Failed to probe upstream billing in batch:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.upstreamBilling.probeFailed')))
  } finally {
    accountIDs.forEach(id => probingUpstreamBilling.delete(id))
    bulkActionInProgress.value = false
  }
}
const normalizeBulkSchedulableResult = (
  result: {
    success?: number
    failed?: number
    success_ids?: number[]
    failed_ids?: number[]
    results?: Array<{ account_id: number; success: boolean }>
  },
  accountIds: number[]
) => {
  const responseSuccessIds = Array.isArray(result.success_ids) ? result.success_ids : []
  const responseFailedIds = Array.isArray(result.failed_ids) ? result.failed_ids : []
  if (responseSuccessIds.length > 0 || responseFailedIds.length > 0) {
    return {
      successIds: responseSuccessIds,
      failedIds: responseFailedIds,
      successCount: typeof result.success === 'number' ? result.success : responseSuccessIds.length,
      failedCount: typeof result.failed === 'number' ? result.failed : responseFailedIds.length,
      hasIds: true,
      hasCounts: true
    }
  }

  const results = Array.isArray(result.results) ? result.results : []
  if (results.length > 0) {
    const successIds = results.filter(item => item.success).map(item => item.account_id)
    const failedIds = results.filter(item => !item.success).map(item => item.account_id)
    return {
      successIds,
      failedIds,
      successCount: typeof result.success === 'number' ? result.success : successIds.length,
      failedCount: typeof result.failed === 'number' ? result.failed : failedIds.length,
      hasIds: true,
      hasCounts: true
    }
  }

  const hasExplicitCounts = typeof result.success === 'number' || typeof result.failed === 'number'
  const successCount = typeof result.success === 'number' ? result.success : 0
  const failedCount = typeof result.failed === 'number' ? result.failed : 0
  if (hasExplicitCounts && failedCount === 0 && successCount === accountIds.length && accountIds.length > 0) {
    return {
      successIds: accountIds,
      failedIds: [],
      successCount,
      failedCount,
      hasIds: true,
      hasCounts: true
    }
  }

  return {
    successIds: [],
    failedIds: [],
    successCount,
    failedCount,
    hasIds: false,
    hasCounts: hasExplicitCounts
  }
}
const handleBulkToggleSchedulable = async (schedulable: boolean) => {
  if (bulkActionInProgress.value) return
  const accountIds = [...selIds.value]
  if (!accountIds.length) return
  bulkActionInProgress.value = true
  try {
    const result = await adminAPI.accounts.bulkUpdate(accountIds, { schedulable })
    const { failedIds, successCount, failedCount, hasIds, hasCounts } = normalizeBulkSchedulableResult(result, accountIds)
    if (!hasIds && !hasCounts) {
      appStore.showError(t('admin.accounts.bulkSchedulableResultUnknown'))
      setSelectedIds(accountIds)
      await refreshCurrentPage()
      return
    }
    if (successCount > 0 && failedCount === 0) {
      const message = schedulable
        ? t('admin.accounts.bulkSchedulableEnabled', { count: successCount })
        : t('admin.accounts.bulkSchedulableDisabled', { count: successCount })
      appStore.showSuccess(message)
    }
    if (failedCount > 0) {
      const message = hasCounts || hasIds
        ? t('admin.accounts.bulkSchedulablePartial', { success: successCount, failed: failedCount })
        : t('admin.accounts.bulkSchedulableResultUnknown')
      appStore.showError(message)
      setSelectedIds(failedIds.length > 0 ? failedIds : accountIds)
    } else {
      if (hasIds) clearSelection()
      else setSelectedIds(accountIds)
    }
    await refreshCurrentPage()
  } catch (error) {
    console.error('Failed to bulk toggle schedulable:', error)
    appStore.showError(t('common.error'))
  } finally {
    bulkActionInProgress.value = false
  }
}
const buildBulkEditFilterSnapshot = () => {
  const rawParams = toRaw(params) as Record<string, unknown>
  const sortOrder: AccountSortOrder = rawParams.sort_order === 'desc' ? 'desc' : 'asc'
  const rawUploaderUserID = rawParams.uploader_user_id
  const uploaderUserID = typeof rawUploaderUserID === 'number'
    ? (rawUploaderUserID > 0 ? rawUploaderUserID : undefined)
    : (/^\d+$/.test(String(rawUploaderUserID || '')) ? Number(rawUploaderUserID) : undefined)
  const filters = {
    platform: typeof rawParams.platform === 'string' ? rawParams.platform : '',
    type: typeof rawParams.type === 'string' ? rawParams.type : '',
    status: typeof rawParams.status === 'string' ? rawParams.status : '',
    group: typeof rawParams.group === 'string' ? rawParams.group : '',
    search: typeof rawParams.search === 'string' ? rawParams.search : '',
    privacy_mode: typeof rawParams.privacy_mode === 'string' ? rawParams.privacy_mode : '',
    uploader_user_id: uploaderUserID,
    uploader_unassigned: rawUploaderUserID === 'unassigned' || undefined,
    sort_by: typeof rawParams.sort_by === 'string' ? rawParams.sort_by : '',
    sort_order: sortOrder
  }
  if (embedded && workbenchScope.value === 'batch' && selectedWorkbenchBatchID.value) {
    return { ...filters, import_batch_id: selectedWorkbenchBatchID.value }
  }
  if (embedded && workbenchScope.value === 'standalone') {
    return { ...filters, import_batch_scope: 'standalone' as const }
  }
  if (embedded && workbenchScope.value === 'uploader') {
    return {
      ...filters,
      uploader_user_id: Number(selectedWorkbenchUploaderID.value) || undefined,
      uploader_unassigned: selectedWorkbenchUploaderID.value === 'unassigned' || undefined,
      import_batch_scope: 'batched' as const
    }
  }
  return filters
}

const openBulkEditSelected = () => {
  if (!selIds.value.length) return
  bulkEditTarget.value = {
    mode: 'selected',
    accountIds: [...selIds.value],
    selectedPlatforms: [...selPlatforms.value],
    selectedTypes: [...selTypes.value]
  }
  showBulkEdit.value = true
}

const hasEffectiveFilters = computed(() => {
  return Boolean(
    params.platform || params.type || params.status || params.group ||
    String(params.search || '').trim() || params.privacy_mode || params.uploader_user_id
  )
})

const openBulkEditFiltered = async () => {
  if (bulkActionInProgress.value || !hasEffectiveFilters.value || selIds.value.length > 0) return
  const filters = buildBulkEditFilterSnapshot()
  const filterKey = JSON.stringify(filters)
  const { uploader_unassigned: uploaderUnassigned, ...summaryFilters } = filters
  bulkActionInProgress.value = true
  try {
    const summary = await adminAPI.accounts.getSelectionSummary({
      ...summaryFilters,
      uploader_user_id: uploaderUnassigned ? 'unassigned' : filters.uploader_user_id
    })
    if (!hasEffectiveFilters.value || selIds.value.length > 0 || JSON.stringify(buildBulkEditFilterSnapshot()) !== filterKey) return
    bulkEditTarget.value = {
      mode: 'filtered',
      filters,
      previewCount: summary.total,
      selectedPlatforms: summary.platforms as AccountPlatform[],
      selectedTypes: summary.types as AccountType[]
    }
    showBulkEdit.value = true
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.bulkEdit.failed')))
  } finally {
    bulkActionInProgress.value = false
  }
}

const handleBulkUpdated = (failedIDs: number[] = []) => {
  showBulkEdit.value = false
  bulkEditTarget.value = null
  setSelectedIds(failedIDs)
  void refreshCurrentPage()
}
const handleDataImported = (importedAccounts: Array<{ id: number; name: string }>) => {
  showImportData.value = false
  void reload()
  if (embedded) emit('pool-imported', importedAccounts)
}
const buildAccountQueryFilters = () => {
  const filters = workbenchRequestFilters({
    platform: params.platform || '',
    type: params.type || '',
    status: params.status || '',
    group: params.group || '',
    privacy_mode: params.privacy_mode || '',
    uploader_user_id: params.uploader_user_id || '',
    search: params.search || '',
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order
  })
  if (embedded && workbenchScope.value === 'batch' && selectedWorkbenchBatchID.value) {
    filters.import_batch_id = selectedWorkbenchBatchID.value
  } else if (embedded && workbenchScope.value === 'standalone') {
    filters.import_batch_scope = 'standalone'
  }
  return filters
}
const handleProbeUpstreamBilling = async (account: Account) => {
  if (probingUpstreamBilling.has(account.id)) return
  probingUpstreamBilling.add(account.id)
  try {
    const result = await adminAPI.accounts.probeUpstreamBilling(account.id)
    if (result.snapshot) {
      await refreshCurrentPage()
    }
  } catch (error) {
    console.error('Failed to probe upstream billing:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.upstreamBilling.probeFailed')))
  } finally {
    probingUpstreamBilling.delete(account.id)
  }
}
const handleAccountUpdated = async () => {
  await refreshCurrentPage()
  enterAutoRefreshSilentWindow()
}
const formatExportTimestamp = () => {
  const now = new Date()
  const pad2 = (value: number) => String(value).padStart(2, '0')
  return `${now.getFullYear()}${pad2(now.getMonth() + 1)}${pad2(now.getDate())}${pad2(now.getHours())}${pad2(now.getMinutes())}${pad2(now.getSeconds())}`
}
const openExportDataDialog = () => {
  showExportDataDialog.value = true
}
const handleExportData = async () => {
  if (exportingData.value) return
  exportingData.value = true
  try {
    const dataPayload = await accountExportStepUp.run(() => adminAPI.accounts.exportData(
      selIds.value.length > 0
        ? { ids: selIds.value, includeProxies: false }
        : {
            includeProxies: false,
            filters: buildAccountQueryFilters()
          }
    ))
    const timestamp = formatExportTimestamp()
    const filename = `sub2api-account-${timestamp}.json`
    const blob = new Blob([JSON.stringify(dataPayload, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
    // spark 影子账号被后端排除出备份(其凭据透传母账号、调度配置不可经凭据型导入重建);
    // 跳过非零时明确提示用户,避免「下载成功但少了账号」的静默丢失。
    if (dataPayload.skipped_shadows && dataPayload.skipped_shadows > 0) {
      appStore.showWarning(t('admin.accounts.dataExportedSkippedShadows', { count: dataPayload.skipped_shadows }))
    } else {
      appStore.showSuccess(t('admin.accounts.dataExported'))
    }
  } catch (error: any) {
    if (isStepUpCancelled(error)) {
      // 用户主动取消 step-up 验证，静默返回，不弹错误提示。
    } else if (isStepUpBlocked(error)) {
      appStore.showError(
        stepUpBlockReason(error) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN'
          ? t('stepUp.adminApiKeyForbidden')
          : t('stepUp.notEnabled')
      )
    } else {
      appStore.showError(error?.message || t('admin.accounts.dataExportFailed'))
    }
  } finally {
    exportingData.value = false
    showExportDataDialog.value = false
  }
}
const accountExportStepUp = useStepUp()
const closeTestModal = () => { showTest.value = false; testingAcc.value = null }
const closeStatsModal = () => { showStats.value = false; statsAcc.value = null }
const closeReAuthModal = () => { showReAuth.value = false; reAuthAcc.value = null }
const handleTest = (a: Account) => { testingAcc.value = a; showTest.value = true }
const handleViewStats = (a: Account) => { statsAcc.value = a; showStats.value = true }
const handleSchedule = async (a: Account) => {
  scheduleAcc.value = a
  scheduleModelOptions.value = []
  showSchedulePanel.value = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(a.id)
    scheduleModelOptions.value = models.map((m: ClaudeModel) => ({ value: m.id, label: m.display_name || m.id }))
  } catch {
    scheduleModelOptions.value = []
  }
}
const closeSchedulePanel = () => { showSchedulePanel.value = false; scheduleAcc.value = null; scheduleModelOptions.value = [] }
const handleReAuth = (a: Account) => { reAuthAcc.value = a; showReAuth.value = true }
const duplicatingAccountIDs = new Set<number>()
const handleDuplicateAccount = async (a: Account) => {
  if (duplicatingAccountIDs.has(a.id)) return
  duplicatingAccountIDs.add(a.id)
  try {
    const duplicate = await adminAPI.accounts.duplicate(a.id)
    appStore.showSuccess(t('admin.accounts.duplicateSuccess', { name: duplicate.name }))
    await refreshCurrentPage()
    emit('pool-record', duplicate)
  } catch (error: any) {
    console.error('Failed to duplicate account:', error)
    appStore.showError(error?.message || t('admin.accounts.duplicateFailed'))
  } finally {
    duplicatingAccountIDs.delete(a.id)
  }
}
const handleRefresh = async (a: Account) => {
  try {
    await adminAPI.accounts.refreshCredentials(a.id)
    await refreshCurrentPage()
    enterAutoRefreshSilentWindow()
  } catch (error) {
    console.error('Failed to refresh credentials:', error)
  }
}
const handleRecoverState = async (a: Account) => {
  try {
    await adminAPI.accounts.recoverState(a.id)
    await refreshCurrentPage()
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('admin.accounts.recoverStateSuccess'))
  } catch (error: any) {
    console.error('Failed to recover account state:', error)
    appStore.showError(error?.message || t('admin.accounts.recoverStateFailed'))
  }
}
const handleResetQuota = async (a: Account) => {
  try {
    await adminAPI.accounts.resetAccountQuota(a.id)
    await refreshCurrentPage()
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('common.success'))
  } catch (error) {
    console.error('Failed to reset quota:', error)
  }
}

const privacyResultMessageKey = (account: Account): { type: 'success' | 'error'; key: string } => {
  const mode = typeof account.extra?.privacy_mode === 'string' ? account.extra.privacy_mode : ''
  if (account.platform === 'openai') {
    switch (mode) {
      case 'training_off':
        return { type: 'success', key: 'admin.accounts.privacyTrainingOff' }
      case 'training_set_cf_blocked':
        return { type: 'error', key: 'admin.accounts.privacyCfBlocked' }
      default:
        return { type: 'error', key: 'admin.accounts.privacyFailed' }
    }
  }
  if (account.platform === 'antigravity') {
    if (mode === 'privacy_set') {
      return { type: 'success', key: 'admin.accounts.privacyAntigravitySet' }
    }
    return { type: 'error', key: 'admin.accounts.privacyAntigravityFailed' }
  }
  return { type: 'error', key: 'admin.accounts.privacyFailed' }
}

const handleSetPrivacy = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.setPrivacy(a.id)
    await refreshCurrentPage()
    enterAutoRefreshSilentWindow()
    const result = privacyResultMessageKey(updated)
    if (result.type === 'success') {
      appStore.showSuccess(t(result.key))
    } else {
      appStore.showError(t(result.key))
    }
  } catch (error: any) {
    console.error('Failed to set privacy:', error)
    appStore.showError(error?.response?.data?.message || t('admin.accounts.privacyFailed'))
  }
}
const onRevertFallback = async (a: Account) => {
  try {
    await adminAPI.accounts.revertProxyFallback(a.id)
    appStore.showSuccess(t('admin.accounts.revertProxySuccess'))
    void refreshCurrentPage()
  } catch (error: any) {
    console.error('Failed to revert proxy fallback:', error)
    appStore.showError(error?.response?.data?.message || t('admin.accounts.revertProxyFailed'))
  }
}
const handleCreateSparkShadow = (a: Account) => {
  creatingShadowAcc.value = a
  showCreateShadowDialog.value = true
}
const confirmCreateSparkShadow = async () => {
  const a = creatingShadowAcc.value
  if (!a) return
  try {
    await adminAPI.accounts.createSparkShadow(a.id, { name: `${a.name} (Spark)` })
    showCreateShadowDialog.value = false
    creatingShadowAcc.value = null
    appStore.showSuccess(t('admin.accounts.createSparkShadowSuccess'))
    void refreshCurrentPage()
  } catch (error: any) {
    console.error('Failed to create spark shadow:', error)
    appStore.showError(error?.response?.data?.message || t('admin.accounts.createSparkShadowFailed'))
  }
}
const closeDeleteDialog = () => {
  if (deleteSubmitting.value) return
  showDeleteDialog.value = false
  deletingAcc.value = null
}
const handleDelete = async (account: Account) => {
  deletingAcc.value = account
  showDeleteDialog.value = true
}
const confirmDelete = async () => {
  const account = deletingAcc.value
	if (!account) return
  deleteSubmitting.value = true
  try {
		const result = await adminAPI.accounts.delete(account.id)
    showDeleteDialog.value = false
    deletingAcc.value = null
		appStore.showSuccess(t(result.approval ? 'admin.sharedPool.delete.approvalSubmitted' : 'admin.sharedPool.delete.success'))
		if (result.approval) void loadPendingApprovalCount()
		else reload()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.sharedPool.delete.failed'), {
      POOL_APPROVAL_CONFLICT: t('admin.sharedPool.approval.errors.conflict')
    }))
  } finally {
    deleteSubmitting.value = false
  }
}
const handleToggleSchedulable = async (a: Account) => {
  const nextSchedulable = !a.schedulable
  togglingSchedulable.value = a.id
  try {
    await adminAPI.accounts.setSchedulable(a.id, nextSchedulable)
    await refreshCurrentPage()
    enterAutoRefreshSilentWindow()
  } catch (error) {
    console.error('Failed to toggle schedulable:', error)
    appStore.showError(t('admin.accounts.failedToToggleSchedulable'))
  } finally {
    togglingSchedulable.value = null
  }
}
const handleShowTempUnsched = (a: Account) => { tempUnschedAcc.value = a; showTempUnsched.value = true }
const handleTempUnschedReset = async () => {
  showTempUnsched.value = false
  tempUnschedAcc.value = null
  await refreshCurrentPage()
  enterAutoRefreshSilentWindow()
}
const formatExpiresAt = (value: number | null) => {
  if (!value) return '-'
  return formatDateTime(
    new Date(value * 1000),
    {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    },
    'sv-SE'
  )
}
const isExpired = (value: number | null) => {
  if (!value) return false
  return value * 1000 <= Date.now()
}
// 所绑定代理的有效期(逻辑同 /admin/proxies,见 utils/proxyExpiry)
const proxyExpiryBadge = (p: AccountProxy): string => proxyExpiryBadgeClass(p.expires_at, p.status)
const proxyExpiryText = (p: AccountProxy): string => {
  const { key, params } = proxyExpiryLabelKey(p.expires_at, p.status)
  return params ? t(key, params) : t(key)
}

// 点击外部关闭自动刷新选项。
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (autoRefreshDropdownRef.value && !autoRefreshDropdownRef.value.contains(target)) {
    showAutoRefreshDropdown.value = false
  }
}

onMounted(async () => {
  workbenchNavigatorMobileMedia = window.matchMedia('(max-width: 1179px)')
  workbenchNavigatorMobileMedia.addEventListener('change', handleWorkbenchNavigatorMediaChange)
  load()
  loadUpstreamBillingProbeGlobalState()
  void loadPendingApprovalCount()
  try {
    const [p, g] = await Promise.all([
      adminAPI.proxies.getAll(),
      adminAPI.groups.getAll()
    ])
    proxies.value = p
    groups.value = g
  } catch (error) {
    console.error('Failed to load account reference options:', error)
  }
  try {
    const users = await adminAPI.users.list(1, 200, { sort_by: 'email', sort_order: 'asc' })
    uploaderOptions.value = users.items.map(user => ({ value: user.id, label: user.username || user.email }))
  } catch (error) {
    console.error('Failed to load account uploader options:', error)
  }
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleWorkbenchNavigatorKeydown)

  if (autoRefreshEnabled.value) {
    autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
    resumeAutoRefresh()
  } else {
    pauseAutoRefresh()
  }
})

onUnmounted(() => {
  if (credentialClearTimer) clearTimeout(credentialClearTimer)
  credentialReveal.value = null
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleWorkbenchNavigatorKeydown)
  workbenchNavigatorMobileMedia?.removeEventListener('change', handleWorkbenchNavigatorMediaChange)
  workbenchNavigatorMobileMedia = null
  releaseWorkbenchNavigatorBodyLock()
})
</script>

<style scoped>
.dialog-section-title {
  @apply mb-2 text-xs font-semibold uppercase text-gray-500 dark:text-gray-400;
  letter-spacing: 0;
}

.dialog-action-grid {
  @apply grid grid-cols-1 gap-2 sm:grid-cols-2;
}

.dialog-action-button {
  @apply flex min-h-11 min-w-0 items-center gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-left text-sm font-medium text-gray-800 hover:border-primary-300 hover:bg-primary-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-700 dark:hover:bg-primary-900/20;
}

.display-choice {
  @apply min-h-11 rounded-lg border border-gray-200 px-3 py-2 text-left text-sm font-medium text-gray-700 hover:border-primary-300 hover:bg-primary-50 dark:border-dark-700 dark:text-gray-200 dark:hover:border-primary-700 dark:hover:bg-primary-900/20;
}

.display-choice-active {
  @apply border-primary-400 bg-primary-50 text-primary-700 ring-1 ring-primary-200 dark:border-primary-700 dark:bg-primary-900/30 dark:text-primary-300 dark:ring-primary-800;
}

.display-setting-row {
  @apply flex min-h-11 items-center justify-between gap-3 rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 dark:border-dark-700 dark:text-gray-200;
}

.workbench-view-switch {
  @apply inline-flex min-h-11 items-center rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-800;
}

.workbench-view-option {
  @apply min-h-9 rounded-md px-3 text-xs font-medium text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white;
}

.workbench-view-option-active {
  @apply bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300;
}

.workbench-navigator-open,
.workbench-navigator-backdrop {
  display: none;
}

.account-operation-band {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  overflow: hidden;
  border: 1px solid rgb(226 232 240);
  border-radius: 8px;
  background: rgb(255 255 255);
}

.dark .account-operation-band {
  border-color: rgb(51 65 85);
  background: rgb(30 41 59 / 0.7);
}

.account-operation-filters {
  flex: 1 1 auto;
  min-width: 0;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.account-operation-filters :deep(p) {
  display: none;
}

.account-operation-actions {
  flex: 0 0 auto;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.workbench-sidebar-header {
  @apply shrink-0 border-b border-gray-200 bg-white/90 p-3 dark:border-dark-700 dark:bg-dark-900/80;
}

.workbench-scope-list {
  @apply shrink-0 space-y-0.5 px-2 pb-1 pt-2;
}

.workbench-nav-item {
  @apply relative flex min-h-11 w-full min-w-0 items-center gap-2 rounded-md border-l-[3px] border-transparent px-2.5 py-2 text-left text-sm text-gray-700 transition-colors hover:bg-white focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500 dark:text-gray-200 dark:hover:bg-dark-800;
}

.workbench-nav-item-active {
  @apply border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-300;
}

.workbench-nav-count {
  @apply ml-auto min-w-5 shrink-0 text-right text-xs tabular-nums text-gray-500 dark:text-gray-400;
}

.workbench-navigator-modes {
  @apply mx-2 my-2 grid shrink-0 grid-cols-2 rounded-md bg-gray-100 p-1 dark:bg-dark-800;
}

.workbench-navigator-mode {
  @apply min-h-11 rounded px-2 text-xs font-medium text-gray-500 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-400 dark:hover:text-white;
}

.workbench-navigator-mode-active {
  @apply bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white;
}

.workbench-navigator-list {
  @apply min-h-0 flex-1 overflow-y-auto px-2 pb-3;
  overscroll-behavior: contain;
}

.workbench-batch-item {
  @apply flex min-h-[52px] min-w-0 items-stretch rounded-md text-left text-xs transition-colors hover:bg-white dark:hover:bg-dark-800;
}

.workbench-batch-item-active {
  @apply border-l-[3px] border-l-primary-500 bg-primary-50/70 dark:border-l-primary-400 dark:bg-primary-900/20;
}

.workbench-batch-main {
  @apply flex min-h-[52px] min-w-0 flex-1 items-center gap-2 px-2.5 py-2 text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500;
}

.workbench-batch-meta {
  @apply mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400;
}

.workbench-batch-context {
  @apply mb-1 flex min-h-11 items-center justify-between gap-2 border-b border-gray-200 px-2 text-xs font-medium text-gray-600 dark:border-dark-700 dark:text-gray-300;
}

.workbench-context-clear {
  @apply inline-flex min-h-11 shrink-0 items-center px-2 text-xs font-semibold text-primary-600 hover:text-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-primary-400;
}

.workbench-sort-control {
  @apply flex min-h-11 items-center gap-1.5 rounded-md border border-gray-200 bg-white px-2 text-gray-500 focus-within:border-primary-400 focus-within:ring-2 focus-within:ring-primary-100 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300 dark:focus-within:ring-primary-900/40;
}

.workbench-sort-control select {
  @apply min-h-10 max-w-40 cursor-pointer border-0 bg-transparent pr-6 text-xs font-medium text-gray-700 outline-none focus:ring-0 dark:text-gray-200;
}

.workbench-view-all {
  @apply flex min-h-11 w-full items-center justify-center gap-1 text-xs font-medium text-primary-600 hover:bg-primary-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-50 dark:text-primary-400 dark:hover:bg-primary-900/20;
}

.workbench-navigator-empty {
  @apply px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400;
}

.workbench-layout {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

@media (max-width: 1439px) {
  .account-operation-band {
    flex-wrap: wrap;
    align-items: center;
  }

  .account-operation-filters,
  .account-operation-actions {
    display: contents;
  }

  .account-operation-filters :deep(> div:first-child) {
    display: contents;
  }

  .account-operation-filters :deep(> div:first-child > *),
  .account-operation-actions :deep(> *) {
    order: 3;
  }

  .account-operation-filters :deep(> div:first-child > :first-child) {
    order: 1;
    width: auto;
    min-width: 0;
    flex: 1 1 240px;
  }

  .account-operation-actions :deep(> .btn-primary) {
    order: 2;
    flex: 0 0 auto;
  }

  .account-operation-filters :deep(> div:not(:first-child)) {
    order: 4;
    flex: 1 0 100%;
  }
}

@media (max-width: 1179px) {
  .workbench-navigator-open {
    display: inline-flex;
  }

  .workbench-navigator-backdrop {
    position: fixed;
    inset: 0;
    z-index: 60;
    display: block;
    background: rgb(15 23 42 / 0.45);
  }

  .workbench-sidebar.workbench-mobile-collapsed {
    display: none;
  }

  .workbench-sidebar:not(.workbench-mobile-collapsed) {
    position: fixed;
    inset: max(12px, env(safe-area-inset-top)) 12px max(12px, env(safe-area-inset-bottom));
    z-index: 61;
    display: flex;
    width: min(420px, calc(100vw - 24px));
    min-height: 0;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid rgb(226 232 240);
    border-radius: 8px;
    background: white;
    box-shadow: 0 20px 48px rgb(15 23 42 / 0.24);
  }

  .dark .workbench-sidebar:not(.workbench-mobile-collapsed) {
    border-color: rgb(51 65 85);
    background: rgb(15 23 42);
  }

  .workbench-sidebar:not(.workbench-mobile-collapsed) .workbench-navigator-list {
    display: block;
    min-height: 0;
    flex: 1 1 auto;
    overflow-y: auto;
  }

  .workbench-sidebar:not(.workbench-mobile-collapsed) .workbench-nav-item,
  .workbench-sidebar:not(.workbench-mobile-collapsed) .workbench-batch-item {
    width: 100%;
    min-width: 0;
  }

  .workbench-mobile-toggle {
    display: inline-flex;
  }
}

@media (min-width: 1180px) {
  .workbench-layout {
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr);
  }

  .workbench-sidebar {
    display: flex;
    min-height: 0;
    flex-direction: column;
    border-right: 1px solid rgb(226 232 240);
    border-bottom: 0;
  }

  .dark .workbench-sidebar {
    border-right-color: rgb(51 65 85);
  }

  .workbench-navigator-list {
    display: block;
    min-height: 0;
    flex: 1 1 auto;
    overflow-y: auto;
  }

  .workbench-nav-item,
  .workbench-batch-item {
    width: 100%;
    min-width: 0;
  }

}

.account-usage-card {
  @apply min-w-[232px] rounded-lg border border-gray-200/80 bg-gray-50/70 px-3 py-2 dark:border-dark-700 dark:bg-dark-800/60;
}

.account-usage-finance {
  display: grid;
  min-width: 0;
  gap: 12px;
}

.account-finance-summary {
  width: 100%;
  padding-top: 12px;
  border-top: 1px solid rgb(226 232 240 / 0.9);
}

.account-workbench-table.is-finance-first .account-finance-summary {
  order: -1;
}

.account-workbench-table.is-compact :deep(.table-body tr[data-row-id] td) {
  padding-top: 8px;
  padding-bottom: 8px;
}

.account-workbench-table.is-compact :deep(.table-body tr[data-row-id]) {
  margin-top: 4px;
  margin-bottom: 4px;
}

.dark .account-finance-summary {
  border-top-color: rgb(51 65 85 / 0.8);
}

/* Preserve the regular account page table; only the embedded pool view uses the composite card grid. */
.account-workbench-table:not(.is-embedded) :deep(.table-wrapper table) {
  border-collapse: separate;
  border-spacing: 0 8px;
}

.account-workbench-table:not(.is-embedded) :deep(.table-body) {
  background: transparent;
}

.account-workbench-table:not(.is-embedded) :deep(.table-body tr[data-row-id]) {
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.05);
}

.account-workbench-table:not(.is-embedded) :deep(.table-body tr[data-row-id] td) {
  border-top: 1px solid rgb(226 232 240 / 0.9);
  border-bottom: 1px solid rgb(226 232 240 / 0.9);
  background: rgb(255 255 255 / 0.96);
}

.account-workbench-table:not(.is-embedded) :deep(.table-body tr[data-row-id] td:first-child) {
  border-left: 1px solid rgb(226 232 240 / 0.9);
  border-radius: 12px 0 0 12px;
}

.account-workbench-table:not(.is-embedded) :deep(.table-body tr[data-row-id] td:last-child) {
  border-right: 1px solid rgb(226 232 240 / 0.9);
  border-radius: 0 12px 12px 0;
}

.dark .account-workbench-table:not(.is-embedded) :deep(.table-body tr[data-row-id] td) {
  border-color: rgb(71 85 105 / 0.7);
  background: rgb(30 41 59 / 0.78);
}

@media (min-width: 768px) {
  .account-workbench-table.is-embedded :deep(.table-wrapper table),
  .account-workbench-table.is-embedded :deep(.table-body) {
    display: block;
    width: 100%;
    min-width: 0;
    background: transparent;
  }

  .account-workbench-table.is-embedded :deep(.table-header) {
    display: none;
  }

  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id]) {
    display: grid;
    width: 100%;
    min-width: 0;
    margin: 8px 0;
    overflow: hidden;
    border: 1px solid rgb(226 232 240 / 0.9);
    border-radius: 8px;
    background: rgb(255 255 255 / 0.96);
    box-shadow: 0 1px 2px rgb(15 23 42 / 0.05);
  }

  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id] td) {
    display: flex;
    min-width: 0;
    align-items: center;
    padding: 12px;
    white-space: normal;
    background: transparent;
  }

  .account-workbench-table.is-embedded :deep(.table-body tr:not([data-row-id])) {
    display: block;
    width: 100%;
  }

  .account-workbench-table.is-embedded :deep(.table-body tr:not([data-row-id]) td) {
    display: block;
    width: 100%;
  }

  .dark .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id]) {
    border-color: rgb(71 85 105 / 0.7);
    background: rgb(30 41 59 / 0.78);
  }
}

@media (min-width: 768px) and (max-width: 1439px) {
  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id]) {
    grid-template-columns: 44px minmax(0, 1fr) minmax(150px, 190px) 104px;
  }

  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id] td:nth-child(1)) {
    grid-column: 1;
    grid-row: 1;
  }

  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id] td:nth-child(2)) {
    grid-column: 2;
    grid-row: 1;
  }

  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id] td:nth-child(3)) {
    grid-column: 1 / 5;
    grid-row: 2;
    border-top: 1px solid rgb(241 245 249);
  }

  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id] td:nth-child(4)) {
    grid-column: 3;
    grid-row: 1;
  }

  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id] td:nth-child(5)) {
    grid-column: 4;
    grid-row: 1;
  }

  .account-usage-finance {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 16px;
  }

  .account-finance-summary {
    padding-top: 0;
    padding-left: 16px;
    border-top: 0;
    border-left: 1px solid rgb(226 232 240 / 0.9);
  }

  .dark .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id] td:nth-child(3)),
  .dark .account-finance-summary {
    border-color: rgb(51 65 85 / 0.8);
  }
}

@media (min-width: 1440px) {
  .account-workbench-table.is-embedded :deep(.table-body tr[data-row-id]) {
    grid-template-columns: 44px minmax(180px, 210px) minmax(300px, 1fr) minmax(190px, 220px) 104px;
  }

  .account-usage-finance {
    grid-template-columns: minmax(0, 1fr) minmax(0, 0.85fr);
    gap: 16px;
  }

  .account-finance-summary {
    padding-top: 0;
    padding-left: 16px;
    border-top: 0;
    border-left: 1px solid rgb(226 232 240 / 0.9);
  }

  .dark .account-finance-summary {
    border-top-color: transparent;
    border-left-color: rgb(51 65 85 / 0.8);
  }
}

.workbench-mobile-toggle {
  display: none;
}

@media (max-width: 1179px) {
  .workbench-mobile-toggle {
    display: inline-flex;
  }
}

@media (max-width: 767px) {
  .account-operation-band {
    gap: 8px;
    padding: 8px;
  }

  .workbench-mobile-toggle {
    display: inline-flex;
  }

  .account-workbench-table.is-embedded :deep(.space-y-3 > .rounded-lg > .space-y-3) {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 12px;
  }

  .account-workbench-table.is-embedded :deep(.space-y-3 > .rounded-lg > .space-y-3 > *) {
    order: 3;
  }

  .account-workbench-table.is-embedded :deep([data-field='select']) {
    order: 0;
  }

  .account-workbench-table.is-embedded :deep([data-field='name']) {
    order: 1;
  }

  .account-workbench-table.is-embedded :deep([data-field='status']) {
    order: 2;
  }

  .account-workbench-table.is-embedded :deep([data-field='usage']) {
    order: 4;
  }

  .account-workbench-table.is-embedded :deep([data-field='name']),
  .account-workbench-table.is-embedded :deep([data-field='usage']),
  .account-workbench-table.is-embedded :deep([data-field='status']) {
    display: block;
  }

  .account-workbench-table.is-embedded :deep([data-field='name'] > span),
  .account-workbench-table.is-embedded :deep([data-field='usage'] > span),
  .account-workbench-table.is-embedded :deep([data-field='status'] > span) {
    display: none;
  }

  .account-workbench-table.is-embedded :deep([data-field='name'] > div),
  .account-workbench-table.is-embedded :deep([data-field='usage'] > div),
  .account-workbench-table.is-embedded :deep([data-field='status'] > div) {
    width: 100%;
    max-width: none;
    text-align: left;
  }
}

.account-workbench-table.workbench-view-runtime .account-usage-finance,
.account-workbench-table.workbench-view-finance .account-usage-finance {
  grid-template-columns: minmax(0, 1fr);
}

@media (min-width: 768px) {
  .account-workbench-table.workbench-view-finance .account-finance-summary {
    padding-left: 0;
    border-left: 0;
  }
}
</style>
