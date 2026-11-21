import { describe, expect, it } from 'vitest'
import { getContributionSummary } from './contributionSummary'

const item = { text: 'a', detail: '', keys: '' }

describe('getContributionSummary', () => {
  it('counts tools and search functions', () => {
    expect(
      getContributionSummary([
        { kind: 'tools', items: [item, item] },
        { kind: 'searchFunctions', items: [item] },
        { kind: 'commands', items: [item] },
      ]),
    ).toBe('2 tools, 1 search function')
  })

  it('is empty without them', () => {
    expect(getContributionSummary([{ kind: 'commands', items: [item] }])).toBe('')
    expect(getContributionSummary(undefined)).toBe('')
  })
})
