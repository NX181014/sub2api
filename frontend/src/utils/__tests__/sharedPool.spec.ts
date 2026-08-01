import { describe, expect, it } from 'vitest'
import {
  accountStatusPresentation,
  buildPoolPeriodParams,
  formatPoolMoney,
  formatPoolPaidAtInput,
  isPoolPaidAtFuture,
  latestPoolRecords,
  poolPaidAtToISOString,
  resolvePoolPeriod,
  settlementStatusPresentation
} from '../sharedPool'

describe('shared pool presentation helpers', () => {
  it('formats CNY amounts consistently', () => {
    const formatted = formatPoolMoney(1234.5, 'CNY', 'zh-CN')
    expect(formatted).toContain('1,234.50')
  })

  it('maps account and settlement statuses to visible states', () => {
    expect(accountStatusPresentation('banned')).toEqual({ badge: 'danger', key: 'banned' })
    expect(settlementStatusPresentation('locked')).toEqual({ badge: 'success', key: 'locked' })
  })

  it('builds week and custom period parameters', () => {
    expect(resolvePoolPeriod('week', new Date(2026, 6, 29))).toEqual({
      start: '2026-07-27',
      end: '2026-08-03'
    })
    expect(resolvePoolPeriod('day', new Date(2026, 6, 29))).toEqual({
      start: '2026-07-29',
      end: '2026-07-30'
    })
    expect(resolvePoolPeriod('month', new Date(2026, 6, 29))).toEqual({
      start: '2026-07-01',
      end: '2026-08-01'
    })
    expect(buildPoolPeriodParams('custom', '2026-07-01', '2026-07-18')).toEqual({
      period_type: 'custom',
      start: '2026-07-01',
      end: '2026-07-18'
    })
  })

  it('keeps the newest pool record for each account', () => {
    const base = {
      account_id: 7,
      account_name: 'account-7',
      provider_identity: 'identity',
      contributor_name: 'contributor',
      uploader_name: 'uploader',
      purchase_source_name: 'source',
      purchase_cost: 20,
      currency: 'CNY',
      service_start: '2026-07-01',
      service_end: '2026-08-01',
      status: 'active' as const,
      usage_value: 10,
      roi_rate: 50,
      remaining_cost: 10,
      banned_loss: 0
    }
    const records = latestPoolRecords([{ ...base, id: 2 }, { ...base, id: 9, purchase_cost: 30 }])

    expect(records[7]).toMatchObject({ id: 9, purchase_cost: 30 })
  })

  it('keeps exact purchase times and rejects clearly future values', () => {
    const value = '2026-08-01T08:11'
    const iso = poolPaidAtToISOString(value)

    expect(iso).toBe(new Date(value).toISOString())
    expect(formatPoolPaidAtInput(iso)).toBe(value)
    expect(poolPaidAtToISOString('')).toBeUndefined()
    expect(isPoolPaidAtFuture('2026-08-01T08:16', new Date('2026-08-01T08:10:59').getTime())).toBe(true)
    expect(isPoolPaidAtFuture('2026-08-01T08:15', new Date('2026-08-01T08:10:59').getTime())).toBe(false)
  })
})
