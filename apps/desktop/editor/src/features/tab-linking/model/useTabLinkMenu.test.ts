import { describe, expect, it, vi } from 'vitest'
import type { TabLinkGroup, TabLinksState } from '../types'
import { useTabLinkMenu } from './useTabLinkMenu'

const KINDS: Record<string, string> = { a: 'note', b: 'files', c: 'agent' }

const setup = (groups: Record<string, TabLinkGroup> = {}, targets: string[] = []) => {
  const getGroupOf = vi.fn((tab: string) => groups[tab] ?? null)
  const tabLinks = {
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
    const group = { id: 'g', colorIndex: 0, tabs: ['a', 'b'] }
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
    const group = { id: 'g', colorIndex: 0, tabs: ['a', 'b'] }
    const { menu } = setup({ a: group, b: group }, ['b'])
    menu.open('a', { x: 0, y: 0 })

    expect(menu.isLinked.value).toBe(true)
    expect(menu.linkedTargetTitle.value).toBe('B')
  })
})
