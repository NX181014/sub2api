import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SharedPoolView from '../SharedPoolView.vue'

const {
  getOverview,
  listSources,
  listAccountCosts,
  listPurchaseSources,
  getLatestFXRate
} = vi.hoisted(() => ({
  getOverview: vi.fn(),
  listSources: vi.fn(),
  listAccountCosts: vi.fn(),
  listPurchaseSources: vi.fn(),
  getLatestFXRate: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    sharedPool: {
      getOverview,
      listSources,
      listAccountCosts,
      listPurchaseSources,
      getLatestFXRate
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: { id: 1 } })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh-CN' }
    })
  }
})

vi.mock('vue-chartjs', () => ({ Bar: { template: '<div />' } }))
vi.mock('chart.js', () => ({
  BarElement: {},
  CategoryScale: {},
  Chart: { register: vi.fn() },
  Legend: {},
  LinearScale: {},
  Tooltip: {}
}))

describe('SharedPoolView route state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getOverview.mockResolvedValue({
      summary: {
        total_accounts: 0,
        active_accounts: 0,
        recovered_accounts: 0,
        banned_accounts: 0,
        total_purchase_cost: 0,
        total_usage_value: 0,
        pending_recovery: 0,
        banned_loss: 0,
        roi_rate: 0
      },
      accounts: [],
      period_start: '2026-07-01',
      period_end: '2026-08-01',
      currency: 'CNY'
    })
    listSources.mockResolvedValue({ items: [] })
    listAccountCosts.mockResolvedValue({ items: [], total: 0 })
    listPurchaseSources.mockResolvedValue([])
    getLatestFXRate.mockResolvedValue(1)
  })

  it('follows tab history and keeps account_id in overview and source requests', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/?tab=overview&account_id=7')
    await router.isReady()
    const wrapper = mount(SharedPoolView, {
      global: {
        plugins: [router],
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          AccountsView: {
            props: ['initialWorkbenchContext'],
            template: '<div data-test="accounts-context">{{ initialWorkbenchContext.scope }}|{{ initialWorkbenchContext.import_batch_id }}|{{ initialWorkbenchContext.usage_status }}|{{ initialWorkbenchContext.page }}</div>'
          },
          AccountTracePanel: true,
          CostLedgerPanel: true,
          BaseDialog: true,
          ConfirmDialog: true,
          DataTable: true,
          DateRangePicker: true,
          EmptyState: true,
          FormDialogActions: true,
          Icon: true,
          LoadingSpinner: true,
          Pagination: true,
          Select: true,
          StatCard: true,
          StatusBadge: true,
          Toggle: true
        }
      }
    })
    await flushPromises()

    expect(getOverview).toHaveBeenLastCalledWith(expect.objectContaining({ account_id: 7 }))

    listSources.mockResolvedValueOnce({
      items: [
        { uploader_user_id: 1, uploader_name: 'uploader-a', account_count: 1, purchase_cost: 10, usage_value: 8, roi_rate: 80, ban_rate_30d: 0, sources: [{ name: 'source-a', account_count: 1, sample_size: 1, purchase_cost: 10, usage_value: 8, roi_rate: 80, ban_rate_7d: 0, ban_rate_30d: 0, ban_rate_90d: 0, refund_rate: 0, average_survival_days: 30, accounts: [] }] },
        { uploader_user_id: 2, uploader_name: 'uploader-b', account_count: 1, purchase_cost: 20, usage_value: 22, roi_rate: 110, ban_rate_30d: 0, sources: [{ name: 'source-a', account_count: 1, sample_size: 1, purchase_cost: 20, usage_value: 22, roi_rate: 110, ban_rate_7d: 0, ban_rate_30d: 0, ban_rate_90d: 0, refund_rate: 0, average_survival_days: 20, accounts: [] }] }
      ]
    })
    await router.push('/?tab=sources&account_id=7')
    await flushPromises()

    expect(listSources).toHaveBeenLastCalledWith(expect.objectContaining({ account_id: 7 }))
    expect(wrapper.findAll('h3').filter((node) => node.text() === 'source-a')).toHaveLength(1)

    await router.push('/?tab=accounts&account_scope=batch&import_batch_id=batch-9&account_usage_status=ready&account_page=3')
    await flushPromises()

    expect(wrapper.get('[data-test="accounts-context"]').text()).toBe('batch|batch-9|ready|3')
  })
})
