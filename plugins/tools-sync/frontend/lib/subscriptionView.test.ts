import { describe, expect, it } from 'vitest'
import { formatSize, getPlanLabel, getPlanName, getUsageText } from './subscriptionView'

const GIB = 2 ** 30

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

describe('getPlanName', () => {
  it('names the plans the client knows', () => {
    expect(getPlanName('basic')).toBe('Basic')
    expect(getPlanName('plus')).toBe('Plus')
    expect(getPlanName('pro')).toBe('Pro')
  })

  it('has no name for another id', () => {
    expect(getPlanName('gold')).toBe('')
  })
})

describe('getPlanLabel', () => {
  it('joins the name and the size of a known plan', () => {
    expect(getPlanLabel({ id: 'plus', quotaBytes: 100 * GIB })).toBe('Plus, 100 GB')
  })

  it('shows only the size of a plan the client does not know', () => {
    expect(getPlanLabel({ id: 'gold', quotaBytes: 50 * GIB })).toBe('50 GB')
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
