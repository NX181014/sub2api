import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { Account } from '@/types'
import AccountTracePanel from '../AccountTracePanel.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${JSON.stringify(params)}` : key,
      locale: { value: 'zh-CN' }
    })
  }
})

const account = {
  id: 7,
  name: 'account-7',
  platform: 'openai',
  type: 'oauth',
  status: 'error',
  schedulable: false,
  error_message: 'credential expired',
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  expires_at: null,
  auto_pause_on_expired: false,
  concurrency: 1,
  priority: 0,
  last_used_at: null,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  cost_sharing_enabled: true
} as Account

const settlement = {
  id: 3,
  status: 'draft' as const,
  period_type: 'month' as const,
  period_start: '2026-07-01',
  period_end: '2026-08-01',
  currency: 'CNY',
  total_cost: 10,
  total_usage_weight: 1,
  carry_forward: 0,
  unpriced_usage_count: 0,
  pricing_coverage: 100,
  lines: [],
  account_contexts: [],
  account_costs: [{ id: 1, settlement_id: 3, account_id: 7, cost_entry_id: 11, kind: 'period' as const, payer_user_id: 2, amount: 10 }],
  account_lines: [{ id: 2, settlement_id: 3, account_id: 7, user_id: 2, user_name: 'payer', account_usage_weight: 1, usage_share: 100, allocated_cost: 10, contribution_credit: 0, adjustment: 0, net_amount: 10, trace_quality: 'exact' as const }]
}

function mountPanel(panelSettlement = settlement) {
  return mount(AccountTracePanel, {
    props: {
      show: true,
      loading: false,
      accountId: 7,
      account,
      entries: [],
      entriesPage: 1,
      entriesTotal: 0,
      settlement: panelSettlement,
      settlementPage: 1,
      settlementTotal: 2,
      recovery: null,
      lifecycle: [{
        id: 5,
        account_id: 7,
        account_name: 'account-7',
        event_type: 'recovered',
        occurred_at: '2026-07-20T00:00:00Z',
        transferred_cost_minor: 0,
        source: 'manual',
        created_at: '2026-07-20T00:00:00Z'
      }],
      approvals: [{
        id: 9,
        action_type: 'DELETE_ACCOUNT',
        account_id: 7,
        account_name: 'account-7',
        status: 'pending',
        reason: 'review deletion',
        requested_by_user_id: 1,
        requested_by_email: 'requester@example.com',
        requested_at: '2026-07-21T00:00:00Z'
      }],
      approvalsPage: 1,
      approvalsTotal: 1
    },
    global: {
      stubs: {
        Teleport: true,
        AccountStatusIndicator: {
          props: ['account'],
          template: '<span>runtime-status:{{ account.status }}</span>'
        },
        PlatformTypeBadge: true,
        Icon: true,
        LoadingSpinner: true,
        EmptyState: { props: ['title'], template: '<p>{{ title }}</p>' },
        StatusBadge: { props: ['label'], template: '<span>{{ label }}</span>' },
        DataTable: {
          props: ['data'],
          template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-trace_quality" :row="row" /></div></div>'
        },
        Pagination: {
          template: '<button data-test="next-settlement" @click="$emit(\'update:page\', 2)">next</button>'
        }
      }
    }
  })
}

describe('AccountTracePanel', () => {
  it('keeps runtime and lifecycle state separate and pages settlement history', async () => {
    const wrapper = mountPanel()

    expect(wrapper.text()).toContain('runtime-status:error')
    expect(wrapper.text()).toContain('admin.sharedPool.event.recovered')
    expect(wrapper.text()).toContain('credential expired')
    expect(wrapper.text()).toContain('admin.sharedPool.intake.pendingNotice')

    await wrapper.get('button[role="tab"]:nth-of-type(2)').trigger('click')
    await wrapper.get('[data-test="next-settlement"]').trigger('click')
    expect(wrapper.emitted('settlement-page')).toEqual([[2]])

    await wrapper.get('button[role="tab"]:nth-of-type(5)').trigger('click')
    expect(wrapper.text()).toContain('review deletion')
    expect(wrapper.text()).toContain('admin.sharedPool.approval.deleteAccount')
  })

  it('derives data quality from the current account instead of the settlement header', async () => {
    const wrapper = mountPanel({ ...settlement, unpriced_usage_count: 3 })

    expect(wrapper.text()).not.toContain('admin.sharedPool.settlement.unpricedWarning')

    await wrapper.setProps({
      settlement: {
        ...settlement,
        unpriced_usage_count: 0,
        account_lines: [{ ...settlement.account_lines[0], trace_quality: 'unavailable' as const }]
      }
    })
    expect(wrapper.text()).toContain('admin.sharedPool.settlement.unpricedWarning')
  })

  it('shows effective scheduling after runtime blockers', async () => {
    const wrapper = mountPanel()
    expect(wrapper.text()).toContain('admin.accounts.schedulableDisabled')

    await wrapper.setProps({
      account: {
        ...account,
        status: 'active',
        schedulable: true,
        error_message: null,
        rate_limit_reset_at: new Date(Date.now() + 60_000).toISOString()
      }
    })
    expect(wrapper.text()).toContain('admin.accounts.schedulableDisabled')

    await wrapper.setProps({
      account: {
        ...account,
        status: 'active',
        schedulable: true,
        error_message: null
      }
    })
    expect(wrapper.text()).toContain('admin.accounts.schedulableEnabled')

    await wrapper.setProps({
      account: {
        ...account,
        status: 'active',
        type: 'apikey',
        schedulable: true,
        error_message: null,
        quota_daily_used: 1,
        quota_daily_limit: 1,
        extra: { quota_daily_start: new Date().toISOString() }
      }
    })
    expect(wrapper.text()).toContain('admin.accounts.schedulableDisabled')

    await wrapper.setProps({
      account: {
        ...account,
        status: 'active',
        type: 'apikey',
        schedulable: true,
        error_message: null,
        quota_daily_used: 1,
        quota_daily_limit: 1,
        extra: { quota_daily_start: new Date(Date.now() - 2 * 86_400_000).toISOString() }
      }
    })
    expect(wrapper.text()).toContain('admin.accounts.schedulableEnabled')
  })
})
