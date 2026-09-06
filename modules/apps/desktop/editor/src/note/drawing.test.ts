/**
 * What each note is drawn with, asked without a browser.
 *
 * A change drawn after the note stopped changing is an overlay a person cannot
 * get rid of, and one dropped too early is a change that happens invisibly.
 */
import { describe, expect, it } from 'vitest'
import type { NoteEdit } from '../core'
import { drawing, holding } from './drawing'

const said = (over: Partial<NoteEdit> = {}): NoteEdit => ({
  change: 'one',
  path: 'Note.md',
  from: 2,
  to: 12,
  text: 'An axe',
  done: false,
  ...over,
})

describe('a change being made', () => {
  it('is what its note is drawn with', () => {
    const drawn = drawing()
    drawn.told(said())
    expect(drawn.shown('Note.md')).toEqual({ id: 'one', from: 2, to: 12, text: 'An axe' })
  })

  it('is drawn over its own note and no other', () => {
    const drawn = drawing()
    drawn.told(said())
    expect(drawn.shown('Other.md')).toBeNull()
  })

  it('is replaced by the next report of itself', () => {
    const drawn = drawing()
    drawn.told(said())
    drawn.told(said({ text: 'An axe, two-bladed' }))
    expect(drawn.shown('Note.md')?.text).toBe('An axe, two-bladed')
  })
})

describe('a change that is over', () => {
  it('is still drawn, because the note has not caught up', () => {
    const drawn = drawing()
    drawn.told(said())
    drawn.told(said({ done: true }))
    expect(drawn.shown('Note.md')).not.toBeNull()
  })

  it('waits the bound out where its text never arrives', () => {
    const drawn = drawing()
    drawn.told(said())
    expect(drawn.told(said({ done: true }))).toEqual({
      path: 'Note.md',
      after: holding.bound,
    })
  })

  it('is let go of shortly after the note changes under it', () => {
    const drawn = drawing()
    drawn.told(said())
    drawn.told(said({ done: true }))
    expect(drawn.arrived('Note.md')).toEqual({ path: 'Note.md', after: holding.settle })
  })

  it('is gone once the interval fires', () => {
    const drawn = drawing()
    drawn.told(said())
    drawn.told(said({ done: true }))
    drawn.fired('Note.md')
    expect(drawn.shown('Note.md')).toBeNull()
  })
})

describe('a note that changed on its own', () => {
  it('arms nothing, because nothing is drawn over it', () => {
    const drawn = drawing()
    expect(drawn.arrived('Note.md')).toBeNull()
  })

  it('arms nothing while the change it is drawn with is still being made', () => {
    const drawn = drawing()
    drawn.told(said())
    expect(drawn.arrived('Note.md')).toBeNull()
  })
})

describe('the end of a change nobody is drawing', () => {
  it('arms nothing', () => {
    const drawn = drawing()
    expect(drawn.told(said({ done: true }))).toBeNull()
  })
})

describe('a note the window closed', () => {
  it('is drawn with nothing', () => {
    const drawn = drawing()
    drawn.told(said())
    drawn.shut('Note.md')
    expect(drawn.shown('Note.md')).toBeNull()
  })
})

describe('a change nobody says any more about', () => {
  it('is let go of on a bound of its own, so no drawing outlives its agent', () => {
    const drawn = drawing()
    expect(drawn.told(said())).toEqual({ path: 'Note.md', after: holding.abandoned })
  })

  it('has that bound put off again by every report of itself', () => {
    const drawn = drawing()
    drawn.told(said())
    expect(drawn.told(said({ text: 'An axe, two-bladed' }))).toEqual({
      path: 'Note.md',
      after: holding.abandoned,
    })
  })
})
