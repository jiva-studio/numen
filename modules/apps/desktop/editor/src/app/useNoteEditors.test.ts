/**
 * The editors of a window: what each is made with, what is flushed, what raises conflicts.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNoteEditors } from './useNoteEditors'

const h = vi.hoisted(() => ({
  calls: {} as Record<string, any[]>,
  parts: {} as Record<string, any>,
}))

vi.mock('@/entities/deck', () => ({ cards: { name: 'cards' }, presets: { name: 'presets' } }))
vi.mock('@/entities/note', () => ({
  openNotes: (...args: unknown[]) => {
    h.calls.openNotes = args
    return h.parts.notes
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
    h.calls.stencils = args
    return h.parts.stencils
  },
}))
vi.mock('@/pages/preset-editor', () => ({
  usePresetTab: (...args: unknown[]) => {
    h.calls.schedules = args
    return h.parts.schedules
  },
}))
vi.mock('@/pages/note-editor', () => ({
  createNoteChanges: () => h.parts.changes,
  createNoteWriter: (...args: unknown[]) => {
    h.calls.writer = args
    return h.parts.making
  },
  useNoteTab: (...args: unknown[]) => {
    h.calls.noted = args
    return h.parts.noted
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

const store = (title: string | undefined) => ({
  has: (id: string) => title !== undefined && id === 'known-' + title,
  getTitle: (id: string) => 'title of ' + id,
})

beforeEach(() => {
  h.calls = { raised: [] }
  const flush = (name: string) => ({ name: `${name}.flush` })
  h.parts = {
    changes: { handleNoteChange: vi.fn(), close: vi.fn() },
    notes: { flush: flush('notes') },
    making: { name: 'making' },
    noted: { kept: store('note'), kind: 'noteKind' },
    decks: { flush: flush('decks'), kept: store('deck'), kind: 'deckKind' },
    stencils: { flush: flush('stencils'), kept: store('stencil'), kind: 'stencilKind' },
    schedules: { flush: flush('schedules'), kind: 'scheduleKind' },
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
  const held: any = { handle: { name: 'handle' } }
  const day = () => 'monday'
  const editors = useNoteEditors({ core, log, tabOpeners, held, day })
  return { editors, writers, core, tabOpeners, held, day }
}

describe('the note editors', () => {
  it('opens the notes with the change handler for a replaced note', () => {
    const { core } = create()

    expect(h.calls.openNotes[0]).toBe(core)
    expect(h.calls.openNotes[1]).toEqual({ onReplaced: h.parts.changes.handleNoteChange })
    expect(h.calls.openNotes[1].onReplaced).toBe(h.parts.changes.handleNoteChange)
  })

  it('writes each editor message through the writer of its own name', () => {
    const { writers, day } = create()

    expect(h.calls.writer[1]).toBe(writers.made)
    expect(h.calls.stencils[3]).toBe(writers.stencil)
    expect(h.calls.schedules[3]).toBe(writers.preset)
    expect(h.calls.schedules[4]).toBe(day)
    expect(new Set(Object.keys(writers))).toEqual(new Set(['made', 'stencil', 'preset']))
  })

  it('flushes the notes, decks, stencils and schedules', () => {
    create()

    expect(h.parts.fileFlush.addHandler.mock.calls.map((c: any[]) => c[0])).toEqual([
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

    expect(editors.stores).toEqual([h.parts.noted.kept, h.parts.decks.kept, h.parts.stencils.kept])
    expect(h.calls.reached[0]).toBe(editors.stores)
    expect(h.calls.reached[1]).toBe(tabOpeners)
    expect(editors.reached).toBe(h.parts.reached)
  })

  it('lists the kinds of the note, deck, stencil and schedule editors', () => {
    const { editors } = create()

    expect(editors.kinds).toEqual(['noteKind', 'deckKind', 'stencilKind', 'scheduleKind'])
  })

  it('gets a title from the first store that has the id, and an empty one from none', () => {
    const { editors } = create()

    expect(editors.getTitle('known-deck')).toBe('title of known-deck')
    expect(editors.getTitle('unknown')).toBe('')
  })

  it('closes the changes and the flush', () => {
    const { editors } = create()

    editors.close()

    expect(h.parts.changes.close).toHaveBeenCalledTimes(1)
    expect(h.parts.fileFlush.close).toHaveBeenCalledTimes(1)
  })
})
