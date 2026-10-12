/** The changes a note is drawn with, and the intervals they are let go of on. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createNoteChanges } from './changes'
import type { NoteEdit } from '@/entities/note'

const limits = { noteChangedMs: 10, endedMs: 40, unendedMs: 100 }

/** One change reported on a note. */
const createEdit = (over: Partial<NoteEdit> = {}): NoteEdit => ({
  change: 'c1',
  path: 'Entropy.md',
  span: { from: 0, to: 5 },
  text: 'Order',
  isComplete: false,
  ...over,
})

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('a change reported on a note', () => {
  it('is what the note is drawn with', () => {
    const changes = createNoteChanges(limits)

    changes.reportChange(createEdit())

    expect(changes.getChange('Entropy.md')).toEqual({ id: 'c1', from: 0, to: 5, text: 'Order' })
  })

  it('is nothing on a note nobody is drawing', () => {
    expect(createNoteChanges(limits).getChange('Entropy.md')).toBeNull()
  })

  it('is let go of where nothing more is said about it', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())

    vi.advanceTimersByTime(limits.unendedMs)

    expect(changes.getChange('Entropy.md')).toBeNull()
  })

  it('waits again from the last word about it', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())

    vi.advanceTimersByTime(limits.unendedMs - 1)
    changes.reportChange(createEdit({ text: 'Order and disorder' }))
    vi.advanceTimersByTime(limits.unendedMs - 1)

    expect(changes.getChange('Entropy.md')?.text).toBe('Order and disorder')
  })
})

describe('a change that has ended', () => {
  it('stays until the note changes under it', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())
    changes.reportChange(createEdit({ isComplete: true }))

    changes.reportNoteChange('Entropy.md')
    expect(changes.getChange('Entropy.md')).not.toBeNull()

    vi.advanceTimersByTime(limits.noteChangedMs)
    expect(changes.getChange('Entropy.md')).toBeNull()
  })

  it('is let go of on the longer endedMs where the text never arrives', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())
    changes.reportChange(createEdit({ isComplete: true }))

    vi.advanceTimersByTime(limits.endedMs)

    expect(changes.getChange('Entropy.md')).toBeNull()
  })
})

describe('a note the window is no longer showing', () => {
  it('is drawn with nothing, and waits on nothing', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())

    changes.closeNote('Entropy.md')

    expect(changes.getChange('Entropy.md')).toBeNull()
    expect(vi.getTimerCount()).toBe(0)
  })
})

describe('a window that is going', () => {
  it('lets go of every interval it is waiting on', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())
    changes.reportChange(createEdit({ path: 'Order.md' }))

    changes.close()

    expect(vi.getTimerCount()).toBe(0)
  })
})

describe('an interval that has been let go of', () => {
  /** Every handle passed to clearTimeout from now on. */
  const watchClears = () => vi.spyOn(globalThis, 'clearTimeout')

  it('is forgotten once it fired', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())
    vi.advanceTimersByTime(limits.unendedMs)
    const cleared = watchClears()

    changes.closeNote('Entropy.md')

    expect(cleared).toHaveBeenCalledExactlyOnceWith(undefined)
  })

  it('is forgotten once the note was closed', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())
    changes.closeNote('Entropy.md')
    const cleared = watchClears()

    changes.closeNote('Entropy.md')

    expect(cleared).toHaveBeenCalledExactlyOnceWith(undefined)
  })

  it('is forgotten once the window went', () => {
    const changes = createNoteChanges(limits)
    changes.reportChange(createEdit())
    changes.close()
    const cleared = watchClears()

    changes.closeNote('Entropy.md')

    expect(cleared).toHaveBeenCalledExactlyOnceWith(undefined)
  })
})
