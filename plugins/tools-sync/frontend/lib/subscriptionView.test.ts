import { describe, expect, it } from 'vitest'
import {
  formatDate,
  formatSize,
  getPlanLabel,
  getPlanPrice,
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

describe('getPlanLabel', () => {
  it('joins the name and the size of a plan', () => {
    expect(getPlanLabel(PLUS)).toBe('Plus, 100 GB')
  })

  it('shows only the size of a plan with no name', () => {
    expect(getPlanLabel({ ...PLUS, name: '' })).toBe('100 GB')
  })
})

describe('getPlanPrice', () => {
  it('prices a plan for a month', () => {
    expect(getPlanPrice(PLUS)).toBe('$15 / month')
  })

  it('has no price for a plan with no price', () => {
    expect(getPlanPrice({ ...PLUS, priceCents: 0 })).toBe('')
  })

  it('shows whole dollars without cents and cents when there are some', () => {
    expect(getPlanPrice({ ...PLUS, priceCents: 500 })).toBe('$5 / month')
    expect(getPlanPrice({ ...PLUS, priceCents: 499 })).toBe('$4.99 / month')
  })

  it('shows the symbol of euro and the code of another currency', () => {
    expect(getPlanPrice({ ...PLUS, priceCents: 1000, currency: 'EUR' })).toBe('€10 / month')
    expect(getPlanPrice({ ...PLUS, priceCents: 700, currency: 'CHF' })).toBe('7 CHF / month')
  })
})

describe('formatDate', () => {
  it('writes the month in full, then the day and the year', () => {
    expect(formatDate('2026-11-23T12:00:00Z')).toBe('November 23, 2026')
  })

  it('is empty for an empty or an unreadable time', () => {
    expect(formatDate('')).toBe('')
    expect(formatDate('soon')).toBe('')
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
    expect(view.text).toBe('Create an identity to buy a subscription.')
  })
})

describe('getValidity', () => {
  const DATES = { periodEnd: '2026-12-23T12:00:00Z', writesEnd: '', deleteAt: '' }

  it('says until when the paid period lasts', () => {
    expect(getValidity(DATES, false)).toStrictEqual({
      label: 'Active until',
      date: 'December 23, 2026',
    })
  })

  it('says until when the trial lasts', () => {
    expect(getValidity(DATES, true).label).toBe('Trial until')
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
    expect(getUsageText(2 * GIB, 100 * GIB)).toBe('2 GB of 100 GB used')
  })

  it('shows the used space above the quota as it is', () => {
    expect(getUsageText(20 * GIB, 10 * GIB)).toBe('20 GB of 10 GB used')
  })
})
