/**
 * What the person has open is told to the core, and each tab draws the icon of its kind.
 */
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { findTabKindIcon } from '@/entities/tab'
import { useOpenTabs } from './useOpenTabs'

interface Entry {
  id: string
  title: string
  kind?: string
  icon?: object
  openTab?: Record<string, string>
  /** The tab is listed but the window cannot resolve it. */
  unresolved?: boolean
}

const create = (entries: Entry[], front: { id: string; kind: string } | null = null) => {
  const writeOpenTabs = vi.fn(async (_tabs: unknown) => {})
  const tabs = ref(entries.map(({ id, title }) => ({ id, title })))
  const windowTabs = {
    handle: { front: () => front },
    tabs,
    getTab: (id: string) => {
      const one = entries.find((e) => e.id === id)
      if (!one || one.unresolved) return undefined
      return {
        state: {},
        kind: { kind: one.kind ?? 'note', icon: one.icon, getOpenTab: () => one.openTab },
      }
    },
  } as unknown as Parameters<typeof useOpenTabs>[0]['windowTabs']
  return { ...useOpenTabs({ core: { writeOpenTabs }, windowTabs }), writeOpenTabs, tabs }
}

describe('the open tabs', () => {
  it('lists each tab with its kind, title, path and what it points at', () => {
    const held = create([
      { id: 'a', title: 'A', kind: 'note', openTab: { path: 'a.md' } },
      { id: 'b', title: 'B', kind: 'document', openTab: { path: 'b.pdf', document: 'doc-1' } },
      { id: 'c', title: 'C', kind: 'recording', openTab: { path: 'c.mp3', recording: 'rec-1' } },
      { id: 'd', title: 'D', kind: 'book', openTab: { path: 'd.epub', book: 'book-1' } },
    ])

    expect(held.open.value.tabs).toStrictEqual([
      { id: 'a', kind: 'note', title: 'A', path: 'a.md' },
      { id: 'b', kind: 'document', title: 'B', path: 'b.pdf', document: 'doc-1' },
      { id: 'c', kind: 'recording', title: 'C', path: 'c.mp3', recording: 'rec-1' },
      { id: 'd', kind: 'book', title: 'D', path: 'd.epub', book: 'book-1' },
    ])
  })

  it('lists a tab the window cannot resolve with an empty kind and path', () => {
    const held = create([{ id: 'a', title: 'A', unresolved: true }])

    expect(held.open.value.tabs).toStrictEqual([{ id: 'a', kind: '', title: 'A', path: '' }])
  })

  it('lists a tab whose kind has no open-tab form with an empty path', () => {
    const held = create([{ id: 'a', title: 'A', kind: 'files' }])

    expect(held.open.value.tabs).toStrictEqual([{ id: 'a', kind: 'files', title: 'A', path: '' }])
  })

  it('tells the core the list when it is made, and again when it changes', async () => {
    const held = create([{ id: 'a', title: 'A', openTab: { path: 'a.md' } }])

    expect(held.writeOpenTabs).toHaveBeenCalledTimes(1)
    expect(held.writeOpenTabs).toHaveBeenLastCalledWith(held.open.value)

    held.tabs.value = [...held.tabs.value, { id: 'b', title: 'B' }]
    await new Promise((r) => setTimeout(r))

    expect(held.writeOpenTabs).toHaveBeenCalledTimes(2)
  })

  it('carries on when the core cannot take the list', () => {
    const rejecting = {
      core: { writeOpenTabs: () => Promise.reject(new Error('down')) },
      windowTabs: { handle: { front: () => null }, tabs: ref([]), getTab: () => undefined },
    } as unknown as Parameters<typeof useOpenTabs>[0]

    expect(() => useOpenTabs(rejecting)).not.toThrow()
  })
})

describe('the active tab', () => {
  it('is the tab in front', () => {
    const held = create(
      [
        { id: 'a', title: 'A' },
        { id: 'b', title: 'B' },
      ],
      { id: 'a', kind: 'note' },
    )

    expect(held.getActiveTabId()).toBe('a')
    expect(held.open.value.front).toBe('a')
  })

  it('is the last tab that is not the agent when the agent is in front', () => {
    const held = create(
      [
        { id: 'a', title: 'A', kind: 'note' },
        { id: 'b', title: 'B', kind: 'plex' },
        { id: 'ag', title: 'Agent', kind: 'agent' },
      ],
      { id: 'ag', kind: 'agent' },
    )

    expect(held.getActiveTabId()).toBe('b')
  })

  it('counts a tab the window cannot resolve as not the agent', () => {
    const held = create(
      [
        { id: 'a', title: 'A', kind: 'note' },
        { id: 'gone', title: 'Gone', unresolved: true },
      ],
      { id: 'ag', kind: 'agent' },
    )

    expect(held.getActiveTabId()).toBe('gone')
  })

  it('is the agent when it is the only tab', () => {
    const held = create([{ id: 'ag', title: 'Agent', kind: 'agent' }], { id: 'ag', kind: 'agent' })

    expect(held.getActiveTabId()).toBe('ag')
  })

  it('is empty when nothing is open', () => {
    expect(create([]).getActiveTabId()).toBe('')
  })
})

describe('the tab icon', () => {
  it('is the icon of the kind of the tab', () => {
    const held = create([{ id: 'a', title: 'A', kind: 'note' }])

    expect(held.getTabIcon('a')).toBe(findTabKindIcon('note'))
    expect(held.getTabIcon('a')).not.toBe(findTabKindIcon(''))
  })

  it('is the icon of no kind for a tab the window cannot resolve', () => {
    const held = create([])

    expect(held.getTabIcon('missing')).toBe(findTabKindIcon(''))
  })
})

describe('the tab icon a kind declares', () => {
  it('is drawn in place of the one the window keeps for that kind', () => {
    const icon = { name: 'DeclaredIcon' }
    const held = create([{ id: 'p', title: 'P', kind: 'agent', icon }])

    expect(held.getTabIcon('p')).toBe(icon)
  })
})
