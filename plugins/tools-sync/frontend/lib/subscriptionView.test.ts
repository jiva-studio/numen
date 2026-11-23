import { describe, expect, it } from 'vitest'
import {
  formatPrice,
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

describe('formatPrice', () => {
  it('shows whole dollars without cents', () => {
    expect(formatPrice(500, 'USD')).toBe('$5')
  })

  it('shows cents when there are some', () => {
    expect(formatPrice(499, 'USD')).toBe('$4.99')
  })

  it('shows the symbol of euro and the code of another currency', () => {
    expect(formatPrice(1000, 'EUR')).toBe('€10')
    expect(formatPrice(700, 'CHF')).toBe('7 CHF')
  })

  it('shows nothing for a plan with no price', () => {
    expect(formatPrice(0, 'USD')).toBe('')
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
})

describe('getUsageText', () => {
  it('says how much of the plan is used', () => {
    expect(getUsageText(2 * GIB, 100 * GIB)).toBe('2 GB of 100 GB used')
  })

  it('shows the used space above the quota as it is', () => {
    expect(getUsageText(20 * GIB, 10 * GIB)).toBe('20 GB of 10 GB used')
  })
})
