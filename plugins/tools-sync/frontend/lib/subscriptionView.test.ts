import { describe, expect, it } from 'vitest'
import {
  formatPrice,
  formatSize,
  getPlanLine,
  getSubscriptionView,
  getUsagePercent,
  getUsageText,
  getValidity,
  type SubscriptionPhase,
} from './subscriptionView'

const GIB = 2 ** 30
const PLUS = { id: 'plus', name: 'Plus', quotaBytes: 100 * GIB, priceCents: 1500, currency: 'USD' }

describe('formatSize', () => {
  it('shows whole gibibytes as GB', () => {
    expect(formatSize(10 * GIB)).toBe('10 GB')
    expect(formatSize(250 * GIB)).toBe('250 GB')
  })

  it('shows one decimal for a part of a GB', () => {
    expect(formatSize(1.5 * GIB)).toBe('1.5 GB')
  })

  it('shows 0 GB for no bytes', () => {
    expect(formatSize(0)).toBe('0 GB')
  })
})

describe('getPlanLine', () => {
  it('joins the size and the price for a month of a plan', () => {
    expect(getPlanLine(PLUS, false)).toBe('100 GB, $15 / month')
  })

  it('puts the name in front where the badge does not name the plan', () => {
    expect(getPlanLine(PLUS, true)).toBe('Plus, 100 GB, $15 / month')
  })

  it('leaves out an empty name and an empty price', () => {
    expect(getPlanLine({ ...PLUS, name: '' }, true)).toBe('100 GB, $15 / month')
    expect(getPlanLine({ ...PLUS, priceCents: 0 }, false)).toBe('100 GB')
  })
})

describe('formatPrice', () => {
  it('has no price for no cents', () => {
    expect(formatPrice(0, 'USD')).toBe('')
  })

  it('shows whole dollars without cents and cents when there are some', () => {
    expect(formatPrice(500, 'USD')).toBe('$5')
    expect(formatPrice(499, 'USD')).toBe('$4.99')
  })

  it('shows the symbol of euro and the code of another currency', () => {
    expect(formatPrice(1000, 'EUR')).toBe('€10')
    expect(formatPrice(700, 'CHF')).toBe('7 CHF')
  })
})

describe('getSubscriptionView', () => {
  const DATES = { periodEnd: '2026-12-23T12:00:00Z', writesEnd: '', deleteAt: '' }
  const ready = (phase: SubscriptionPhase, planName = '') =>
    getSubscriptionView({ status: 'ready', phase, dates: DATES, planName })

  it('has no badge when there is no subscription, because the plans say it', () => {
    expect(ready('none').badge).toBe('')
  })

  it('badges an active subscription with its plan', () => {
    expect(ready('active', 'Plus').badge).toBe('Plus')
  })

  it('badges an ended subscription with how it ended', () => {
    expect(ready('grace').badge).toBe('Ended')
    expect(ready('readOnly').badge).toBe('Read only')
    expect(ready('deleted').badge).toBe('Deleted')
  })

  it('has no badge for an account without identity, because the text says it', () => {
    const view = getSubscriptionView({ status: 'noIdentity', phase: '', dates: DATES })

    expect(view.badge).toBe('')
    expect(view.text).toBe(
      'Numen Sync needs an identity to tell your devices apart. Create one in the Identity card above.',
    )
  })

  it('badges an unreachable server as offline and names it in the text', () => {
    const view = getSubscriptionView({
      status: 'unreachable',
      phase: '',
      dates: DATES,
      server: 'http://127.0.0.1:8787',
    })

    expect(view).toStrictEqual({
      badge: 'Offline',
      text: 'Could not connect to http://127.0.0.1:8787. Start the server, or change the address in the settings.',
      tone: 'warning',
    })
  })
})

describe('getValidity', () => {
  const DATES = { periodEnd: '2026-12-23T12:00:00Z', writesEnd: '', deleteAt: '' }

  it('says when the paid period renews', () => {
    expect(getValidity(DATES, false)).toStrictEqual({
      label: 'Renews',
      date: 'December 23, 2026',
    })
  })

  it('says until when the trial lasts', () => {
    expect(getValidity(DATES, true).label).toBe('Trial until')
  })

  it('writes the month in full, then the day and the year', () => {
    expect(getValidity({ ...DATES, periodEnd: '2026-11-23T12:00:00Z' }, false).date).toBe(
      'November 23, 2026',
    )
  })

  it('has no date for an empty or an unreadable end of the period', () => {
    expect(getValidity({ ...DATES, periodEnd: '' }, false).date).toBe('')
    expect(getValidity({ ...DATES, periodEnd: 'soon' }, false).date).toBe('')
  })
})

describe('getUsagePercent', () => {
  it('is the share of the quota in whole percent', () => {
    expect(getUsagePercent(25 * GIB, 100 * GIB)).toBe(25)
  })

  it('stops at 100 when the space used is above the quota and is 0 without a quota', () => {
    expect(getUsagePercent(20 * GIB, 10 * GIB)).toBe(100)
    expect(getUsagePercent(GIB, 0)).toBe(0)
  })
})

describe('getUsageText', () => {
  it('says how much of the plan is used', () => {
    expect(getUsageText(2 * GIB, 100 * GIB)).toBe('2 GB of 100 GB')
  })

  it('shows the used space above the quota as it is', () => {
    expect(getUsageText(20 * GIB, 10 * GIB)).toBe('20 GB of 10 GB')
  })
})
