import { describe, expect, it, vi } from 'vitest'
import type { TabLinkGroup, TabLinksState } from '../types'
import { useTabLinkMenu } from './useTabLinkMenu'

const KINDS: Record<string, string> = { a: 'note', b: 'files', c: 'agent' }

const setup = (groups: Record<string, TabLinkGroup> = {}, targets: string[] = []) => {
  const tabLinks = {
    getGroupOf: (tab: string) => groups[tab] ?? null,
    getLinkedTargets: () => targets,
    linkTabs: vi.fn(),
    unlinkTab: vi.fn(),
  } as unknown as TabLinksState
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
    const { menu } = setup()
    expect(menu.tabId.value).toBeNull()
    expect(menu.eligibleTabs.value).toStrictEqual([])
  })

  it('opens on a tab of a kind that can be linked, where it was requested', () => {
    const { menu } = setup()
    menu.open('a', { x: 5, y: 6 })

    expect(menu.tabId.value).toBe('a')
    expect(menu.at.value).toStrictEqual({ x: 5, y: 6 })
    expect(menu.eligibleTabs.value.map((tab) => tab.id)).toStrictEqual(['b'])
  })

  it('stays closed on a tab of a kind that cannot be linked', () => {
    const { menu } = setup()
    menu.open('c', { x: 5, y: 6 })
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
