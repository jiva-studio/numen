/** The changes a note is drawn with, and the intervals they are let go of on. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { noteChanges } from './changes'
import type { NoteEdit } from '@/entities/note'

const limits = { settle: 10, bound: 40, abandoned: 100 }

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
    const changes = noteChanges(limits)

    changes.reportChange(createEdit())

    expect(changes.getChange('Entropy.md')).toEqual({ id: 'c1', from: 0, to: 5, text: 'Order' })
  })

  it('is nothing on a note nobody is drawing', () => {
    expect(noteChanges(limits).getChange('Entropy.md')).toBeNull()
  })

  it('is let go of where nothing more is said about it', () => {
    const changes = noteChanges(limits)
    changes.reportChange(createEdit())

    vi.advanceTimersByTime(limits.abandoned)

    expect(changes.getChange('Entropy.md')).toBeNull()
  })

  it('waits again from the last word about it', () => {
    const changes = noteChanges(limits)
    changes.reportChange(createEdit())

    vi.advanceTimersByTime(limits.abandoned - 1)
    changes.reportChange(createEdit({ text: 'Order and disorder' }))
    vi.advanceTimersByTime(limits.abandoned - 1)

    expect(changes.getChange('Entropy.md')?.text).toBe('Order and disorder')
  })
})

describe('a change that has ended', () => {
  it('stays until the note changes under it', () => {
    const changes = noteChanges(limits)
    changes.reportChange(createEdit())
    changes.reportChange(createEdit({ isComplete: true }))

    changes.handleNoteChange('Entropy.md')
    expect(changes.getChange('Entropy.md')).not.toBeNull()

    vi.advanceTimersByTime(limits.settle)
    expect(changes.getChange('Entropy.md')).toBeNull()
  })

  it('is let go of on the longer bound where the text never arrives', () => {
    const changes = noteChanges(limits)
    changes.reportChange(createEdit())
    changes.reportChange(createEdit({ isComplete: true }))

    vi.advanceTimersByTime(limits.bound)

    expect(changes.getChange('Entropy.md')).toBeNull()
  })
})

describe('a note the window is no longer showing', () => {
  it('is drawn with nothing, and waits on nothing', () => {
    const changes = noteChanges(limits)
    changes.reportChange(createEdit())

    changes.closeNote('Entropy.md')

    expect(changes.getChange('Entropy.md')).toBeNull()
    expect(vi.getTimerCount()).toBe(0)
  })
})

describe('a window that is going', () => {
  it('lets go of every interval it is waiting on', () => {
    const changes = noteChanges(limits)
    changes.reportChange(createEdit())
    changes.reportChange(createEdit({ path: 'Order.md' }))

    changes.close()

    expect(vi.getTimerCount()).toBe(0)
  })
})
