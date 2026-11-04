import { describe, expect, it, vi } from 'vitest'
import type { TabLinkGroup, TabLinksState } from '../types'
import { useTabLinkMenu } from './useTabLinkMenu'

const KINDS: Record<string, string> = { a: 'note', b: 'files', c: 'agent' }

const setup = (groups: Record<string, TabLinkGroup> = {}, targets: string[] = []) => {
  const getGroupOf = vi.fn((tab: string) => groups[tab] ?? null)
  const tabLinks = {
    tabs: [],
    getGroupOf,
    getLinkedTargets: () => targets,
    linkTabs: vi.fn(),
    unlinkTab: vi.fn(),
  } as unknown as TabLinksState & { getGroupOf: ReturnType<typeof vi.fn> }
  const menu = useTabLinkMenu({
    tabLinks,
    getTabs: () => [
      { id: 'a', title: 'A' },
      { id: 'b', title: 'B' },
      { id: 'c', title: 'C' },
    ],
    getKind: (tab) => KINDS[tab],
    isLinkable: (tab) => tab !== 'c' && tab in KINDS,
  })
  return { menu, tabLinks }
}

describe('the menu on a tab', () => {
  it('is closed until requested for', () => {
    const group = { id: 'g', colorIndex: 0, tabs: ['a', 'b'], path: '', title: '' }
    const { menu, tabLinks } = setup({ a: group }, ['b'])
    expect(menu.tabId.value).toBeNull()
    expect(menu.at.value).toStrictEqual({ x: 0, y: 0 })
    expect(menu.eligibleTabs.value).toStrictEqual([])
    expect(menu.linkedTargetTitle.value).toBe('')
    expect(menu.isLinked.value).toBe(false)
    expect(tabLinks.getGroupOf).not.toHaveBeenCalled()
  })

  it('reports an unlinked tab as not linked when open', () => {
    const { menu } = setup()
    menu.open('a', { x: 5, y: 6 })
    expect(menu.isLinked.value).toBe(false)
    expect(menu.linkedTargetTitle.value).toBe('')
  })

  it('opens on a tab of a kind that can be linked, where it was requested', () => {
    const { menu } = setup()
    menu.open('a', { x: 5, y: 6 })

    expect(menu.tabId.value).toBe('a')
    expect(menu.at.value).toStrictEqual({ x: 5, y: 6 })
    expect(menu.eligibleTabs.value.map((tab) => tab.id)).toStrictEqual(['b'])
  })

  it('stays closed on a tab of a kind that cannot be linked when detach is not provided', () => {
    const { menu } = setup()
    menu.open('c', { x: 5, y: 6 })
    expect(menu.tabId.value).toBeNull()
    expect(menu.canDetach.value).toBe(false)
  })

  it('opens on an unlinkable tab when detach is provided, and detaches it on request', () => {
    const onDetach = vi.fn()
    const tabLinks = {
      tabs: [],
      getGroupOf: () => null,
      getLinkedTargets: () => [],
      linkTabs: vi.fn(),
      unlinkTab: vi.fn(),
    } as unknown as TabLinksState
    const menu = useTabLinkMenu({
      tabLinks,
      getTabs: () => [{ id: 'c', title: 'C' }],
      getKind: (tab) => KINDS[tab],
      isLinkable: () => false,
      onDetach,
    })

    menu.open('c', { x: 12, y: 34 })
    expect(menu.tabId.value).toBe('c')
    expect(menu.canDetach.value).toBe(true)
    expect(menu.eligibleTabs.value).toStrictEqual([])

    menu.detach()
    expect(onDetach).toHaveBeenCalledWith('c', { x: 12, y: 34 })
    expect(menu.tabId.value).toBeNull()
  })

  it('links the tab to the one chosen and closes', () => {
    const { menu, tabLinks } = setup()
    menu.open('a', { x: 0, y: 0 })
    menu.link('b')

    expect(tabLinks.linkTabs).toHaveBeenCalledWith('a', 'b')
    expect(menu.tabId.value).toBeNull()
  })

  it('unlinks the tab and closes', () => {
    const { menu, tabLinks } = setup()
    menu.open('a', { x: 0, y: 0 })
    menu.unlink()

    expect(tabLinks.unlinkTab).toHaveBeenCalledWith('a')
    expect(menu.tabId.value).toBeNull()
  })

  it('does nothing when it is closed', () => {
    const { menu, tabLinks } = setup()
    menu.link('b')
    menu.unlink()
    menu.detach()
    expect(tabLinks.linkTabs).not.toHaveBeenCalled()
    expect(tabLinks.unlinkTab).not.toHaveBeenCalled()
  })

  it('says what the tab is linked to', () => {
    const group = { id: 'g', colorIndex: 0, tabs: ['a', 'b'], path: '', title: '' }
    const { menu } = setup({ a: group, b: group }, ['b'])
    menu.open('a', { x: 0, y: 0 })

    expect(menu.isLinked.value).toBe(true)
    expect(menu.linkedTargetTitle.value).toBe('B')
  })
})

