import { describe, expect, it } from 'vitest'
import {
  calculateBatchCostAllocations,
  DEFAULT_EXPECTED_TOKEN_COUNT,
  millionsToTokens,
  tokensToMillions
} from '@/utils/sharedPoolLedger'

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
