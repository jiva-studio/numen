/**
 * The editors of a window: what each is made with, what is flushed, what raises conflicts.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNoteEditors } from './useNoteEditors'

const h = vi.hoisted(() => ({
  calls: {} as any,
  parts: {} as any,
}))

vi.mock('@/entities/note', () => ({
  openNotes: (...args: unknown[]) => {
    h.calls.openNotes = args
    return h.parts.notes
  },
  createNoteWriter: (...args: unknown[]) => {
    h.calls.writer = args
    return h.parts.making
  },
}))
vi.mock('@/features/file-conflict', () => ({
  useFileFlush: (...args: unknown[]) => {
    h.calls.flush = args
    return h.parts.fileFlush
  },
  raiseConflicts: (...args: unknown[]) => {
    h.calls.raised.push(args)
  },
}))
vi.mock('@/features/command-palette', () => ({
  createNotes: (...args: unknown[]) => {
    h.calls.reached = args
    return h.parts.reached
  },
}))

beforeEach(() => {
  h.calls = { raised: [] }
  const flush = (name: string) => ({ name: `${name}.flush` })
  h.parts = {
    changes: { reportNoteChange: vi.fn(), close: vi.fn() },
    notes: {
      flush: flush('notes'),
      has: (id: string) => id === 'known-note',
      getPath: (id: string) => 'title of ' + id,
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
  const writers: Record<string, any> = {}
  const log: any = {
    getWriter: vi.fn((name: string) => (writers[name] ??= vi.fn())),
  }
  const core: any = { name: 'core' }
  const tabOpeners: any = { name: 'openers' }
  const held: any = { handle: { name: 'handle' }, tabs: { value: [] }, getTab: vi.fn() }
  const day = () => 'monday'
  const editors = useNoteEditors({ core, log, tabOpeners, held, day })
  return { editors, writers, core, tabOpeners, held, day }
}

describe('the note editors', () => {
  it('opens the notes with the change handler for a replaced note', () => {
    const { core } = create()

    expect(h.calls.openNotes[0]).toBe(core)
    expect(typeof h.calls.openNotes[1].onReplaced).toBe('function')
  })

  it('writes each editor message through the writer of its own name', () => {
    const { writers } = create()

    expect(h.calls.writer[1]).toBe(writers.made)
    expect(new Set(Object.keys(writers))).toEqual(new Set(['made']))
  })

  it('flushes the notes', () => {
    create()

    expect(h.parts.fileFlush.addHandler.mock.calls.map((c: any[]) => c[0])).toEqual([
      h.parts.notes.flush,
    ])
  })

  it('raises conflicts for the notes in the flush', () => {
    create()

    expect(h.calls.raised).toEqual([
      [h.parts.notes, h.parts.fileFlush],
    ])
  })

  it('keeps the stores of notes and reaches notes through them', () => {
    const { editors, tabOpeners } = create()

    expect(editors.stores.length).toBe(1)
    expect(h.calls.reached[0]).toBe(editors.stores)
    expect(h.calls.reached[1]).toBe(tabOpeners)
    expect(editors.reached).toBe(h.parts.reached)
  })

  it('lists the kinds of the note editors', () => {
    const { editors } = create()

    expect(editors.kinds).toEqual([])
  })

  it('gets a title from the first store that has the id, and an empty one from none', () => {
    const { editors } = create()

    expect(editors.getTitle('known-note')).toBe('title of known-note')
    expect(editors.getTitle('unknown')).toBe('')
  })

  it('closes the changes and the flush', () => {
    const { editors } = create()

    editors.close()

    expect(h.parts.fileFlush.close).toHaveBeenCalledTimes(1)
  })
})
