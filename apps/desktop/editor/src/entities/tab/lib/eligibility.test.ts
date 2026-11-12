import { describe, expect, it } from 'vitest'
import { getAllTabs, getEligibleTabs } from './eligibility'
import type { TabLinkGroup } from './links'

const TABS = [
  { id: 'a', title: 'A' },
  { id: 'b', title: 'B' },
  { id: 'c', title: 'C' },
  { id: 'd', title: 'D' },
]
const KINDS: Record<string, string> = { a: 'note', b: 'files', c: 'agent', d: 'plex' }
const LINKABLE = new Set(['a', 'b', 'd'])

const input = (groups: Record<string, TabLinkGroup> = {}) => ({
  from: 'a',
  tabs: TABS,
  getKind: (tab: string) => KINDS[tab],
  isLinkable: (tab: string) => LINKABLE.has(tab),
  getGroupOf: (tab: string) => groups[tab] ?? null,
})

describe('the tabs a link may end at', () => {
  it('are the linkable ones other than the tab itself', () => {
    expect(getEligibleTabs(input()).map((tab) => tab.id)).toStrictEqual(['b', 'd'])
  })

  it('leave out the tabs of the group the tab is in', () => {
    const group = { id: 'g', colorIndex: 0, tabs: ['a', 'b'], path: '', title: '' }
    expect(getEligibleTabs(input({ a: group, b: group })).map((tab) => tab.id)).toStrictEqual(['d'])
  })

  it('say the kind and whether the tab is linked already', () => {
    const group = { id: 'g', colorIndex: 0, tabs: ['d', 'x'], path: '', title: '' }
    expect(getEligibleTabs(input({ d: group }))).toStrictEqual([
      { id: 'b', title: 'B', kind: 'files', isLinked: false },
      { id: 'd', title: 'D', kind: 'plex', isLinked: true },
    ])
  })

  it('are none when the origin tab itself cannot be linked', () => {
    expect(getEligibleTabs({ ...input(), from: 'c' })).toStrictEqual([])
  })
})

describe('getAllTabs', () => {
  it('returns the tabs of this window, then the other tabs not open here', () => {
    const own = [{ id: 'plex:1', title: 'Orbits' }]
    const linkable = [
      { id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' },
      { id: 'files:2', kind: 'files', title: 'Files', windowId: 'window-b' },
    ]

    expect(getAllTabs(own, linkable)).toStrictEqual([
      { id: 'plex:1', title: 'Orbits' },
      { id: 'files:2', title: 'Files' },
    ])
  })
})
