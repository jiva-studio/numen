import { describe, expect, it } from 'vitest'
import { browserTabIds } from './tabIds'
import { NOTE, SETTINGS } from '../lib/workspace'

describe('the identity a tab is filed under', () => {
  it('is a new one every time, so a second of a kind is a second tab', () => {
    const names = [
      browserTabIds.createTabId(NOTE),
      browserTabIds.createTabId(NOTE),
      browserTabIds.createTabId(NOTE),
    ]

    expect(new Set(names).size).toBe(3)
  })

  it('is never the name of a tab of another kind', () => {
    expect(browserTabIds.createTabId(NOTE)).not.toBe(browserTabIds.createTabId(SETTINGS))
  })

  it('says what kind of thing the tab holds', () => {
    expect(browserTabIds.createTabId(SETTINGS).startsWith(`${SETTINGS}:`)).toBe(true)
  })
})

describe('the name a conversation is answered under', () => {
  it('is a new one for every conversation opened', () => {
    const names = [
      browserTabIds.createTabId('agent'),
      browserTabIds.createTabId('agent'),
      browserTabIds.createTabId('agent'),
    ]

    expect(new Set(names).size).toBe(3)
  })
})
