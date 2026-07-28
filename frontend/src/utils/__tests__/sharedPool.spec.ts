import { describe, expect, it } from 'vitest'
import {
  accountStatusPresentation,
  buildPoolPeriodParams,
  formatPoolMoney,
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
})
