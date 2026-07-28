import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { get, post } }))

import { createAccountIntake, getOverview } from '@/api/admin/sharedPool'

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
      currency: 'CNY', fx_rate: '1', cny_amount_minor: 2000,
      service_start: '2026-07-01', service_end: '2026-08-01'
    }
    post.mockRejectedValue(new Error('network timeout'))

    await expect(createAccountIntake(42, payload)).rejects.toThrow('network timeout')
    await expect(createAccountIntake(42, { ...payload, original_amount: '21' })).rejects.toThrow('network timeout')

    expect(post.mock.calls[0][2].headers['Idempotency-Key']).not.toBe(post.mock.calls[1][2].headers['Idempotency-Key'])
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
})
