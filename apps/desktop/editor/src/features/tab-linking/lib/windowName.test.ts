import { describe, expect, it } from 'vitest'
import { getWindowName } from './windowName'

const TIDES = {
  id: 'window-b',
  isMain: false,
  activeTitle: 'Tides',
  activeKind: 'plex',
  tabCount: 1,
}

describe('the name of a window', () => {
  it('is the title of its active tab and the count of its other tabs', () => {
    expect(getWindowName(TIDES)).toBe('Tides')
    expect(getWindowName({ ...TIDES, activeTitle: 'Orbits', tabCount: 3 })).toBe('Orbits (+2)')
  })

  it('is the kind of its active tab without a title, or Empty window without a tab', () => {
    expect(getWindowName({ ...TIDES, activeTitle: '' })).toBe('plex')
    expect(getWindowName({ ...TIDES, activeTitle: '', activeKind: '', tabCount: 0 })).toBe(
      'Empty window',
    )
  })
})
