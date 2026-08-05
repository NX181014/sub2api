import { describe, expect, it } from 'vitest'
import type { SharedPoolSettlementAccountLine } from '@/api/admin/sharedPool'
import { buildSettlementTransferPreview } from '@/utils/settlementTransfers'

const line = (accountID: number, userID: number, netAmount: number): SharedPoolSettlementAccountLine => ({
  id: accountID * 10 + userID,
  settlement_id: 1,
  account_id: accountID,
  user_id: userID,
  user_name: `user-${userID}`,
  account_usage_weight: 0,
  usage_share: 0,
  allocated_cost: 0,
  contribution_credit: 0,
  adjustment: 0,
  net_amount: netAmount,
  trace_quality: 'exact'
})

describe('settlement transfer preview', () => {
  it('pairs balances inside each account without cross-account netting', () => {
    const transfers = buildSettlementTransferPreview([
      line(1, 1, 10),
      line(1, 2, -10),
      line(2, 1, -8),
      line(2, 3, 8)
    ])

    expect(transfers.map(({ account_id, from_user_id, to_user_id, amount }) => ({
      account_id,
      from_user_id,
      to_user_id,
      amount
    }))).toEqual([
      { account_id: 1, from_user_id: 1, to_user_id: 2, amount: 10 },
      { account_id: 2, from_user_id: 3, to_user_id: 1, amount: 8 }
    ])
  })
})
