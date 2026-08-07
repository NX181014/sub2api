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
    getOverview.mockResolvedValueOnce({
      summary: {
        total_accounts: 2,
        active_accounts: 1,
        recovered_accounts: 1,
        banned_accounts: 0,
        total_purchase_cost: 20,
        total_usage_value: 13,
        pending_recovery: 9,
        banned_loss: 0,
        roi_rate: 65
      },
      accounts: [
        { id: 1, account_id: 1, account_name: 'Healthy account', provider_identity: 'healthy', contributor_name: 'payer', uploader_name: 'uploader', purchase_source_name: 'source', purchase_cost: 10, currency: 'CNY', service_start: '2026-06-01', service_end: '2026-07-01', status: 'active', account_status: 'active', availability_status: 'normal', usage_value: 12, roi_rate: 120, remaining_cost: 0, banned_loss: 0, net_profit: 2 },
        { id: 2, account_id: 2, account_name: 'Attention account', provider_identity: 'attention', contributor_name: 'payer', uploader_name: 'uploader', purchase_source_name: 'source', purchase_cost: 10, currency: 'CNY', service_start: '2026-06-01', service_end: '2026-07-01', status: 'warning', account_status: 'error', availability_status: 'error', usage_value: 1, roi_rate: 10, remaining_cost: 9, banned_loss: 0, net_profit: -9 }
      ],
      period_start: '2026-06-01',
      period_end: '2026-06-30',
      currency: 'CNY'
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/?tab=overview&account_id=7&period_type=custom&period_start=2026-06-01&period_end=2026-06-30')
    await router.isReady()
    const wrapper = mount(SharedPoolView, {
      global: {
        plugins: [router],
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          AccountsView: {
            props: ['initialWorkbenchContext'],
            template: '<div data-test="accounts-context">{{ initialWorkbenchContext.axis }}|{{ initialWorkbenchContext.scope }}|{{ initialWorkbenchContext.import_batch_id }}|{{ initialWorkbenchContext.usage_status }}|{{ initialWorkbenchContext.subscription_tier }}|{{ initialWorkbenchContext.page }}</div>'
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

    expect(getOverview).toHaveBeenLastCalledWith(expect.objectContaining({
      account_id: 7,
      period_type: 'custom',
      start: '2026-06-01',
      end: '2026-06-30'
    }))
    expect(wrapper.findAll('button').filter((node) => node.text().endsWith('account')).map((node) => node.text())).toEqual([
      'Attention account',
      'Healthy account'
    ])

    listSources.mockResolvedValueOnce({
      items: [
        { uploader_user_id: 1, uploader_name: 'uploader-a', account_count: 1, purchase_cost: 10, usage_value: 8, roi_rate: 80, ban_rate_30d: 0, sources: [{ name: 'source-a', account_count: 1, sample_size: 1, purchase_cost: 10, usage_value: 8, roi_rate: 80, ban_rate_7d: 0, ban_rate_30d: 0, ban_rate_90d: 0, refund_rate: 0, average_survival_days: 30, accounts: [] }] },
        { uploader_user_id: 2, uploader_name: 'uploader-b', account_count: 1, purchase_cost: 20, usage_value: 22, roi_rate: 110, ban_rate_30d: 0, sources: [{ name: 'source-a', account_count: 1, sample_size: 1, purchase_cost: 20, usage_value: 22, roi_rate: 110, ban_rate_7d: 0, ban_rate_30d: 0, ban_rate_90d: 0, refund_rate: 0, average_survival_days: 20, accounts: [] }] }
      ]
    })
    await router.push('/?tab=sources&account_id=7&period_type=week&period_start=2026-06-22&period_end=2026-06-29')
    await flushPromises()

    expect(listSources).toHaveBeenLastCalledWith(expect.objectContaining({
      account_id: 7,
      period_type: 'week',
      start: '2026-06-22',
      end: '2026-06-29'
    }))
    expect(wrapper.findAll('h3').filter((node) => node.text() === 'source-a')).toHaveLength(1)

    await router.push('/?tab=accounts&account_axis=source&account_scope=batch&import_batch_id=batch-9&account_usage_status=all&account_page=3')
    await flushPromises()

    expect(wrapper.get('[data-test="accounts-context"]').text()).toBe('source|batch|batch-9|all||3')

    await router.push('/?tab=accounts&account_axis=usage&account_usage_status=auth_issue')
    await flushPromises()
    expect(wrapper.get('[data-test="accounts-context"]').text()).toContain('usage|all||auth_issue')

    await router.push('/?tab=accounts')
    await flushPromises()
    expect(wrapper.get('[data-test="accounts-context"]').text()).toContain('in_use')
  })
})
