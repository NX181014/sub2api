import type {
  SharedPoolSettlementAccountLine,
  SharedPoolSettlementLine
} from '@/api/admin/sharedPool'

export interface SettlementTransferPreview {
  account_id?: number
  from_user_id: number
  from_user_name: string
  to_user_id: number
  to_user_name: string
  amount: number
}

type TransferLine = SharedPoolSettlementLine | SharedPoolSettlementAccountLine

/**
 * Convert member net balances into deterministic one-to-one transfers.
 * Balances are already rounded by the server, so the preview preserves cents.
 */
export function buildSettlementTransferPreview(lines: TransferLine[]): SettlementTransferPreview[] {
  const result: SettlementTransferPreview[] = []
  const groups = new Map<number, TransferLine[]>()
  for (const line of lines) {
    const accountID = 'account_id' in line ? line.account_id : 0
    const group = groups.get(accountID)
    if (group) group.push(line)
    else groups.set(accountID, [line])
  }

  for (const [accountID, accountLines] of groups) {
    const debtors = accountLines
      .filter(line => line.net_amount > 0)
      .map(line => ({ id: line.user_id, name: line.user_name, remaining: Math.round(line.net_amount * 100) }))
      .sort((a, b) => b.remaining - a.remaining || a.id - b.id)
    const creditors = accountLines
      .filter(line => line.net_amount < 0)
      .map(line => ({ id: line.user_id, name: line.user_name, remaining: Math.round(Math.abs(line.net_amount) * 100) }))
      .sort((a, b) => b.remaining - a.remaining || a.id - b.id)

    let debtorIndex = 0
    let creditorIndex = 0
    while (debtorIndex < debtors.length && creditorIndex < creditors.length) {
      const debtor = debtors[debtorIndex]
      const creditor = creditors[creditorIndex]
      const cents = Math.min(debtor.remaining, creditor.remaining)
      if (cents > 0) {
        result.push({
          account_id: accountID || undefined,
          from_user_id: debtor.id,
          from_user_name: debtor.name,
          to_user_id: creditor.id,
          to_user_name: creditor.name,
          amount: cents / 100
        })
      }
      debtor.remaining -= cents
      creditor.remaining -= cents
      if (debtor.remaining === 0) debtorIndex += 1
      if (creditor.remaining === 0) creditorIndex += 1
    }
  }
  return result
}
