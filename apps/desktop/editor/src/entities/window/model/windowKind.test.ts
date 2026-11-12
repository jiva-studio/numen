import { describe, expect, it } from 'vitest'
import { getDetachedTabId, isSecondaryWindow } from './windowKind'

describe('the kind of a window', () => {
  it('is secondary where the query names it so, and main otherwise', () => {
    expect(isSecondaryWindow('?window=secondary')).toBe(true)
    expect(isSecondaryWindow('?secondary')).toBe(true)
    expect(isSecondaryWindow('?window=main')).toBe(false)
    expect(isSecondaryWindow('')).toBe(false)
  })

  it('names the tab a detach shows, and none for any other window', () => {
    expect(getDetachedTabId('?secondary&id=plex%3A7')).toBe('plex:7')
    expect(getDetachedTabId('?secondary')).toBe('')
  })
})
