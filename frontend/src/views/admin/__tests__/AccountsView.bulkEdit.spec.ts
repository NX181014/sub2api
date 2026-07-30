import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listRows,
  listWithEtag,
  getBatchTodayStats,
  getUpstreamBillingProbeSettings,
  getAllProxies,
  getAllGroups,
  probeUpstreamBillingBatch,
  bulkDelete,
  batchClearError,
  batchRefresh,
  getSelectionSummary,
  listImportBatch,
  listUsers
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listRows: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getUpstreamBillingProbeSettings: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn(),
  probeUpstreamBillingBatch: vi.fn(),
  bulkDelete: vi.fn(),
  batchClearError: vi.fn(),
  batchRefresh: vi.fn(),
  getSelectionSummary: vi.fn(),
  listImportBatch: vi.fn(),
  listUsers: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listRows,
      listWithEtag,
      getBatchTodayStats,
      getUpstreamBillingProbeSettings,
      delete: vi.fn(),
      batchClearError,
      batchRefresh,
      bulkDelete,
      getSelectionSummary,
      listImportBatch,
      probeUpstreamBillingBatch,
      toggleSchedulable: vi.fn()
    },
    proxies: {
      getAll: getAllProxies
    },
    groups: {
      getAll: getAllGroups
    },
    users: {
      list: listUsers
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    token: 'test-token'
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      te: () => false
    })
  }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div data-test="data-table">
      <div data-test="select-header"><slot name="header-select" /></div>
      <span v-for="column in columns" :key="column.key" data-test="column-key">{{ column.key }}</span>
      <div v-for="row in data" :key="row.id">
        <div data-test="select-row"><slot name="cell-select" :row="row" /></div>
        <slot name="cell-name" :value="row.name" :row="row" />
        <slot name="cell-created_at" :value="row.created_at" :row="row" />
      </div>
    </div>
  `
}

const AccountBulkActionsBarStub = {
  props: ['selectedIds', 'filteredCount', 'hasActiveFilters', 'hiddenSelectedCount', 'allPageSelected', 'pageSelectedCount', 'busy'],
  emits: ['delete', 'edit-selected', 'edit-filtered', 'probe-upstream-billing', 'toggle-page', 'reset-status', 'refresh-token'],
  template: `
    <div>
      <button data-test="edit-filtered" @click="$emit('edit-filtered')">edit filtered</button>
      <button data-test="edit-selected" @click="$emit('edit-selected')">edit selected</button>
      <button data-test="probe-upstream-billing" @click="$emit('probe-upstream-billing')">probe</button>
      <button data-test="bulk-delete" @click="$emit('delete')">delete</button>
      <button data-test="bulk-reset" @click="$emit('reset-status')">reset</button>
      <button data-test="bulk-refresh" @click="$emit('refresh-token')">refresh</button>
      <button data-test="toggle-page" @click="$emit('toggle-page')">toggle page</button>
    </div>
  `
}

const AccountTableFiltersStub = {
  props: ['filters', 'searchQuery'],
  emits: ['update:filters', 'update:searchQuery', 'clear'],
  template: `
    <div>
      <button data-test="filter-openai" @click="$emit('update:filters', { ...filters, platform: 'openai' })">filter</button>
      <button data-test="clear-filters" @click="$emit('clear')">clear</button>
    </div>
  `
}

const PaginationStub = {
  emits: ['update:page'],
  template: '<button data-test="next-page" @click="$emit(\'update:page\', 2)">next</button>'
}

const BulkEditAccountModalStub = {
  props: ['show', 'target'],
  emits: ['updated'],
  template: '<div data-test="bulk-edit-modal" :data-show="String(show)" :data-target-mode="target?.mode ?? \'\'" :data-platforms="target?.selectedPlatforms?.join(\',\') ?? \'\'" :data-uploader-unassigned="String(target?.filters?.uploader_unassigned ?? false)"><button data-test="bulk-updated-partial" @click="$emit(\'updated\', [2])">updated</button></div>'
}

const account = (id: number) => ({
  id,
  name: `account-${id}`,
  platform: 'openai',
  type: 'apikey',
  status: 'active',
  schedulable: true,
  created_at: '2026-07-29T00:00:00Z',
  updated_at: '2026-07-29T00:00:00Z'
})

const batchRowsResponse = (batchID: string, matchedCount: number) => ({
  items: [{
    kind: 'import_batch',
    batch: {
      id: batchID,
      created_at: '2026-07-29T00:00:00Z',
      matched_count: matchedCount,
      total_count: matchedCount,
      schedulable_count: matchedCount,
      names: ['account-1'],
      status: { normal: matchedCount, error: 0, inactive: 0, rate_limited: 0, overloaded: 0, temp_unschedulable: 0, manual_unschedulable: 0 }
    }
  }],
  total: 1,
  page: 1,
  page_size: 20,
  pages: 1
})

const mountAccountsView = (stubs: Record<string, unknown> = {}) => mount(AccountsView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
      DataTable: DataTableStub,
      Pagination: PaginationStub,
      ConfirmDialog: true,
      AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
      AccountTableFilters: AccountTableFiltersStub,
      AccountBulkActionsBar: AccountBulkActionsBarStub,
      AccountActionMenu: true,
      ImportDataModal: true,
      ReAuthAccountModal: true,
      AccountTestModal: true,
      AccountStatsModal: true,
      ScheduledTestsPanel: true,
      SyncFromCrsModal: true,
      TempUnschedStatusModal: true,
      ErrorPassthroughRulesModal: true,
      TLSFingerprintProfilesModal: true,
      CreateAccountModal: true,
      EditAccountModal: true,
      BulkEditAccountModal: BulkEditAccountModalStub,
      PlatformTypeBadge: true,
      AccountCapacityCell: true,
      AccountStatusIndicator: true,
      AccountTodayStatsCell: true,
      AccountGroupsCell: true,
      AccountUsageCell: true,
      Icon: true,
      ...stubs
    }
  }
})

describe('admin AccountsView bulk edit scope', () => {
  beforeEach(() => {
    localStorage.clear()

    listAccounts.mockReset()
    listRows.mockReset()
    listWithEtag.mockReset()
    getBatchTodayStats.mockReset()
    getUpstreamBillingProbeSettings.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()
    probeUpstreamBillingBatch.mockReset()
    bulkDelete.mockReset()
		batchClearError.mockReset()
		batchRefresh.mockReset()
		getSelectionSummary.mockReset()
		listImportBatch.mockReset()
		listUsers.mockReset()

    listAccounts.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    listRows.mockImplementation(async (...args) => {
      const result = await listAccounts(...args)
      return { ...result, items: result.items.map((account: unknown) => ({ kind: 'account', account })) }
    })
    listWithEtag.mockResolvedValue({
      notModified: true,
      etag: null,
      data: null
    })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getUpstreamBillingProbeSettings.mockResolvedValue({ enabled: true, interval_minutes: 30 })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
    probeUpstreamBillingBatch.mockResolvedValue([])
    bulkDelete.mockResolvedValue({ deleted: 0, approval_required: 0, failed: 0, results: [] })
		batchClearError.mockResolvedValue({ total: 0, success: 0, failed: 0, errors: [] })
		batchRefresh.mockResolvedValue({ total: 0, success: 0, failed: 0, errors: [] })
		getSelectionSummary.mockResolvedValue({ total: 0, platforms: ['openai'], types: [] })
		listImportBatch.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 0 })
		listUsers.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 200, pages: 0 })
  })

  it('only opens filtered bulk edit after an effective filter and does not preview the first 100 rows', async () => {
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: AccountTableFiltersStub,
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    expect(wrapper.find('[data-test="edit-filtered"]').exists()).toBe(false)
    wrapper.getComponent(AccountTableFiltersStub).vm.$emit('update:filters', { platform: 'openai' })
    await flushPromises()
    await wrapper.get('[data-test="edit-filtered"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-show')).toBe('true')
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-target-mode')).toBe('filtered')
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-platforms')).toBe('openai')
    expect(listAccounts).toHaveBeenCalledTimes(2)
    expect(getSelectionSummary).toHaveBeenCalledWith(expect.objectContaining({ platform: 'openai' }))
  })

  it('runs only one filtered selection summary while the async preflight is pending', async () => {
    let finishSummary!: (value: { total: number; platforms: string[]; types: string[] }) => void
    getSelectionSummary.mockImplementationOnce(() => new Promise(resolve => { finishSummary = resolve }))
    const wrapper = mountAccountsView()
    await flushPromises()
    wrapper.getComponent(AccountTableFiltersStub).vm.$emit('update:filters', { platform: 'openai' })
    await flushPromises()

    await wrapper.get('[data-test="edit-filtered"]').trigger('click')
    await wrapper.get('[data-test="edit-filtered"]').trigger('click')
    expect(getSelectionSummary).toHaveBeenCalledTimes(1)

    finishSummary({ total: 12, platforms: ['openai'], types: ['oauth', 'apikey'] })
    await flushPromises()
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-show')).toBe('true')
  })

  it('preserves the unassigned uploader filter in the exact selection summary', async () => {
    const wrapper = mountAccountsView()
    await flushPromises()
    wrapper.getComponent(AccountTableFiltersStub).vm.$emit('update:filters', { uploader_user_id: 'unassigned' })
    await flushPromises()
    await wrapper.get('[data-test="edit-filtered"]').trigger('click')
    await flushPromises()

    expect(getSelectionSummary).toHaveBeenCalledWith(expect.objectContaining({ uploader_user_id: 'unassigned' }))
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-uploader-unassigned')).toBe('true')
  })

  it('clears cross-page selection and returns to page one when filters change', async () => {
    listAccounts
      .mockResolvedValueOnce({ items: [account(7)], total: 2, page: 1, page_size: 1, pages: 2 })
      .mockResolvedValueOnce({ items: [account(11)], total: 2, page: 2, page_size: 1, pages: 2 })
      .mockResolvedValueOnce({ items: [account(11)], total: 1, page: 1, page_size: 1, pages: 1 })

    const wrapper = mountAccountsView()
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="next-page"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')

    const bulkBar = wrapper.getComponent(AccountBulkActionsBarStub)
    expect(bulkBar.props('selectedIds')).toEqual([7, 11])
    expect(bulkBar.props('hiddenSelectedCount')).toBe(1)

    wrapper.getComponent(AccountTableFiltersStub).vm.$emit('update:filters', { platform: 'openai' })
    await flushPromises()

    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('selectedIds')).toEqual([])
    expect(listAccounts.mock.calls.at(-1)?.[0]).toBe(1)
    expect(listAccounts.mock.calls.at(-1)?.[2]).toMatchObject({ platform: 'openai' })
  })

  it('keeps the header checkbox in unchecked, mixed and checked page states', async () => {
    listAccounts.mockResolvedValue({ items: [account(1), account(2)], total: 2, page: 1, page_size: 20, pages: 1 })
    const wrapper = mountAccountsView()
    await flushPromises()

    const header = () => wrapper.get('[data-test="select-header"] input').element as HTMLInputElement
    expect(header().checked).toBe(false)
    expect(header().indeterminate).toBe(false)

    await wrapper.findAll('[data-test="select-row"] input')[0]!.trigger('change')
    expect(header().checked).toBe(false)
    expect(header().indeterminate).toBe(true)

    await wrapper.get('[data-test="select-header"] input').setValue(true)
    expect(header().checked).toBe(true)
    expect(header().indeterminate).toBe(false)
    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('selectedIds')).toEqual([1, 2])

    await wrapper.get('[data-test="toggle-page"]').trigger('click')
    expect(header().checked).toBe(false)
    expect(wrapper.findComponent(AccountBulkActionsBarStub).exists()).toBe(false)
  })

  it('loads every import-batch page before counting, expanding or selecting the batch', async () => {
    const batchID = 'batch-101'
    const batchAccount = (id: number) => ({ ...account(id), extra: { import_batch_id: batchID } })
    listAccounts.mockResolvedValue({ items: [batchAccount(1)], total: 1, page: 1, page_size: 20, pages: 1 })
    listRows.mockResolvedValue(batchRowsResponse(batchID, 101))
    listImportBatch
      .mockResolvedValueOnce({ items: Array.from({ length: 100 }, (_, index) => batchAccount(index + 1)), total: 101, page: 1, page_size: 100, pages: 2 })
      .mockResolvedValueOnce({ items: [batchAccount(101)], total: 101, page: 2, page_size: 100, pages: 2 })

    const wrapper = mountAccountsView()
    await flushPromises()

    expect(listImportBatch).toHaveBeenNthCalledWith(1, batchID, 1, 100, expect.any(Object))
    expect(listImportBatch).toHaveBeenNthCalledWith(2, batchID, 2, 100, expect.any(Object))
    const batchRow = (wrapper.getComponent(DataTableStub).props('data') as any[])[0]
    expect(batchRow.accounts).toHaveLength(101)

    await wrapper.get('[data-test="select-row"] input').setValue(true)
    await flushPromises()
    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('selectedIds')).toHaveLength(101)
    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('hiddenSelectedCount')).toBe(0)

    await wrapper.get('[data-test="data-table"] button').trigger('click')
    await flushPromises()
    expect(wrapper.getComponent(DataTableStub).props('data')).toHaveLength(102)
  })

  it('reloads batch members with current filters and ignores a stale in-flight batch response', async () => {
    const batchID = 'batch-filtered'
    const batchAccount = (id: number) => ({ ...account(id), extra: { import_batch_id: batchID } })
    let finishStale!: (value: { items: ReturnType<typeof batchAccount>[]; total: number; page: number; page_size: number; pages: number }) => void
    listAccounts
      .mockResolvedValueOnce({ items: [batchAccount(1)], total: 1, page: 1, page_size: 20, pages: 1 })
    listRows
      .mockResolvedValueOnce(batchRowsResponse(batchID, 1))
      .mockResolvedValueOnce(batchRowsResponse(batchID, 1))
      .mockResolvedValueOnce({ items: [batchAccount(1)], total: 1, page: 1, page_size: 20, pages: 1 })
    listImportBatch
      .mockImplementationOnce(() => new Promise(resolve => { finishStale = resolve }))
      .mockResolvedValueOnce({ items: [batchAccount(2)], total: 1, page: 1, page_size: 100, pages: 1 })

    const wrapper = mountAccountsView()
    await flushPromises()
    wrapper.getComponent(AccountTableFiltersStub).vm.$emit('update:filters', { platform: 'openai' })
    await flushPromises()

    expect(listImportBatch).toHaveBeenCalledTimes(2)
    expect(listImportBatch.mock.calls[1]?.[3]).toMatchObject({ platform: 'openai' })
    expect((wrapper.getComponent(DataTableStub).props('data') as any[])[0].accounts.map((item: any) => item.id)).toEqual([2])

    finishStale({ items: [batchAccount(99)], total: 1, page: 1, page_size: 100, pages: 1 })
    await flushPromises()
    expect((wrapper.getComponent(DataTableStub).props('data') as any[])[0].accounts.map((item: any) => item.id)).toEqual([2])
  })

  it.each([
    ['clear-error', '[data-test="bulk-reset"]', batchClearError],
    ['refresh', '[data-test="bulk-refresh"]', batchRefresh]
  ])('keeps only failed IDs after a partial %s operation', async (_name, selector, operation) => {
    listAccounts.mockResolvedValue({ items: [account(1), account(2)], total: 2, page: 1, page_size: 20, pages: 1 })
    operation.mockResolvedValueOnce({ total: 2, success: 1, failed: 1, errors: [{ account_id: 2, error: 'failed' }] })
    vi.stubGlobal('confirm', vi.fn(() => true))
    const wrapper = mountAccountsView()
    await flushPromises()
    await wrapper.findAll('[data-test="select-row"] input')[0]!.trigger('change')
    await wrapper.findAll('[data-test="select-row"] input')[1]!.trigger('change')

    await wrapper.get(selector).trigger('click')
    await flushPromises()

    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('selectedIds')).toEqual([2])
    vi.unstubAllGlobals()
  })

  it('retains the original selection when a partial operation omits failed IDs', async () => {
    listAccounts.mockResolvedValue({ items: [account(1), account(2)], total: 2, page: 1, page_size: 20, pages: 1 })
    batchClearError.mockResolvedValueOnce({ total: 2, success: 1, failed: 1, errors: [] })
    vi.stubGlobal('confirm', vi.fn(() => true))
    const wrapper = mountAccountsView()
    await flushPromises()
    await wrapper.findAll('[data-test="select-row"] input')[0]!.trigger('change')
    await wrapper.findAll('[data-test="select-row"] input')[1]!.trigger('change')

    await wrapper.get('[data-test="bulk-reset"]').trigger('click')
    await flushPromises()

    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('selectedIds')).toEqual([1, 2])
    vi.unstubAllGlobals()
  })

  it('keeps only failed IDs after a partially successful bulk edit', async () => {
    listAccounts.mockResolvedValue({ items: [account(1), account(2)], total: 2, page: 1, page_size: 20, pages: 1 })
    const wrapper = mountAccountsView()
    await flushPromises()
    await wrapper.findAll('[data-test="select-row"] input')[0]!.trigger('change')
    await wrapper.findAll('[data-test="select-row"] input')[1]!.trigger('change')
    await wrapper.get('[data-test="edit-selected"]').trigger('click')
    await wrapper.get('[data-test="bulk-updated-partial"]').trigger('click')
    await flushPromises()

    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('selectedIds')).toEqual([2])
  })

  it('ignores repeated bulk delete triggers while the first request is pending', async () => {
    let finishDelete!: (value: { deleted: number; approval_required: number; failed: number; results: never[] }) => void
    bulkDelete.mockImplementationOnce(() => new Promise(resolve => { finishDelete = resolve }))
    listAccounts.mockResolvedValue({ items: [account(1)], total: 1, page: 1, page_size: 20, pages: 1 })
    vi.stubGlobal('confirm', vi.fn(() => true))

    const wrapper = mountAccountsView()
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="bulk-delete"]').trigger('click')
    await wrapper.get('[data-test="bulk-delete"]').trigger('click')

    expect(bulkDelete).toHaveBeenCalledTimes(1)
    expect(wrapper.getComponent(AccountBulkActionsBarStub).props('busy')).toBe(true)
    finishDelete({ deleted: 1, approval_required: 0, failed: 0, results: [] })
    await flushPromises()
    vi.unstubAllGlobals()
  })

	it('splits more than 100 selected accounts into bounded delete requests', async () => {
		const items = Array.from({ length: 101 }, (_, index) => ({
			id: index + 1,
			name: `account-${index + 1}`,
			platform: 'anthropic',
			type: 'oauth',
			status: 'active',
			schedulable: true,
			created_at: '2026-07-29T00:00:00Z',
			updated_at: '2026-07-29T00:00:00Z'
		}))
		listAccounts.mockResolvedValue({ items, total: items.length, page: 1, page_size: items.length, pages: 1 })
		bulkDelete
			.mockResolvedValueOnce({ deleted: 100, approval_required: 0, failed: 0, results: [] })
			.mockResolvedValueOnce({ deleted: 1, approval_required: 0, failed: 0, results: [] })
		vi.stubGlobal('confirm', vi.fn(() => true))

		const wrapper = mount(AccountsView, {
			global: {
				stubs: {
					AppLayout: { template: '<div><slot /></div>' },
					TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
					DataTable: DataTableStub,
					Pagination: true,
					ConfirmDialog: true,
					AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
					AccountTableFilters: true,
					AccountBulkActionsBar: AccountBulkActionsBarStub,
					AccountActionMenu: true,
					ImportDataModal: true,
					ReAuthAccountModal: true,
					AccountTestModal: true,
					AccountStatsModal: true,
					ScheduledTestsPanel: true,
					SyncFromCrsModal: true,
					TempUnschedStatusModal: true,
					ErrorPassthroughRulesModal: true,
					TLSFingerprintProfilesModal: true,
					CreateAccountModal: true,
					EditAccountModal: true,
					BulkEditAccountModal: true,
					PlatformTypeBadge: true,
					AccountCapacityCell: true,
					AccountStatusIndicator: true,
					AccountTodayStatsCell: true,
					AccountGroupsCell: true,
					AccountUsageCell: true,
					Icon: true
				}
			}
		})

		await flushPromises()
		for (const checkbox of wrapper.findAll('[data-test="select-row"] input')) await checkbox.setValue(true)
		await wrapper.get('[data-test="bulk-delete"]').trigger('click')
		await flushPromises()

		expect(bulkDelete).toHaveBeenCalledTimes(2)
		expect(bulkDelete.mock.calls[0]?.[0]).toHaveLength(100)
		expect(bulkDelete.mock.calls[1]?.[0]).toHaveLength(1)
		vi.unstubAllGlobals()
	})

  it('renders the created_at column by default', async () => {
    listAccounts.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'test-account',
          platform: 'anthropic',
          type: 'oauth',
          status: 'active',
          schedulable: true,
          created_at: '2026-03-07T10:00:00Z',
          updated_at: '2026-03-07T10:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()

    const columnKeys = wrapper.findAll('[data-test="column-key"]').map(node => node.text())
    expect(columnKeys).toContain('created_at')
    const columns = wrapper.getComponent(DataTableStub).props('columns') as Array<{ key: string; label: string; sortable: boolean }>
    expect(columns.find(column => column.key === 'created_at')).toMatchObject({
      label: 'admin.accounts.columns.createdAt',
      sortable: true
    })
  })

  it('passes the loaded global probe state to every upstream billing cell', async () => {
    listAccounts.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'upstream',
          platform: 'openai',
          type: 'apikey',
          status: 'active',
          schedulable: true,
          created_at: '2026-07-13T00:00:00Z',
          updated_at: '2026-07-13T00:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getUpstreamBillingProbeSettings.mockResolvedValue({ enabled: false, interval_minutes: 30 })

    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-upstream_billing_rate" :row="row" /></div></div>'
          },
          UpstreamBillingRateCell: {
            props: ['globalProbeEnabled'],
            template: '<span data-test="upstream-billing-cell" :data-global-enabled="String(globalProbeEnabled)"></span>'
          },
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: true,
          AccountTableFilters: true,
          AccountBulkActionsBar: true,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: true,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(getUpstreamBillingProbeSettings).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-test="upstream-billing-cell"]').attributes('data-global-enabled')).toBe('false')
  })

  it('submits selected account IDs from every page for backend eligibility checks', async () => {
    const account = (id: number) => ({
      id,
      name: `account-${id}`,
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      created_at: '2026-07-13T00:00:00Z',
      updated_at: '2026-07-13T00:00:00Z'
    })
    listAccounts
      .mockResolvedValueOnce({ items: [account(7)], total: 2, page: 1, page_size: 1, pages: 2 })
      .mockResolvedValueOnce({ items: [account(11)], total: 2, page: 2, page_size: 1, pages: 2 })

    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="table" /><slot name="pagination" /></div>' },
          DataTable: DataTableStub,
          Pagination: PaginationStub,
          ConfirmDialog: true,
          AccountTableActions: true,
          AccountTableFilters: true,
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="next-page"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="probe-upstream-billing"]').trigger('click')
    await flushPromises()

    expect(probeUpstreamBillingBatch).toHaveBeenCalledWith([7, 11])
  })

  it('keeps the logical-row order after a batch probe changes a snapshot', async () => {
    const account = (id: number) => ({
      id,
      name: `account-${id}`,
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      created_at: '2026-07-13T00:00:00Z',
      updated_at: '2026-07-13T00:00:00Z'
    })
    listAccounts
      .mockResolvedValueOnce({ items: [account(7)], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [account(7)], total: 1, page: 1, page_size: 20, pages: 1 })
    probeUpstreamBillingBatch.mockResolvedValue([
      {
        account_id: 7,
        snapshot: {
          status: 'ok',
          data: { effective_rate_multiplier: 0.5 },
          last_attempt_at: '2026-07-13T00:00:00Z',
          next_probe_at: '2026-07-13T00:30:00Z'
        }
      }
    ])

    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="table" /></div>' },
          DataTable: DataTableStub,
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountTableActions: true,
          AccountTableFilters: true,
          AccountActionMenu: true,
          Pagination: true,
          ConfirmDialog: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="probe-upstream-billing"]').trigger('click')
    await flushPromises()

    expect(probeUpstreamBillingBatch).toHaveBeenCalledWith([7])
    expect(listAccounts).toHaveBeenCalledTimes(1)
  })
})
