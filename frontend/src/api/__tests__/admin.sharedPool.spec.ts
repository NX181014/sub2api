import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { get, post } }))

import {
  approveApproval,
  createBatchCosts,
  createAccountIntake,
  createApproval,
  createCost,
  confirmSettlement,
  getOverview,
  listAccountCosts,
  listCostSummaries,
  listLedgerEntries,
  markSettlementPaid,
  previewSettlement,
  listApprovals,
  revealApproval
} from '@/api/admin/sharedPool'

describe('admin shared-pool API', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    sessionStorage.clear()
    get.mockReset()
    post.mockReset()
  })

  it('reuses the intake idempotency key after an ambiguous failure', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue('11111111-1111-4111-8111-111111111111')
    const payload = {
      provider_identity: 'provider@example.com',
      contributor_user_id: 7,
      uploader_user_id: 8,
      purchase_source_name: 'Source',
      entry_type: 'purchase' as const,
      original_amount: '20',
      expected_token_count: 1_000_000,
      currency: 'CNY',
      fx_rate: '1',
      cny_amount_minor: 2000,
      service_start: '2026-07-01',
      service_end: '2026-08-01',
    }

    post.mockRejectedValueOnce(new Error('network timeout'))
    await expect(createAccountIntake(42, payload)).rejects.toThrow('network timeout')
    post.mockResolvedValueOnce({ data: {} })
    await createAccountIntake(42, payload)

    expect(post.mock.calls[0][2]).toEqual(post.mock.calls[1][2])
    expect(post.mock.calls[1][2].headers['Idempotency-Key']).toBe(
      'pool-intake-42-11111111-1111-4111-8111-111111111111'
    )
  })

  it('starts a new intake operation when the form payload changes', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID')
      .mockReturnValueOnce('11111111-1111-4111-8111-111111111111')
      .mockReturnValueOnce('22222222-2222-4222-8222-222222222222')
    const payload = {
      provider_identity: 'provider@example.com', contributor_user_id: 7, uploader_user_id: 8,
      purchase_source_name: 'Source', entry_type: 'purchase' as const, original_amount: '20',
      expected_token_count: 1_000_000,
      currency: 'CNY', fx_rate: '1', cny_amount_minor: 2000,
      service_start: '2026-07-01', service_end: '2026-08-01'
    }
    post.mockRejectedValue(new Error('network timeout'))

    await expect(createAccountIntake(42, payload)).rejects.toThrow('network timeout')
    await expect(createAccountIntake(42, { ...payload, original_amount: '21' })).rejects.toThrow('network timeout')

    expect(post.mock.calls[0][2].headers['Idempotency-Key']).not.toBe(post.mock.calls[1][2].headers['Idempotency-Key'])
  })

  it('appends an additional account cost through the cost endpoint', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue('33333333-3333-4333-8333-333333333333')
    const payload = {
      account_id: 42,
      payer_user_id: 7,
      purchase_source_id: 3,
      entry_type: 'renewal' as const,
      original_amount: '20.00',
      expected_token_count: 1_000_000,
      currency: 'CNY',
      fx_rate: '1',
      service_start: '2026-08-01',
      service_end: '2026-09-01'
    }

    post.mockRejectedValueOnce(new Error('network timeout'))
    await expect(createCost(payload)).rejects.toThrow('network timeout')
    post.mockResolvedValueOnce({ data: {} })
    await createCost(payload)

    expect(post).toHaveBeenNthCalledWith(2, '/admin/pool/costs', payload, {
      headers: { 'Idempotency-Key': 'pool-cost-42-33333333-3333-4333-8333-333333333333' }
    })
    expect(post.mock.calls[0][2]).toEqual(post.mock.calls[1][2])
  })

  it('submits a batch cost with one retry-safe operation key', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue('44444444-4444-4444-8444-444444444444')
    const payload = {
      amount_mode: 'per_account' as const,
      common: {
        payer_user_id: 7,
        entry_type: 'purchase' as const,
        original_amount: '10.00',
        currency: 'CNY',
        service_start: '2026-07-01',
        service_end: '2026-08-01',
        expected_token_count: 1000000
      },
      accounts: [
        { account_id: 41, original_amount: '10.00', expected_token_count: 1000000 },
        { account_id: 42, original_amount: '10.00', expected_token_count: 1000000 }
      ]
    }
    post.mockRejectedValueOnce(new Error('network timeout'))
    await expect(createBatchCosts(payload)).rejects.toThrow('network timeout')
    post.mockResolvedValueOnce({ data: { amount_mode: 'per_account', account_count: 2, total_original_amount: '20.00', total_cny_amount_minor: 2000, entries: [] } })

    await expect(createBatchCosts(payload)).resolves.toMatchObject({ account_count: 2 })
    expect(post.mock.calls[0][2]).toEqual(post.mock.calls[1][2])
    expect(post.mock.calls[1][2].headers['Idempotency-Key']).toBe('pool-cost-batch-44444444-4444-4444-8444-444444444444')
  })

  it('passes ledger filters to paginated server endpoints', async () => {
    const page = { items: [], total: 0, page: 2, page_size: 50, pages: 1 }
    get.mockResolvedValueOnce({ data: page }).mockResolvedValueOnce({ data: page })

    await expect(listCostSummaries({ page: 2, page_size: 50, uploader_user_id: 8, has_cost: true })).resolves.toEqual(page)
    expect(get).toHaveBeenNthCalledWith(1, '/admin/pool/cost-summaries', {
      params: { page: 2, page_size: 50, uploader_user_id: 8, has_cost: true }
    })

    await expect(listLedgerEntries({ page: 2, page_size: 50, search: 'order-1', entry_type: 'renewal' })).resolves.toEqual(page)
    expect(get).toHaveBeenNthCalledWith(2, '/admin/pool/cost-entries', {
      params: { page: 2, page_size: 50, search: 'order-1', entry_type: 'renewal' }
    })
  })

  it('confirms only the current member endpoint and maps confirmation state', async () => {
    const settlement = {
      id: 9, period_type: 'month', period_start: '2026-07-01T00:00:00Z', period_end: '2026-08-01T00:00:00Z',
      status: 'locked', total_cost_minor: 1200, carry_out_minor: 0, total_usage_weight: '1',
      pricing_coverage: '1', unpriced_usage_count: 0, fx_rate: '1',
      lines: [{
        user_id: 23, user_email: 'member@example.com', username: 'member', usage_weight: '1', usage_share: '1',
        allocated_cost_minor: 1200, contribution_credit_minor: 0, adjustment_minor: 0, net_amount_minor: 1200,
        payment_status: 'unpaid', confirmation_status: 'confirmed', confirmed_by_user_id: 23, confirmed_at: '2026-07-28T00:00:00Z'
      }]
    }
    post.mockResolvedValueOnce({ data: settlement }).mockResolvedValueOnce({ data: { ...settlement, status: 'paid' } })

    await expect(confirmSettlement(9)).resolves.toMatchObject({
      status: 'locked', lines: [{ user_id: 23, confirmation_status: 'confirmed', confirmed_by_user_id: 23 }]
    })
    expect(post).toHaveBeenNthCalledWith(1, '/admin/pool/settlements/9/confirm')
    await expect(markSettlementPaid(9)).resolves.toMatchObject({ status: 'paid' })
    expect(post).toHaveBeenNthCalledWith(2, '/admin/pool/settlements/9/paid')
  })

  it('sends the selected settlement scope to the draft endpoint', async () => {
    post.mockResolvedValueOnce({ data: {
      id: 7, period_type: 'custom', period_start: '2026-07-01T00:00:00Z', period_end: '2026-07-08T00:00:00Z',
      status: 'draft', total_cost_minor: 0, carry_out_minor: 0, total_usage_weight: '0', pricing_coverage: '1',
      unpriced_usage_count: 0, fx_rate: '1', lines: []
    } })

    await previewSettlement({
      start: '2026-07-01', end: '2026-07-07', period_type: 'custom',
      account_id: 9, uploader_user_id: 8, payer_user_id: 7, purchase_source_id: 6
    })

    expect(post).toHaveBeenCalledWith('/admin/pool/settlements/draft', {
      period_type: 'custom', start_date: '2026-07-01', end_date: '2026-07-07',
      account_id: 9, uploader_user_id: 8, payer_user_id: 7, purchase_source_id: 6
    })
  })

  it('keeps exact per-account payback fields', async () => {
    get.mockResolvedValueOnce({
      data: {
        start_at: '2026-07-01T00:00:00Z',
        end_at: '2026-08-01T00:00:00Z',
        total_cost_minor: 10000,
        total_value_minor: 12500,
        unrecovered_minor: 0,
        banned_loss_minor: 0,
        recovery_rate: '1.25',
        recovered_accounts: 1,
        total_accounts: 1,
        accounts: [{
          account_id: 9,
          account_name: 'account-9',
          lifecycle_status: 'active',
          net_cost_minor: 10000,
          value_minor: 12500,
          unrecovered_minor: 0,
          banned_loss_minor: 0,
          current_net_loss_minor: 0,
          net_profit_minor: 2500,
          recovery_rate: '1.25',
          first_recovery_at: '2026-07-12T03:04:05Z',
          latest_recovery_at: '2026-07-12T03:04:05Z',
          currently_recovered: true,
          observation_days: 12
        }]
      }
    })

    const result = await getOverview({ start: '2026-07-01', end: '2026-08-01' })

    expect(result.accounts[0]).toMatchObject({
      account_id: 9,
      recovered_at: '2026-07-12T03:04:05Z',
      currently_recovered: true,
      net_profit: 25,
      current_net_loss: 0,
      observation_days: 12
    })
  })

  it('keeps saved pool profile fields for dialog prefill', async () => {
    get
      .mockResolvedValueOnce({ data: [{
        id: 3, account_id: 9, account_name: 'account-9', payer_user_id: 7,
        payer_email: 'payer@example.com', purchase_source: 'source-a', entry_type: 'purchase',
        currency: 'CNY', original_amount: '20', cny_amount_minor: 2000, fx_rate: '1',
        service_start: '2026-07-01T00:00:00Z', service_end: '2026-08-01T00:00:00Z',
        note: 'invoice stored'
      }] })
      .mockResolvedValueOnce({ data: [{
        id: 9, name: 'account-9', provider_identity: 'provider@example.com',
        contributor_user_id: 7, created_by_user_id: 8
      }] })

    const result = await listAccountCosts()

    expect(result.items[0]).toMatchObject({
      account_id: 9,
      provider_identity: 'provider@example.com',
      notes: 'invoice stored'
    })
  })

  it('uses the shared approval contract for update, review, and one-time reveal', async () => {
    const approval = {
      id: 19,
      action_type: 'UPDATE_ACCOUNT' as const,
      account_id: 42,
      account_name: 'account-42',
      status: 'pending' as const,
      reason: 'rename',
      requested_by_user_id: 2,
      requested_by_email: 'admin@example.com',
      requested_at: '2026-07-28T00:00:00Z'
    }
    post.mockResolvedValueOnce({ data: approval })

    await expect(createApproval({
      action_type: 'UPDATE_ACCOUNT',
      account_id: 42,
      reason: 'rename',
      payload: { account_update: { name: 'renamed' } }
    })).resolves.toEqual(approval)
    expect(post).toHaveBeenNthCalledWith(1, '/admin/pool/approvals', {
      action_type: 'UPDATE_ACCOUNT',
      account_id: 42,
      reason: 'rename',
      payload: { account_update: { name: 'renamed' } }
    })

    get.mockResolvedValueOnce({ data: { items: [approval], total: 1, page: 1, page_size: 20, pages: 1 } })
    await listApprovals({ status: 'pending', page: 1, page_size: 20 })
    expect(get).toHaveBeenCalledWith('/admin/pool/approvals', {
      params: { status: 'pending', page: 1, page_size: 20 }
    })

    post.mockResolvedValueOnce({ data: { ...approval, status: 'approved' } })
    await approveApproval(19, 'checked')
    expect(post).toHaveBeenNthCalledWith(2, '/admin/pool/approvals/19/approve', { reason: 'checked' })

    const reveal = { account_id: 42, credentials: { token: 'secret' }, revealed_at: '2026-07-28T00:01:00Z' }
    post.mockResolvedValueOnce({ data: reveal })
    await expect(revealApproval(19)).resolves.toEqual(reveal)
    expect(post).toHaveBeenNthCalledWith(3, '/admin/pool/approvals/19/reveal')
  })
})
