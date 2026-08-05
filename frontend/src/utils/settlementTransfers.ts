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
	id?: number
	member_user_id: number
	member_user_name: string
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
	return memberBalances(lines).sort((a, b) => a.cents - b.cents || a.id - b.id)[0]?.id
}

/** Create deterministic pairwise transfers from member net balances. */
export function buildSettlementTransferPreview(
	lines: TransferLine[],
	accountNamesOrLegacyUser: Record<number, string> | number = {},
	legacyAccountNamesOrStatuses: Record<number, string> | Record<string, 'pending' | 'paid'> = {},
	legacyPaymentStatuses: Record<number, 'pending' | 'paid'> = {}
): SettlementTransferPreview[] {
	const legacyUserID = typeof accountNamesOrLegacyUser === 'number' ? accountNamesOrLegacyUser : undefined
	const accountNames = (typeof accountNamesOrLegacyUser === 'number' ? legacyAccountNamesOrStatuses : accountNamesOrLegacyUser) as Record<number, string>
	const paymentStatuses = (typeof accountNamesOrLegacyUser === 'number' ? legacyPaymentStatuses : legacyAccountNamesOrStatuses) as Record<string, 'pending' | 'paid'>
	const members = memberBalances(lines).filter(member => member.cents !== 0)
	if (legacyUserID) {
		const hub = members.find(member => member.id === legacyUserID) || members.find(member => member.id === defaultSettlementUserID(lines))
		if (!hub) return []
		return members.filter(member => member.id !== hub.id).sort((a, b) => Math.abs(b.cents) - Math.abs(a.cents) || a.id - b.id).map(member => {
			const allocations = lines.filter(line => line.user_id === member.id).map((line): SettlementTransferAllocation => {
				const accountID = 'account_id' in line ? line.account_id : undefined
				return { id: 'id' in line ? line.id : undefined, account_id: accountID, account_name: accountID ? accountNames[accountID] || `#${accountID}` : '-', net_amount: line.net_amount }
			})
			const accountIDs = [...new Set(allocations.flatMap(item => item.account_id ? [item.account_id] : []))]
			const fromHub = member.cents < 0
			return { member_user_id: member.id, member_user_name: member.name, from_user_id: fromHub ? hub.id : member.id, from_user_name: fromHub ? hub.name : member.name, to_user_id: fromHub ? member.id : hub.id, to_user_name: fromHub ? member.name : hub.name, amount: Math.abs(member.cents) / 100, payment_status: paymentStatuses[String(member.id)] || 'pending', allocation_ids: allocations.flatMap(item => item.id ? [item.id] : []), account_ids: accountIDs, account_names: accountIDs.map(id => accountNames[id] || `#${id}`), allocations }
		})
	}
	const payables = members.filter(member => member.cents > 0).sort((a, b) => b.cents - a.cents || a.id - b.id)
	const receivables = members.filter(member => member.cents < 0).map(member => ({ ...member, cents: -member.cents })).sort((a, b) => b.cents - a.cents || a.id - b.id)
	const result: SettlementTransferPreview[] = []
	for (let i = 0, j = 0; i < payables.length && j < receivables.length;) {
		const amount = Math.min(payables[i].cents, receivables[j].cents)
		const allocations = lines
			.filter(line => line.user_id === payables[i].id || line.user_id === receivables[j].id)
			.map((line): SettlementTransferAllocation => {
				const accountID = 'account_id' in line ? line.account_id : undefined
				return { id: 'id' in line ? line.id : undefined, account_id: accountID, account_name: accountID ? accountNames[accountID] || `#${accountID}` : '-', net_amount: line.net_amount }
			})
		const accountIDs = [...new Set(allocations.flatMap(item => item.account_id ? [item.account_id] : []))]
		const key = `${payables[i].id}:${receivables[j].id}:${amount}`
		result.push({
			member_user_id: payables[i].id,
			member_user_name: payables[i].name,
			from_user_id: payables[i].id,
			from_user_name: payables[i].name,
			to_user_id: receivables[j].id,
			to_user_name: receivables[j].name,
			amount: amount / 100,
			payment_status: paymentStatuses[key] || 'pending',
			allocation_ids: allocations.flatMap(item => item.id ? [item.id] : []),
			account_ids: accountIDs,
			account_names: accountIDs.map(id => accountNames[id] || `#${id}`),
			allocations
		})
		payables[i].cents -= amount
		receivables[j].cents -= amount
		if (payables[i].cents === 0) i++
		if (receivables[j].cents === 0) j++
	}
	return result
}
