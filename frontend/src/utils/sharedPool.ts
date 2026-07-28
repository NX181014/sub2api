import { formatPaymentAmount } from '@/components/payment/currency'
import type {
  PoolAccountStatus,
  PoolPeriodParams,
  PoolPeriodType,
  PoolSettlementStatus,
  SharedPoolAccountCost
} from '@/api/admin/sharedPool'

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function resolvePoolPeriod(type: PoolPeriodType, reference = new Date()): { start: string; end: string } {
  const start = new Date(reference.getFullYear(), reference.getMonth(), reference.getDate())

  if (type === 'week') {
    const mondayOffset = (start.getDay() + 6) % 7
    start.setDate(start.getDate() - mondayOffset)
  } else if (type === 'month') {
    start.setDate(1)
  }

  const end = new Date(start)
  if (type === 'month') end.setMonth(end.getMonth() + 1)
  else if (type === 'week') end.setDate(end.getDate() + 7)
  else end.setDate(end.getDate() + 1)

  return { start: formatLocalDate(start), end: formatLocalDate(end) }
}

export function buildPoolPeriodParams(
  periodType: PoolPeriodType,
  start: string,
  end: string
): PoolPeriodParams {
  return { period_type: periodType, start, end }
}

export function formatPoolMoney(amount: number, currency = 'CNY', locale?: string): string {
  return formatPaymentAmount(Number.isFinite(amount) ? amount : 0, currency, locale)
}

export function accountStatusPresentation(status: PoolAccountStatus): { badge: string; key: string } {
  const states: Record<PoolAccountStatus, { badge: string; key: string }> = {
    active: { badge: 'active', key: 'active' },
    warning: { badge: 'warning', key: 'warning' },
    banned: { badge: 'danger', key: 'banned' },
    inactive: { badge: 'inactive', key: 'inactive' }
  }
  return states[status] ?? states.inactive
}

export function settlementStatusPresentation(status: PoolSettlementStatus): { badge: string; key: string } {
  const states: Record<PoolSettlementStatus, { badge: string; key: string }> = {
    draft: { badge: 'warning', key: 'draft' },
    locked: { badge: 'success', key: 'locked' },
    paid: { badge: 'active', key: 'paid' }
  }
  return states[status] ?? states.draft
}

export function latestPoolRecords(records: SharedPoolAccountCost[]): Record<number, SharedPoolAccountCost> {
  return records.reduce<Record<number, SharedPoolAccountCost>>((latest, record) => {
    if (!latest[record.account_id] || latest[record.account_id].id < record.id) latest[record.account_id] = record
    return latest
  }, {})
}