describe('useTabLinkMenu with other windows open', () => {
  const OTHER_TABS = [
    { id: 'a', kind: 'note', title: 'A', windowId: 'window-a' },
    { id: 'plex:7', kind: 'plex', title: 'Tides', windowId: 'window-b' },
    { id: 'files:8', kind: 'files', title: 'Files', windowId: 'window-b' },
  ]

  const setupWithOthers = (groups: Record<string, TabLinkGroup> = {}, targets: string[] = []) => {
    const tabLinks = {
      tabs: OTHER_TABS,
      getGroupOf: (tab: string) => groups[tab] ?? null,
      getLinkedTargets: () => targets,
      linkTabs: vi.fn(),
      unlinkTab: vi.fn(),
    } as unknown as TabLinksState
    return useTabLinkMenu({
      tabLinks,
      getTabs: () => [
        { id: 'a', title: 'A' },
        { id: 'b', title: 'B' },
      ],
      getKind: (tab) => KINDS[tab],
      isLinkable: (tab) => tab === 'a' || tab === 'b',
    })
  }

  it('lists the tabs of this window, then the linkable tabs of other windows', () => {
    const menu = setupWithOthers()
    menu.open('a', { x: 0, y: 0 })

    expect(menu.eligibleTabs.value).toStrictEqual([
      { id: 'b', title: 'B', kind: 'files', isLinked: false },
      { id: 'plex:7', title: 'Tides', kind: 'plex', isLinked: false },
      { id: 'files:8', title: 'Files', kind: 'files', isLinked: false },
    ])
  })

  it('excludes a tab of this window whose kind cannot link, even when the core lists it', () => {
    const tabLinks = {
      tabs: [{ id: 'c', kind: 'agent', title: 'C', windowId: 'window-a' }],
      getGroupOf: () => null,
      getLinkedTargets: () => [],
    } as unknown as TabLinksState
    const menu = useTabLinkMenu({
      tabLinks,
      getTabs: () => [
        { id: 'a', title: 'A' },
        { id: 'c', title: 'C' },
      ],
      getKind: (tab) => KINDS[tab],
      isLinkable: (tab) => tab === 'a',
    })
    menu.open('a', { x: 0, y: 0 })

    expect(menu.eligibleTabs.value).toStrictEqual([])
  })

  it('shows the title of a linked tab of another window', () => {
    const group = { id: 'g', colorIndex: 1, tabs: ['a', 'plex:7'], path: '', title: '' }
    const menu = setupWithOthers({ a: group, 'plex:7': group }, ['plex:7'])
    menu.open('a', { x: 0, y: 0 })

    expect(menu.linkedTargetTitle.value).toBe('Tides')
    expect(menu.eligibleTabs.value.map((tab) => tab.id)).toStrictEqual(['b', 'files:8'])
  })
})

describe('the Move to window menu state', () => {
  const WINDOWS = [
    { id: 'window-a', isMain: true, activeTitle: 'Orbits', activeKind: 'note', tabCount: 2 },
    { id: 'window-b', isMain: false, activeTitle: 'Tides', activeKind: 'plex', tabCount: 1 },
    { id: 'window-c', isMain: false, activeTitle: 'Index', activeKind: 'files', tabCount: 1 },
  ]

  const setupMove = (asking: readonly string[] = []) => {
    const tabLinks = {
      tabs: [],
      windows: WINDOWS,
      getGroupOf: () => null,
      getLinkedTargets: () => [],
      linkTabs: vi.fn(),
      unlinkTab: vi.fn(),
    } as unknown as TabLinksState
    const onMove = vi.fn()
    const onHighlight = vi.fn()
    const menu = useTabLinkMenu({
      tabLinks,
      getTabs: () => [{ id: 'c', title: 'C' }],
      getKind: (tab) => KINDS[tab],
      isLinkable: () => false,
      windowId: 'window-b',
      isAsking: (tab) => asking.includes(tab),
      onMove,
      onHighlight,
    })
    return { menu, onMove, onHighlight }
  }

  it('opens for a tab that can only be moved and lists every window except this one', () => {
    const { menu } = setupMove()

    menu.open('c', { x: 1, y: 2 })

    expect(menu.tabId.value).toBe('c')
    expect(menu.windows.value.map((one) => one.id)).toStrictEqual(['window-a', 'window-c'])
    expect(menu.isAsking.value).toBe(false)
  })

  it('reports an asking tab', () => {
    const { menu } = setupMove(['c'])

    menu.open('c', { x: 1, y: 2 })

    expect(menu.isAsking.value).toBe(true)
  })

  it('closes the menu and then calls onMove with the chosen window', () => {
    const { menu, onMove } = setupMove()
    menu.open('c', { x: 1, y: 2 })

    menu.move('window-a')

    expect(menu.tabId.value).toBeNull()
    expect(onMove).toHaveBeenCalledExactlyOnceWith('c', 'window-a')
  })

  it('highlights one window at a time and removes the highlight when the menu closes', () => {
    const { menu, onHighlight } = setupMove()
    menu.open('c', { x: 1, y: 2 })

    menu.highlight('window-a')
    menu.highlight('window-a')
    menu.highlight('window-c')
    menu.highlight(null)
    menu.highlight('window-a')
    menu.close()

    expect(onHighlight.mock.calls).toStrictEqual([
      ['window-a', true],
      ['window-a', false],
      ['window-c', true],
      ['window-c', false],
      ['window-a', true],
      ['window-a', false],
    ])
  })

  it('lists no windows without onMove', () => {
    const tabLinks = {
      tabs: [],
      windows: WINDOWS,
      getGroupOf: () => null,
    } as unknown as TabLinksState
    const menu = useTabLinkMenu({
      tabLinks,
      getTabs: () => [],
      getKind: () => undefined,
      isLinkable: () => true,
    })

    expect(menu.windows.value).toStrictEqual([])
  })
})
