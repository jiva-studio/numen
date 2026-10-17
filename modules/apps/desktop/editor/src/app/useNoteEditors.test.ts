/**
 * The editors of a window: what each is made with, what is flushed, what raises conflicts.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNoteEditors } from './useNoteEditors'

interface MockParts {
  changes?: { reportNoteChange: unknown; close: unknown }
  notes: { flush: unknown; has: (id: string) => boolean; getPath: (id: string) => string; stale: () => null; flushPending: unknown; close: unknown }
  making: { name: string }
  decks: { flush: unknown; kept: unknown; kind: string }
  stencils: { flush: unknown; kept: unknown; kind: string }
  schedules: { flush: unknown; kind: string }
  fileFlush: { addHandler: ReturnType<typeof vi.fn>; close: ReturnType<typeof vi.fn> }
  reached: { name: string }
}

interface MockCalls {
  raised: unknown[][]
  openNotes: [unknown, { onReplaced: () => void }]
  writer: [unknown, unknown]
  decks: unknown[]
  stencils: [unknown, unknown, { writeMessage: unknown }]
  schedules: [{ writeMessage: unknown; today: () => string }]
  flush: unknown[]
  reached: [unknown, unknown]
}

const h = vi.hoisted(() => ({
  calls: {} as MockCalls,
  parts: {} as MockParts,
}))

vi.mock('@/entities/deck', () => ({ cards: { name: 'cards' }, presets: { name: 'presets' } }))
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
vi.mock('@/pages/deck-editor', () => ({
  useDeckTabs: (...args: unknown[]) => {
    h.calls.decks = args
    return h.parts.decks
  },
}))
vi.mock('@/pages/stencil-editor', () => ({
  useStencilTabs: (...args: unknown[]) => {
    h.calls.stencils = args as MockCalls['stencils']
    return h.parts.stencils
  },
}))
vi.mock('@/pages/preset-editor', () => ({
  usePresetTab: (...args: unknown[]) => {
    h.calls.schedules = args as MockCalls['schedules']
    return h.parts.schedules
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
    h.calls.reached = args as MockCalls['reached']
    return h.parts.reached
  },
}))

const store = (title: string | undefined) => ({
  has: (id: string) => title !== undefined && id === 'known-' + title,
  getTitle: (id: string) => 'title of ' + id,
})

beforeEach(() => {
  h.calls = {
    raised: [],
    openNotes: [null, { onReplaced: () => {} }],
    writer: [null, null],
    decks: [],
    stencils: [null, null, { writeMessage: null }],
    schedules: [{ writeMessage: null, today: () => '' }],
    flush: [],
    reached: [null, null],
  }
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
    decks: { flush: flush('decks'), kept: store('deck'), kind: 'deckKind' },
    stencils: { flush: flush('stencils'), kept: store('stencil'), kind: 'stencilKind' },
    schedules: { flush: flush('schedules'), kind: 'scheduleKind' },
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
  const held = { handle: { name: 'handle' } }
  const day = () => 'monday'
  const editors = useNoteEditors({ core, log, tabOpeners, held, day } as unknown as Parameters<typeof useNoteEditors>[0])
  return { editors, writers, core, tabOpeners, held, day }
}

describe('the note editors', () => {
  it('opens the notes with the change handler for a replaced note', () => {
    const { core } = create()

    expect(h.calls.openNotes[0]).toBe(core)
    expect(typeof h.calls.openNotes[1].onReplaced).toBe('function')
  })

  it('writes each editor message through the writer of its own name', () => {
    const { writers, day } = create()

    expect(h.calls.writer[1]).toBe(writers.made)
    expect(h.calls.stencils[2].writeMessage).toBe(writers.stencil)
    expect(h.calls.schedules[0].writeMessage).toBe(writers.preset)
    expect(h.calls.schedules[0].today).toBe(day)
    expect(new Set(Object.keys(writers))).toEqual(new Set(['made', 'stencil', 'preset']))
  })

  it('flushes the notes, decks, stencils and schedules', () => {
    create()

    expect(h.parts.fileFlush.addHandler.mock.calls.map((c: unknown[]) => c[0])).toEqual([
      h.parts.notes.flush,
      h.parts.decks.flush,
      h.parts.stencils.flush,
      h.parts.schedules.flush,
    ])
  })

  it('raises conflicts for the notes, decks and stencils, in the flush', () => {
    create()

    expect(h.calls.raised).toEqual([
      [h.parts.notes, h.parts.fileFlush],
      [h.parts.decks, h.parts.fileFlush],
      [h.parts.stencils, h.parts.fileFlush],
    ])
  })

  it('keeps the stores of notes, decks and stencils and reaches notes through them', () => {
    const { editors, tabOpeners } = create()

    expect(editors.stores.length).toBe(3)
    expect(editors.stores[1]).toBe(h.parts.decks.kept)
    expect(editors.stores[2]).toBe(h.parts.stencils.kept)
    expect(h.calls.reached[0]).toBe(editors.stores)
    expect(h.calls.reached[1]).toBe(tabOpeners)
    expect(editors.reached).toBe(h.parts.reached)
  })

  it('lists the kinds of the deck, stencil and schedule editors', () => {
    const { editors } = create()

    expect(editors.kinds).toEqual(['deckKind', 'stencilKind', 'scheduleKind'])
  })

  it('gets a title from the first store that has the id, and an empty one from none', () => {
    const { editors } = create()

    expect(editors.getTitle('known-deck')).toBe('title of known-deck')
    expect(editors.getTitle('known-note')).toBe('title of known-note')
    expect(editors.getTitle('unknown')).toBe('')
  })

  it('closes the changes and the flush', () => {
    const { editors } = create()

    editors.close()

    expect(h.parts.fileFlush.close).toHaveBeenCalledTimes(1)
  })
})
