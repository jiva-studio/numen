import { describe, expect, it } from 'vitest'
import {
  formatSize,
  getOfferLabel,
  getPlanLabel,
  getUsageText,
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

describe('getOfferLabel', () => {
  it('adds the price to the name and the size', () => {
    expect(getOfferLabel(PLUS)).toBe('Plus, 100 GB, $15')
  })

  it('has no price part for a plan with no price', () => {
    expect(getOfferLabel({ ...PLUS, priceCents: 0 })).toBe('Plus, 100 GB')
  })

  it('shows whole dollars without cents and cents when there are some', () => {
    expect(getOfferLabel({ ...PLUS, priceCents: 500 })).toBe('Plus, 100 GB, $5')
    expect(getOfferLabel({ ...PLUS, priceCents: 499 })).toBe('Plus, 100 GB, $4.99')
  })

  it('shows the symbol of euro and the code of another currency', () => {
    expect(getOfferLabel({ ...PLUS, priceCents: 1000, currency: 'EUR' })).toBe('Plus, 100 GB, €10')
    expect(getOfferLabel({ ...PLUS, priceCents: 700, currency: 'CHF' })).toBe('Plus, 100 GB, 7 CHF')
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
