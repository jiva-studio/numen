/**
 * What each note is drawn with, asked without a browser.
 *
 * A change drawn after the note stopped changing is an overlay a person cannot
 * get rid of, and one dropped too early is a change that happens invisibly.
 */
import { describe, expect, it } from 'vitest'
import type { NoteEdit } from '@/entities/note'
import { createChangeTracker, CHANGE_LIMITS } from './changeTracker'

const createEdit = (over: Partial<NoteEdit> = {}): NoteEdit => ({
  change: 'one',
  path: 'Note.md',
  span: { from: 2, to: 12 },
  text: 'An axe',
  isComplete: false,
  ...over,
})

describe('a change being made', () => {
  it('is what its note is drawn with', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    expect(drawn.getChange('Note.md')).toEqual({ id: 'one', from: 2, to: 12, text: 'An axe' })
  })

  it('is drawn over its own note and no other', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    expect(drawn.getChange('Other.md')).toBeNull()
  })

  it('is replaced by the next report of itself', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    drawn.reportChange(createEdit({ text: 'An axe, two-bladed' }))
    expect(drawn.getChange('Note.md')?.text).toBe('An axe, two-bladed')
  })
})

describe('a change that is over', () => {
  it('is still drawn, because the note has not caught up', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    drawn.reportChange(createEdit({ isComplete: true }))
    expect(drawn.getChange('Note.md')).not.toBeNull()
  })

  it('waits the endedMs out where its text never arrives', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    expect(drawn.reportChange(createEdit({ isComplete: true }))).toEqual({
      path: 'Note.md',
      after: CHANGE_LIMITS.endedMs,
    })
  })

  it('is let go of shortly after the note changes under it', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    drawn.reportChange(createEdit({ isComplete: true }))
    expect(drawn.reportNoteChange('Note.md')).toEqual({
      path: 'Note.md',
      after: CHANGE_LIMITS.noteChangedMs,
    })
  })

  it('is gone once the interval fires', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    drawn.reportChange(createEdit({ isComplete: true }))
    drawn.expireChange('Note.md')
    expect(drawn.getChange('Note.md')).toBeNull()
  })
})

describe('a note that changed on its own', () => {
  it('arms nothing, because nothing is drawn over it', () => {
    const drawn = createChangeTracker()
    expect(drawn.reportNoteChange('Note.md')).toBeNull()
  })

  it('arms nothing while the change it is drawn with is still being made', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    expect(drawn.reportNoteChange('Note.md')).toBeNull()
  })
})

describe('the end of a change nobody is drawing', () => {
  it('arms nothing', () => {
    const drawn = createChangeTracker()
    expect(drawn.reportChange(createEdit({ isComplete: true }))).toBeNull()
  })
})

describe('a note the window closed', () => {
  it('is drawn with nothing', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    drawn.closeNote('Note.md')
    expect(drawn.getChange('Note.md')).toBeNull()
  })
})

describe('a change nobody says any more about', () => {
  it('is let go of on a endedMs of its own, so no drawing outlives its agent', () => {
    const drawn = createChangeTracker()
    expect(drawn.reportChange(createEdit())).toEqual({
      path: 'Note.md',
      after: CHANGE_LIMITS.unendedMs,
    })
  })

  it('has that endedMs put off again by every report of itself', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    expect(drawn.reportChange(createEdit({ text: 'An axe, two-bladed' }))).toEqual({
      path: 'Note.md',
      after: CHANGE_LIMITS.unendedMs,
    })
  })
})

describe('a change reported again after it ended', () => {
  it('is being made again, so the note changing under it arms nothing', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    drawn.reportChange(createEdit({ isComplete: true }))
    drawn.reportChange(createEdit({ text: 'An axe, two-bladed' }))

    expect(drawn.reportNoteChange('Note.md')).toBeNull()
  })
})

describe('a note whose change is gone', () => {
  it('arms nothing when the note changes after the interval fired', () => {
    const drawn = createChangeTracker()
    drawn.reportChange(createEdit())
    drawn.reportChange(createEdit({ isComplete: true }))
    drawn.expireChange('Note.md')

    expect(drawn.reportNoteChange('Note.md')).toBeNull()
  })
})
