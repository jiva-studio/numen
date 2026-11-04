/**
 * The editors of a window: what each is made with, what is flushed, what raises conflicts.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createNoteEditors } from './useNoteEditors'

interface MockParts {
  changes?: { reportNoteChange: unknown; close: unknown }
  notes: {
    flush: unknown
    has: (id: string) => boolean
    getPath: (id: string) => string
    getOpenIds?: () => readonly string[]
    stale: () => null
    flushPending: unknown
    close: unknown
  }
  making: { name: string }
  fileFlush: { addHandler: ReturnType<typeof vi.fn>; close: ReturnType<typeof vi.fn> }
  reached: { name: string }
}

interface MockCalls {
  raised: unknown[][]
  openNotes: [unknown, { onReplaced: () => void }]
  writer: [unknown, unknown]
  flush: unknown[]
  reached: [unknown, unknown]
}

const h = vi.hoisted(() => ({
  calls: {} as MockCalls,
  parts: {} as MockParts,
}))

vi.mock('@/entities/note', () => ({
  openNotes: (...args: unknown[]) => {
    h.calls.openNotes = args as MockCalls['openNotes']
    return h.parts.notes
  },
  createNoteWriter: (...args: unknown[]) => {
    h.calls.writer = args as MockCalls['writer']
    return h.parts.making
  },
}))
vi.mock('@/features/file-conflict', () => ({
  createFileFlush: (...args: unknown[]) => {
    h.calls.flush = args
    return h.parts.fileFlush
  },
  raiseConflicts: (...args: unknown[]) => {
    h.calls.raised.push(args)
  },
}))
vi.mock('@/features/command-palette', () => ({
  createNotes: (...args: unknown[]) => {
    h.calls.reached = args as MockCalls['reached']
    return h.parts.reached
  },
}))

beforeEach(() => {
  h.calls = {
    raised: [],
    openNotes: [null, { onReplaced: vi.fn() }],
    writer: [null, null],
    flush: [],
    reached: [null, null],
  }
  h.parts = {
    changes: { reportNoteChange: vi.fn(), close: vi.fn() },
    notes: {
      flush: { name: 'notes.flush' },
      has: (id: string) => id === 'known-note',
      getPath: (id: string) => 'title of ' + id,
      getOpenIds: () => ['known-note', 'titled-note'],
      stale: () => null,
      flushPending: vi.fn(),
      close: vi.fn(),
    },
    making: { name: 'making' },
    fileFlush: { addHandler: vi.fn(), close: vi.fn() },
    reached: { name: 'reached' },
  }
})

const create = () => {
  const writers: Record<string, unknown> = {}
  const log = {
    getWriter: vi.fn((name: string) => (writers[name] ??= vi.fn())),
  }
  const core = { name: 'core' }
  const tabOpeners = { name: 'openers' }
  const windowTabs = {
    tabs: { value: [{ id: 'titled-note', title: 'Tide table' }] },
    getTab: vi.fn(),
  }
  const editors = createNoteEditors({ core, log, tabOpeners, windowTabs } as unknown as Parameters<
    typeof createNoteEditors
  >[0])
  return { editors, writers, core, tabOpeners, windowTabs }
}

describe('the note editors', () => {
  it('opens the notes with the change handler for a replaced note', () => {
    const { core } = create()

    expect(h.calls.openNotes[0]).toBe(core)
    expect(typeof h.calls.openNotes[1].onReplaced).toBe('function')
  })

  it('writes the message of making a note through the writer of its own name', () => {
    const { writers } = create()

    expect(h.calls.writer[1]).toBe(writers.made)
    expect(Object.keys(writers)).toEqual(['made'])
  })

  it('flushes the notes', () => {
    create()

    expect(h.parts.fileFlush.addHandler.mock.calls.map((c: unknown[]) => c[0])).toEqual([
      h.parts.notes.flush,
    ])
  })

  it('raises conflicts for the notes, in the flush', () => {
    create()

    expect(h.calls.raised).toEqual([[h.parts.notes, h.parts.fileFlush]])
  })

  it('keeps the store of notes and reaches notes through it', () => {
    const { editors, tabOpeners } = create()

    expect(editors.stores.length).toBe(1)
    expect(h.calls.reached[0]).toBe(editors.stores)
    expect(h.calls.reached[1]).toBe(tabOpeners)
    expect(editors.reached).toBe(h.parts.reached)
  })

  it('gets a title from the store that has the id, and an empty one from none', () => {
    const { editors } = create()

    expect(editors.getTitle('known-note')).toBe('title of known-note')
    expect(editors.getTitle('unknown')).toBe('')
  })

  it('closes the changes and the flush', () => {
    const { editors } = create()
    const closeChanges = vi.spyOn(editors.changes, 'close')

    editors.close()

    expect(closeChanges).toHaveBeenCalledTimes(1)
    expect(h.parts.fileFlush.close).toHaveBeenCalledTimes(1)
  })

  it('measures all open note tabs via state.measure or state.editor.measure', () => {
    const { editors, windowTabs } = create()
    const measureDirect = vi.fn()
    const measureNested = vi.fn()
    windowTabs.tabs.value = [
      { id: 't1', title: 'Direct' },
      { id: 't2', title: 'Nested' },
      { id: 't3', title: 'None' },
      { id: 't4', title: 'Missing' },
    ]
    windowTabs.getTab.mockImplementation((id: string) => {
      if (id === 't1') return { state: { measure: measureDirect } }
      if (id === 't2') return { state: { editor: { measure: measureNested } } }
      if (id === 't4') return null
      return { state: {} }
    })

    editors.noted.measureAll()

    expect(measureDirect).toHaveBeenCalledTimes(1)
    expect(measureNested).toHaveBeenCalledTimes(1)
  })
})

describe('the store of open notes, as the palette reads it', () => {
  it('names a note by the title of the tab it is open in', () => {
    const { editors } = create()
    h.parts.notes.has = (id: string) => id === 'titled-note'

    expect(editors.getTitle('titled-note')).toBe('Tide table')
  })

  it('finds the tab a path is open in, and none for a path no tab holds', () => {
    const { editors } = create()
    const store = editors.stores[0]!

    expect(store.getTabAt('title of titled-note')).toBe('titled-note')
    expect(store.getTabAt('notes/Elsewhere.md')).toBeNull()
  })

  it('delegates has, getPath, asking, flushPending, and close with error reporting', async () => {
    const { editors, writers } = create()
    const store = editors.stores[0]!

    expect(store.has('known-note')).toBe(true)
    expect(store.has('other')).toBe(false)
    expect(store.getPath('known-note')).toBe('title of known-note')
    expect(store.asking('known-note')).toBe(false)
    h.parts.notes.stale = (() => ({ disk: 'new' })) as unknown as () => null
    expect(store.asking('known-note')).toBe(true)

    const flushResult = Promise.resolve()
    ;(h.parts.notes.flushPending as ReturnType<typeof vi.fn>).mockReturnValue(flushResult)
    expect(store.flushPending('known-note')).toBe(flushResult)
    expect(h.parts.notes.flushPending).toHaveBeenCalledWith('known-note')

    ;(h.parts.notes.close as ReturnType<typeof vi.fn>).mockReturnValue(
      Promise.reject(new Error('disk locked')),
    )
    store.close('known-note')
    expect(h.parts.notes.close).toHaveBeenCalledWith('known-note')
    await new Promise((r) => setTimeout(r, 10))
    expect(writers.editing).toHaveBeenCalledWith(
      'A note did not close: Numen is not responding; retrying',
      'error',
    )
  })
})
