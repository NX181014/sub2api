import type { SharedPoolBatchAmountMode } from '@/api/admin/sharedPool'

export type BatchAllocationMode = 'equal' | 'manual'

export interface BatchCostOverride {
  originalAmount?: number | '' | null
  expectedTokenCount?: number | '' | null
}

export interface BatchCostAllocation {
  accountId: number
  originalAmount: number
  expectedTokenCount: number
}

export type BatchCostAllocationError =
  | 'accounts_required'
  | 'duplicate_accounts'
  | 'amount_invalid'
  | 'allocation_exceeds_total'
  | 'allocation_total_mismatch'
  | 'expected_tokens_invalid'

export interface BatchCostAllocationResult {
  allocations: BatchCostAllocation[]
  totalAmount: number
  errors: BatchCostAllocationError[]
}

const toCents = (value: number): number => Math.round(value * 100)
const hasValue = (value: number | '' | null | undefined): value is number => value !== '' && value != null

export function calculateBatchCostAllocations(input: {
  accountIds: number[]
  amountMode: SharedPoolBatchAmountMode
  allocationMode: BatchAllocationMode
  commonAmount: number
  commonExpectedTokenCount: number
  overrides: Record<number, BatchCostOverride | undefined>
}): BatchCostAllocationResult {
  const errors: BatchCostAllocationError[] = []
  const accountIds = input.accountIds.filter((id) => Number.isInteger(id) && id > 0)
  if (!accountIds.length) errors.push('accounts_required')
  if (new Set(accountIds).size !== accountIds.length) errors.push('duplicate_accounts')

  const commonCents = toCents(input.commonAmount)
  if (!Number.isFinite(input.commonAmount) || commonCents <= 0) errors.push('amount_invalid')

  const amountByAccount = new Map<number, number>()
  if (input.amountMode === 'per_account') {
    for (const id of accountIds) {
      const override = input.overrides[id]?.originalAmount
      const cents = hasValue(override) ? toCents(override) : commonCents
      if (!Number.isFinite(cents) || cents <= 0) errors.push('amount_invalid')
      amountByAccount.set(id, cents)
    }
  } else if (input.allocationMode === 'manual') {
    let allocated = 0
    for (const id of accountIds) {
      const amount = input.overrides[id]?.originalAmount
      const cents = hasValue(amount) ? toCents(amount) : 0
      if (!Number.isFinite(cents) || cents <= 0) errors.push('amount_invalid')
      amountByAccount.set(id, cents)
      allocated += cents
    }
    if (allocated !== commonCents) errors.push('allocation_total_mismatch')
  } else {
    const fixed = accountIds.filter((id) => hasValue(input.overrides[id]?.originalAmount))
    const flexible = accountIds.filter((id) => !hasValue(input.overrides[id]?.originalAmount))
    let fixedCents = 0
    for (const id of fixed) {
      const cents = toCents(input.overrides[id]!.originalAmount as number)
      if (!Number.isFinite(cents) || cents <= 0) errors.push('amount_invalid')
      amountByAccount.set(id, cents)
      fixedCents += cents
    }
    const remaining = commonCents - fixedCents
    if (remaining < 0 || (!flexible.length && remaining !== 0)) errors.push('allocation_exceeds_total')
    if (flexible.length && remaining > 0) {
      const base = Math.floor(remaining / flexible.length)
      const remainder = remaining % flexible.length
      flexible.forEach((id, index) => amountByAccount.set(id, base + (index < remainder ? 1 : 0)))
    }
    if (flexible.length && remaining <= 0) errors.push('amount_invalid')
  }

  const allocations = accountIds.map((accountId) => {
    const cents = amountByAccount.get(accountId) || 0
    if (cents <= 0) errors.push('amount_invalid')
    const expectedOverride = input.overrides[accountId]?.expectedTokenCount
    const expectedTokenCount = hasValue(expectedOverride) ? expectedOverride : input.commonExpectedTokenCount
    if (!Number.isSafeInteger(expectedTokenCount) || expectedTokenCount <= 0) errors.push('expected_tokens_invalid')
    return {
      accountId,
      originalAmount: cents / 100,
      expectedTokenCount
    }
  })

  return {
    allocations,
    totalAmount: allocations.reduce((sum, item) => sum + item.originalAmount, 0),
    errors: [...new Set(errors)]
  }
}
