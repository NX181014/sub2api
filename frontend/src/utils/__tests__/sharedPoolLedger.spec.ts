import { describe, expect, it } from 'vitest'
import {
  calculateBatchCostAllocations,
  DEFAULT_EXPECTED_TOKEN_COUNT,
  filterRecoveryAccounts,
  filterSettlementLines,
  millionsToTokens,
  recoveryState,
  tokensToMillions
} from '@/utils/sharedPoolLedger'
import type { SharedPoolAccountCost, SharedPoolSettlementLine } from '@/api/admin/sharedPool'

describe('shared-pool expected token input', () => {
  it('uses millions in the UI while preserving raw token counts', () => {
    expect(tokensToMillions(DEFAULT_EXPECTED_TOKEN_COUNT)).toBe(20)
    expect(millionsToTokens(20)).toBe(20_000_000)
    expect(millionsToTokens(1.5)).toBe(1_500_000)
  })
})

describe('shared-pool batch cost allocation', () => {
  it('treats the default amount as a per-account price', () => {
    const result = calculateBatchCostAllocations({
      accountIds: [1, 2, 3],
      amountMode: 'per_account',
      allocationMode: 'equal',
      commonAmount: 10,
      commonExpectedTokenCount: 1_000_000,
      overrides: {}
    })

    expect(result.errors).toEqual([])
    expect(result.totalAmount).toBe(30)
    expect(result.allocations.map((item) => item.originalAmount)).toEqual([10, 10, 10])
  })

  it('allocates order-total cents deterministically after fixed overrides', () => {
    const result = calculateBatchCostAllocations({
      accountIds: [1, 2, 3],
      amountMode: 'order_total',
      allocationMode: 'equal',
      commonAmount: 10,
      commonExpectedTokenCount: 1_000_000,
      overrides: { 1: { originalAmount: 4 } }
    })

    expect(result.errors).toEqual([])
    expect(result.allocations.map((item) => item.originalAmount)).toEqual([4, 3, 3])
    expect(result.totalAmount).toBe(10)
  })

  it('rejects mismatched manual totals and missing expected tokens', () => {
    const result = calculateBatchCostAllocations({
      accountIds: [1, 2],
      amountMode: 'order_total',
      allocationMode: 'manual',
      commonAmount: 10,
      commonExpectedTokenCount: 0,
      overrides: { 1: { originalAmount: 4 }, 2: { originalAmount: 5 } }
    })

    expect(result.errors).toContain('allocation_total_mismatch')
    expect(result.errors).toContain('expected_tokens_invalid')
  })

  it('rejects an order total that would allocate zero cents to an account', () => {
    const result = calculateBatchCostAllocations({
      accountIds: [1, 2],
      amountMode: 'order_total',
      allocationMode: 'equal',
      commonAmount: 0.01,
      commonExpectedTokenCount: 1_000_000,
      overrides: {}
    })

    expect(result.allocations.map((item) => item.originalAmount)).toEqual([0.01, 0])
    expect(result.errors).toContain('amount_invalid')
  })
})

describe('shared-pool page filters', () => {
  const account = (values: Partial<SharedPoolAccountCost>): SharedPoolAccountCost => ({
    id: 1,
    account_id: 1,
    account_name: 'fixture',
    provider_identity: '',
    contributor_name: '',
    uploader_name: '',
    purchase_source_name: '',
    purchase_cost: 100,
    currency: 'CNY',
    service_start: '',
    service_end: '',
    status: 'active',
    usage_value: 50,
    roi_rate: 50,
    remaining_cost: 50,
    banned_loss: 0,
    ...values
  })

  it('classifies recovered, near-payback, unrecovered, and missing-data accounts', () => {
    const accounts = [
      account({ id: 1, currently_recovered: true }),
      account({ id: 2, account_id: 2, roi_rate: 85 }),
      account({ id: 3, account_id: 3 }),
      account({ id: 4, account_id: 4, usage_value: 0, roi_rate: 0, estimated_recovery_days: null })
    ]

    expect(accounts.map(recoveryState)).toEqual(['recovered', 'soon', 'unrecovered', 'no_data'])
    expect(filterRecoveryAccounts(accounts, 'soon').map((item) => item.id)).toEqual([2])
  })

  it('filters settlement lines by payment state and flags invalid numeric data', () => {
    const line = (values: Partial<SharedPoolSettlementLine>): SharedPoolSettlementLine => ({
      user_id: 1,
      user_name: 'fixture',
      usage_weight: 1,
      usage_share: 50,
      allocated_cost: 10,
      contribution_credit: 0,
      adjustment: 0,
      net_amount: 10,
      confirmation_status: 'pending',
      payment_status: 'pending',
      ...values
    })
    const lines = [line({}), line({ user_id: 2, payment_status: 'paid' }), line({ user_id: 3, net_amount: Number.NaN })]

    expect(filterSettlementLines(lines, 'paid').map((item) => item.user_id)).toEqual([2])
    expect(filterSettlementLines(lines, 'abnormal').map((item) => item.user_id)).toEqual([3])
  })
})
