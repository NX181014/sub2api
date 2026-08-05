import type {
  SharedPoolSettlementAccountLine,
  SharedPoolSettlementLine
} from '@/api/admin/sharedPool'

export interface SettlementTransferAllocation {
  id?: number
  account_id?: number
  account_name: string
  net_amount: number
}

export interface SettlementTransferPreview {
  member_user_id: number
  member_user_name: string
  settlement_user_id: number
  settlement_user_name: string
  from_user_id: number
  from_user_name: string
  to_user_id: number
  to_user_name: string
  amount: number
  payment_status: 'pending' | 'paid'
  allocation_ids: number[]
  account_ids: number[]
  account_names: string[]
  allocations: SettlementTransferAllocation[]
}

type TransferLine = SharedPoolSettlementLine | SharedPoolSettlementAccountLine

const cents = (value: number) => Math.round(value * 100)

const memberBalances = (lines: TransferLine[]) => {
  const members = new Map<number, { id: number; name: string; cents: number }>()
  for (const line of lines) {
    const member = members.get(line.user_id)
    if (member) member.cents += cents(line.net_amount)
    else members.set(line.user_id, { id: line.user_id, name: line.user_name, cents: cents(line.net_amount) })
  }
  return [...members.values()]
}

export function defaultSettlementUserID(lines: TransferLine[]): number | undefined {
  return memberBalances(lines)
    .sort((a, b) => a.cents - b.cents || a.id - b.id)[0]?.id
}

/** Aggregate every non-hub member into one transfer while retaining account allocations. */
export function buildSettlementTransferPreview(
  lines: TransferLine[],
  settlementUserID?: number,
  accountNames: Record<number, string> = {},
  paymentStatuses: Record<number, 'pending' | 'paid'> = {}
): SettlementTransferPreview[] {
  const members = memberBalances(lines)
  const settlementUser = members.find(member => member.id === settlementUserID)
    || members.find(member => member.id === defaultSettlementUserID(lines))
  if (!settlementUser) return []

  return members
    .filter(member => member.id !== settlementUser.id && member.cents !== 0)
    .sort((a, b) => Math.abs(b.cents) - Math.abs(a.cents) || a.id - b.id)
    .map((member) => {
      const allocations = lines
        .filter(line => line.user_id === member.id)
        .map((line): SettlementTransferAllocation => {
          const accountID = 'account_id' in line ? line.account_id : undefined
          return {
            id: 'id' in line ? line.id : undefined,
            account_id: accountID,
            account_name: accountID ? accountNames[accountID] || `#${accountID}` : '-',
            net_amount: line.net_amount
          }
        })
      const accountIDs = [...new Set(allocations.flatMap(item => item.account_id ? [item.account_id] : []))]
      const fromHub = member.cents < 0
      return {
        member_user_id: member.id,
        member_user_name: member.name,
        settlement_user_id: settlementUser.id,
        settlement_user_name: settlementUser.name,
        from_user_id: fromHub ? settlementUser.id : member.id,
        from_user_name: fromHub ? settlementUser.name : member.name,
        to_user_id: fromHub ? member.id : settlementUser.id,
        to_user_name: fromHub ? member.name : settlementUser.name,
        amount: Math.abs(member.cents) / 100,
        payment_status: paymentStatuses[member.id] || 'pending',
        allocation_ids: allocations.flatMap(item => item.id ? [item.id] : []),
        account_ids: accountIDs,
        account_names: accountIDs.map(id => accountNames[id] || `#${id}`),
        allocations
      }
    })
}
