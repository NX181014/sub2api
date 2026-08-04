import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CostLedgerPanel from '../CostLedgerPanel.vue'

const {
  listLedgerEntries,
  listCostUploaderSummaries,
  listCostSummaries,
  listPurchaseSources,
  listAccounts,
  listUsers
} = vi.hoisted(() => ({
  listLedgerEntries: vi.fn(),
  listCostUploaderSummaries: vi.fn(),
  listCostSummaries: vi.fn(),
  listPurchaseSources: vi.fn(),
  listAccounts: vi.fn(),
  listUsers: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    sharedPool: {
      listLedgerEntries,
      listCostUploaderSummaries,
      listCostSummaries,
      listPurchaseSources
    },
    accounts: { list: listAccounts },
    users: { list: listUsers }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      te: () => false,
      locale: { value: 'zh-CN' }
    })
  }
})

describe('CostLedgerPanel route state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listLedgerEntries.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    listCostUploaderSummaries.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    listCostSummaries.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    listPurchaseSources.mockResolvedValue([])
    listAccounts.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 200, pages: 0 })
    listUsers.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 200, pages: 0 })
  })

  it('reloads the current account and page when browser history changes', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/?ledger_view=entries&account_id=7&ledger_entry_page=2')
    await router.isReady()
    mount(CostLedgerPanel, {
      global: {
        plugins: [router],
        stubs: {
          BaseDialog: true,
          DataTable: true,
          EmptyState: true,
          LoadingSpinner: true,
          Pagination: true,
          SearchInput: true,
          Select: true,
          StatusBadge: true,
          Icon: true
        }
      }
    })
    await flushPromises()

    expect(listLedgerEntries).toHaveBeenLastCalledWith(expect.objectContaining({ account_id: 7, page: 2 }))

    await router.push('/?ledger_view=entries&account_id=9&ledger_entry_page=3')
    await flushPromises()

    expect(listLedgerEntries).toHaveBeenLastCalledWith(expect.objectContaining({ account_id: 9, page: 3 }))
  })

  it('loads uploader account details in a dialog', async () => {
    listCostUploaderSummaries.mockResolvedValueOnce({
      items: [{ uploader_user_id: 3, uploader_username: 'Uploader A', account_count: 1, net_cost_minor: 1000, recognized_cost_minor: 400, remaining_cost_minor: 600 }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    listCostSummaries.mockResolvedValueOnce({ items: [], total: 0, page: 1, page_size: 10, pages: 0 })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/?ledger_view=summary')
    await router.isReady()
    const wrapper = mount(CostLedgerPanel, {
      global: {
        plugins: [router],
        stubs: {
          BaseDialog: { props: ['show'], template: '<div v-if="show" data-test="uploader-dialog"><slot /></div>' },
          DataTable: true,
          EmptyState: true,
          LoadingSpinner: true,
          Pagination: true,
          SearchInput: true,
          Select: true,
          StatusBadge: true,
          Icon: true
        }
      }
    })
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text().includes('Uploader A'))!.trigger('click')
    await flushPromises()

    expect(listCostSummaries).toHaveBeenLastCalledWith(expect.objectContaining({ uploader_user_id: 3, page: 1 }))
    expect(wrapper.find('[data-test="uploader-dialog"]').exists()).toBe(true)
  })
})
