/**
 * How a preset writes the share of a day's load each day of the week carries.
 *
 * The whole of a day is what a day nothing was said about carries, so the file
 * names only the days standing under it. That is the file's own way of writing
 * it, and nothing that draws a week knows of it.
 */
import { describe, expect, it } from 'vitest'
import { loadOn, loaded, LOADS, WHOLE_LOAD } from './core'

describe('what one day carries', () => {
  it('is the whole of a day for a day nothing was said about', () => {
    expect(loadOn({}, 'mon')).toBe(WHOLE_LOAD)
    expect(loadOn({ sat: 50 }, 'mon')).toBe(WHOLE_LOAD)
    expect(loadOn({ sat: 50 }, 'sat')).toBe(50)
  })

  it('is nothing where a day is put at nothing, which is not the whole of it', () => {
    expect(loadOn({ sun: 0 }, 'sun')).toBe(0)
  })
})

describe('a day put at a share', () => {
  it('names a day standing under the whole, and drops one put back to it', () => {
    expect(loaded({}, 'sat', 50)).toEqual({ sat: 50 })
    expect(loaded({ sat: 50 }, 'sat', 0)).toEqual({ sat: 0 })
    expect(loaded({ sat: 50, sun: 0 }, 'sat', WHOLE_LOAD)).toEqual({ sun: 0 })
    expect(loaded({}, 'sat', WHOLE_LOAD)).toEqual({})
  })

  it('leaves every other day where it stood', () => {
    expect(loaded({ sat: 50, sun: 0 }, 'mon', 25)).toEqual({ sat: 50, sun: 0, mon: 25 })
  })
})

describe('the shares a day is offered', () => {
  it('run from nothing to the whole of a day, in order', () => {
    expect(LOADS[0]).toBe(0)
    expect(LOADS.at(-1)).toBe(WHOLE_LOAD)
    expect([...LOADS].sort((one, two) => one - two)).toEqual([...LOADS])
  })
})
