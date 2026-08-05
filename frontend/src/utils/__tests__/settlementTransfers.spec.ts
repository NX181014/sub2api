import { describe, expect, it } from 'vitest'
import type { SharedPoolSettlementAccountLine } from '@/api/admin/sharedPool'
import {
  buildSettlementTransferPreview,
  defaultSettlementUserID
} from '@/utils/settlementTransfers'

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

const lines = [
  line(1, 1, 10),
  line(1, 2, -10),
  line(2, 1, -8),
  line(2, 3, 8)
]

describe('settlement transfer preview', () => {
  it('nets each member to one designated settlement user and keeps account allocations', () => {
    expect(defaultSettlementUserID(lines)).toBe(2)

    const transfers = buildSettlementTransferPreview(lines, 2, { 1: 'one', 2: 'two' })

    expect(transfers.map(({ member_user_id, from_user_id, to_user_id, amount, account_ids }) => ({
      member_user_id,
      from_user_id,
      to_user_id,
      amount,
      account_ids
    }))).toEqual([
      { member_user_id: 3, from_user_id: 3, to_user_id: 2, amount: 8, account_ids: [2] },
      { member_user_id: 1, from_user_id: 1, to_user_id: 2, amount: 2, account_ids: [1, 2] }
    ])
    expect(transfers[1].allocation_ids).toEqual([11, 21])
    expect(transfers[1].account_names).toEqual(['one', 'two'])
  })

  it('reverses the transfer when the designated user owes a receiver', () => {
    const transfers = buildSettlementTransferPreview(lines, 1)
    const receiver = transfers.find(item => item.member_user_id === 2)

    expect(receiver).toMatchObject({
      from_user_id: 1,
      to_user_id: 2,
      amount: 10
    })
  })
})
