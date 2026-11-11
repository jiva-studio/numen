/**
 * The open tabs reported to the core, and each tab rendering the icon of its kind.
 */
import { nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { findTabKindIcon } from '../lib/icons'
import { parseSavedTabs, useOpenTabs } from './useOpenTabs'

/** The write of the open tabs to the core, which is the edge of the entity. */
const coreCalls = { writeOpenTabs: vi.fn<(open: unknown) => Promise<void>>() }

beforeEach(() => {
  coreCalls.writeOpenTabs.mockReset()
  coreCalls.writeOpenTabs.mockImplementation(() => Promise.resolve())
})

interface Entry {
  id: string
  title: string
  kind?: string
  icon?: object
  openTab?: Record<string, unknown>
  savedState?: unknown
  unresolved?: boolean
  takesText?: boolean
}

const create = (
  entries: Entry[],
  front: { id: string; kind: string } | null = null,
  isWriting = ref(true),
) => {
  const { writeOpenTabs } = coreCalls
  const tabs = ref(entries.map(({ id, title }) => ({ id, title })))
  const windowTabs = {
    groups: {
      handle: { front: () => front },
      findKind: (isWanted: (kind: { kind: string }) => boolean) =>
        entries
          .map((one) => ({ kind: one.kind ?? 'note', display: { icon: one.icon } }))
          .find(isWanted),
    },
    tabs: {
      tabs,
      getTab: (id: string) => {
        const one = entries.find((e) => e.id === id)
        if (!one || one.unresolved) return undefined
        return {
          state: {},
          kind: {
            kind: one.kind ?? 'note',
            display: { icon: one.icon },
            address: {
              ...(one.openTab ? { getOpenTab: () => one.openTab } : {}),
              ...(one.savedState !== undefined ? { getState: () => one.savedState } : {}),
            },
            events: one.takesText ? { receiveText: vi.fn() } : {},
          },
        }
      },
    },
  } as unknown as Parameters<typeof useOpenTabs>[0]['windowTabs']
  return {
    ...useOpenTabs({ windowTabs, isWriting, writeMessage: vi.fn(), writeOpenTabs }),
    writeOpenTabs,
    tabs,
  }
}

describe('the open tabs', () => {
  it('lists each tab with its kind, title, path and what it points at', () => {
    const target = create([
      { id: 'a', title: 'A', kind: 'note', openTab: { path: 'a.md' } },
      { id: 'b', title: 'B', kind: 'document', openTab: { path: 'b.pdf', document: 'doc-1' } },
      { id: 'c', title: 'C', kind: 'recording', openTab: { path: 'c.mp3', recording: 'rec-1' } },
      { id: 'd', title: 'D', kind: 'book', openTab: { path: 'd.epub', book: 'book-1' } },
    ])

    expect(target.open.value.tabs).toStrictEqual([
      { id: 'a', kind: 'note', title: 'A', path: 'a.md', state: '' },
      { id: 'b', kind: 'document', title: 'B', path: 'b.pdf', state: '' },
      { id: 'c', kind: 'recording', title: 'C', path: 'c.mp3', state: '' },
      { id: 'd', kind: 'book', title: 'D', path: 'd.epub', state: '' },
    ])
  })

  it('lists each reader tab with the place its reader reported', () => {
    const target = create([
      { id: 'a', title: 'A', kind: 'note', openTab: { path: 'a.md' } },
      {
        id: 'b',
        title: 'B',
        kind: 'reader:reader-pdf',
        openTab: { path: 'b.pdf', place: { page: 7, pageCount: 12 } },
        savedState: { page: 7, pageCount: 12 },
      },
      {
        id: 'c',
        title: 'C',
        kind: 'reader:recording',
        openTab: { path: 'c.mp3', place: { positionSeconds: 0 } },
      },
      {
        id: 'd',
        title: 'D',
        kind: 'reader:reader-epub',
        openTab: {
          path: 'd.epub',
          place: { cfi: 'epubcfi(/6/4!/4/2/1:0)', chapter: 'The Reed Beds' },
        },
      },
    ])

    expect(target.open.value.tabs).toStrictEqual([
      { id: 'a', kind: 'note', title: 'A', path: 'a.md', state: '' },
      {
        id: 'b',
        kind: 'reader:reader-pdf',
        title: 'B',
        path: 'b.pdf',
        state: '{"page":7,"pageCount":12}',
        place: { page: 7, pageCount: 12 },
      },
      {
        id: 'c',
        kind: 'reader:recording',
        title: 'C',
        path: 'c.mp3',
        state: '',
        place: { positionSeconds: 0 },
      },
      {
        id: 'd',
        kind: 'reader:reader-epub',
        title: 'D',
        path: 'd.epub',
        state: '',
        place: { cfi: 'epubcfi(/6/4!/4/2/1:0)', chapter: 'The Reed Beds' },
      },
    ])
  })

  it('sends the core the place of each tab with the list', () => {
    const target = create([
      {
        id: 'b',
        title: 'B',
        kind: 'reader:reader-pdf',
        openTab: { path: 'b.pdf', place: { page: 3, pageCount: 12 } },
      },
    ])

    expect(target.writeOpenTabs).toHaveBeenLastCalledWith({
      activeId: 'b',
      tabs: [
        {
          id: 'b',
          kind: 'reader:reader-pdf',
          title: 'B',
          path: 'b.pdf',
          state: '',
          place: { page: 3, pageCount: 12 },
        },
      ],
    })
  })

  it('lists a tab the window cannot resolve with an empty kind and path', () => {
    const target = create([{ id: 'a', title: 'A', unresolved: true }])

    expect(target.open.value.tabs).toStrictEqual([
      { id: 'a', kind: '', title: 'A', path: '', state: '' },
    ])
  })

  it('lists a tab whose kind has no open-tab form with an empty path', () => {
    const target = create([{ id: 'a', title: 'A', kind: 'files' }])

    expect(target.open.value.tabs).toStrictEqual([
      { id: 'a', kind: 'files', title: 'A', path: '', state: '' },
    ])
  })

  it('notifies the core the list when it is made, and again when it changes', async () => {
    const tab = create([{ id: 'a', title: 'A', openTab: { path: 'a.md' } }])

    expect(tab.writeOpenTabs).toHaveBeenCalledTimes(1)
    expect(tab.writeOpenTabs).toHaveBeenLastCalledWith(tab.open.value)

    tab.tabs.value = [...tab.tabs.value, { id: 'b', title: 'B' }]
    await new Promise((r) => setTimeout(r))

    expect(tab.writeOpenTabs).toHaveBeenCalledTimes(2)
  })

  it('waits for an in-flight write before sending the latest list', async () => {
    let releaseFirst!: () => void
    let calls = 0
    const received: unknown[] = []
    const { writeOpenTabs } = coreCalls
    writeOpenTabs.mockImplementation((open: unknown) => {
      calls++
      received.push(open)
      if (calls === 1) {
        return new Promise<void>((resolve) => {
          releaseFirst = resolve
        })
      }
      return Promise.resolve()
    })
    const tabs = ref([{ id: 'a', title: 'A' }])
    const windowTabs = {
      groups: { handle: { front: () => null } },
      tabs: {
        tabs,
        getTab: (id: string) => ({
          state: {},
          kind: { kind: 'note', getOpenTab: () => ({ path: `${id}.md` }) },
        }),
      },
    } as unknown as Parameters<typeof useOpenTabs>[0]['windowTabs']

    useOpenTabs({ windowTabs, writeMessage: vi.fn(), writeOpenTabs })
    expect(writeOpenTabs).toHaveBeenCalledTimes(1)

    tabs.value = [
      { id: 'a', title: 'A' },
      { id: 'b', title: 'B' },
    ]
    await new Promise((r) => setTimeout(r))
    tabs.value = [
      { id: 'a', title: 'A' },
      { id: 'b', title: 'B' },
      { id: 'c', title: 'C' },
    ]
    await new Promise((r) => setTimeout(r))
    expect(writeOpenTabs).toHaveBeenCalledTimes(1)

    releaseFirst()
    await new Promise((r) => setTimeout(r))
    expect(writeOpenTabs).toHaveBeenCalledTimes(2)
    expect((received[1] as { tabs: { id: string }[] }).tabs.map((t) => t.id)).toStrictEqual([
      'a',
      'b',
      'c',
    ])

    tabs.value = [{ id: 'a', title: 'A' }]
    await new Promise((r) => setTimeout(r))
    expect(writeOpenTabs).toHaveBeenCalledTimes(3)
  })

  it('carries on when the core cannot take the list', () => {
    coreCalls.writeOpenTabs.mockRejectedValue(new Error('down'))
    const rejecting = {
      windowTabs: {
        groups: { handle: { front: () => null } },
        tabs: { tabs: ref([]), getTab: () => undefined },
      },
      writeMessage: vi.fn(),
      writeOpenTabs: coreCalls.writeOpenTabs,
    } as unknown as Parameters<typeof useOpenTabs>[0]

    expect(() => useOpenTabs(rejecting)).not.toThrow()
  })
})

describe('the active tab', () => {
  it('is the tab in front', () => {
    const target = create(
      [
        { id: 'a', title: 'A' },
        { id: 'b', title: 'B' },
      ],
      { id: 'a', kind: 'note' },
    )

    expect(target.getActiveTabId()).toBe('a')
    expect(target.open.value.activeId).toBe('a')
  })

  it('is the last tab that takes no text when one that takes text is in front', () => {
    const target = create(
      [
        { id: 'a', title: 'A', kind: 'note' },
        { id: 'b', title: 'B', kind: 'graph' },
        { id: 'ag', title: 'Chat', kind: 'chat', takesText: true },
      ],
      { id: 'ag', kind: 'chat' },
    )

    expect(target.getActiveTabId()).toBe('b')
  })

  it('counts a tab the window cannot resolve as one that takes no text', () => {
    const target = create(
      [
        { id: 'a', title: 'A', kind: 'note' },
        { id: 'ag', title: 'Chat', kind: 'chat', takesText: true },
        { id: 'gone', title: 'Gone', unresolved: true },
      ],
      { id: 'ag', kind: 'chat' },
    )

    expect(target.getActiveTabId()).toBe('gone')
  })

  it('is the tab that takes text when it is the only tab', () => {
    const target = create([{ id: 'ag', title: 'Chat', kind: 'chat', takesText: true }], {
      id: 'ag',
      kind: 'chat',
    })

    expect(target.getActiveTabId()).toBe('ag')
  })

  it('is empty when nothing is open', () => {
    expect(create([]).getActiveTabId()).toBe('')
  })
})

describe('the tab icon', () => {
  it('is the icon of the kind of the tab', () => {
    const target = create([{ id: 'a', title: 'A', kind: 'note' }])

    expect(target.getTabIcon('a')).toBe(findTabKindIcon('note'))
    expect(target.getTabIcon('a')).not.toBe(findTabKindIcon(''))
  })

  it('is the icon of no kind for a tab the window cannot resolve', () => {
    const target = create([])

    expect(target.getTabIcon('missing')).toBe(findTabKindIcon(''))
  })
})

describe('the tab icon a kind declares', () => {
  it('is drawn in place of the one the window keeps for that kind', () => {
    const icon = { name: 'DeclaredIcon' }
    const target = create([{ id: 'p', title: 'P', kind: 'agent', icon }])

    expect(target.getTabIcon('p')).toBe(icon)
  })
})

describe('the open tabs while the window leaves its vault', () => {
  it('waits for a write in flight, and writes nothing once paused', async () => {
    const target = create([{ id: 'a', title: 'A', kind: 'note', openTab: { path: 'a.md' } }])
    const writeLog: string[] = []
    let release: () => void = () => undefined
    target.writeOpenTabs.mockImplementation((tabs: unknown) => {
      writeLog.push(
        JSON.stringify((tabs as { tabs: { title: string }[] }).tabs.map((t) => t.title)),
      )
      return new Promise<void>((resolve) => {
        release = resolve
      })
    })
    target.tabs.value = [{ id: 'a', title: 'A1' }]
    await nextTick()

    const paused = target.pause()
    release()
    await Promise.resolve()
    release()
    await paused

    expect(writeLog.at(-1)).toBe('["A1"]')
    const writeCount = writeLog.length
    target.tabs.value = [{ id: 'a', title: 'After the switch' }]
    await nextTick()
    await Promise.resolve()
    expect(writeLog).toHaveLength(writeCount)
  })

  it('writes the tabs again once resumed', async () => {
    const target = create([{ id: 'a', title: 'A', kind: 'note', openTab: { path: 'a.md' } }])
    await target.pause()
    const writeCount = target.writeOpenTabs.mock.calls.length
    target.tabs.value = [{ id: 'a', title: 'A2' }]
    await nextTick()
    expect(target.writeOpenTabs).toHaveBeenCalledTimes(writeCount)

    target.resume()
    await nextTick()

    expect(target.writeOpenTabs).toHaveBeenCalledTimes(writeCount + 1)
  })

  it('does not flush when paused while isWriting is false, and flushes pending state when true', async () => {
    const isWriting = ref(false)
    const target = create(
      [{ id: 'a', title: 'A', kind: 'note', openTab: { path: 'a.md' }, savedState: { page: 2 } }],
      null,
      isWriting,
    )
    expect(target.open.value.tabs[0]?.state).toBe('{"page":2}')
    expect(target.writeOpenTabs).not.toHaveBeenCalled()

    await target.pause()
    expect(target.writeOpenTabs).not.toHaveBeenCalled()

    target.resume()
    isWriting.value = true
    await nextTick()
    target.writeOpenTabs.mockClear()
    await target.pause()
    expect(target.writeOpenTabs).toHaveBeenCalledTimes(1)
  })

  it('parses saved tabs and filters out entries with empty id or kind', () => {
    expect(
      parseSavedTabs({
        activeId: 'a',
        tabs: [
          { id: '', kind: 'note', title: 'Empty id', path: 'x.md', state: '' },
          { id: 'b', kind: '', title: 'Empty kind', path: 'y.md', state: '' },
          { id: 'a', kind: 'note', title: 'A', path: 'a.md', state: '{"scroll":10}' },
          { id: 'c', kind: 'note', title: '', path: 'c.md', state: '{bad json' },
        ],
      }),
    ).toEqual([
      { id: 'a', kind: 'note', title: 'A', path: 'a.md', saved: { scroll: 10 } },
      { id: 'c', kind: 'note', path: 'c.md', saved: undefined },
    ])
  })
})

describe('getKindIcon', () => {
  it('returns the icon the kind declares', () => {
    const icon = { name: 'DeclaredIcon' }
    const target = create([{ id: 'p', title: 'P', kind: 'agent', icon }])

    expect(target.getKindIcon('agent')).toBe(icon)
  })

  it('falls back to the built-in icon of the kind', () => {
    const target = create([])

    expect(target.getKindIcon('note')).toBe(findTabKindIcon('note'))
  })
})
