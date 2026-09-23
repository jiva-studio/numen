/**
 * The marks a book puts into the browser's own highlight registry, and what it
 * takes back out.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { HIGHLIGHT, OTHER_HIGHLIGHT } from './highlight'

/** The registry a browser that draws ranges keeps, and the ranges it holds. */
const registry = () => {
  const held = new Map<string, Set<unknown>>()
  globalThis.CSS = { highlights: held } as unknown as typeof CSS
  globalThis.Highlight = class {
    readonly ranges: readonly unknown[]
    constructor(...ranges: unknown[]) {
      this.ranges = ranges
    }
  } as unknown as typeof Highlight
  return held
}

/** A range over nothing in particular, which is all the registry asks for. */
const range = (): Range => document.createRange()

/** The books a test holds, each known to the registry by being itself. */
const first = { book: 'first' }
const second = { book: 'second' }
const moving = { book: 'moving' }
const kept = { book: 'kept' }
const going = { book: 'going' }
const staying = { book: 'staying' }
const only = { book: 'only' }

// The ranges a book has put where stand between the asks of one test and the
// next, so each test reads a registry nothing was put into before it.
beforeEach(() => {
  registry()
  vi.resetModules()
})

describe('the entries the names name', () => {
  it('are the two a book is drawn with', () => {
    expect(HIGHLIGHT).toBe('numen-book')
    expect(OTHER_HIGHLIGHT).toBe('numen-book-other-highlight')
  })
})

describe('the runs of one book in one entry', () => {
  it('stand in the registry under the entry name', async () => {
    const { highlight: fresh } = await import('./highlight')
    const held = registry()

    fresh(HIGHLIGHT, first, [range()])

    expect(held.get(HIGHLIGHT)).toBeInstanceOf(Highlight)
  })

  it('are the ranges of each book beside the ranges of the others', async () => {
    const { highlight: fresh } = await import('./highlight')
    const held = registry()
    const one = range()
    const other = range()

    fresh(HIGHLIGHT, first, [one])
    fresh(HIGHLIGHT, second, [other])

    const drawn = held.get(HIGHLIGHT) as unknown as { ranges: readonly unknown[] }
    expect(drawn.ranges).toStrictEqual([one, other])
  })

  it('replace what the same book had there, and leave the rest', async () => {
    const { highlight: fresh } = await import('./highlight')
    const held = registry()
    const standing = range()
    const freshRange = range()
    fresh(HIGHLIGHT, kept, [standing])
    fresh(HIGHLIGHT, moving, [range()])

    fresh(HIGHLIGHT, moving, [freshRange])

    const drawn = held.get(HIGHLIGHT) as unknown as { ranges: readonly unknown[] }
    expect(drawn.ranges).toStrictEqual([standing, freshRange])
  })

  it('are taken out of the registry where no book holds any', async () => {
    const { highlight: fresh } = await import('./highlight')
    const held = registry()

    fresh(HIGHLIGHT, only, [range()])
    fresh(HIGHLIGHT, only, [])

    expect(held.has(HIGHLIGHT)).toBe(false)
  })
})

describe('what a book took with it when it went', () => {
  it('is taken back out of every entry, and the rest is left standing', async () => {
    const { highlight: fresh, unhighlight: taken } = await import('./highlight')
    const held = registry()
    fresh(HIGHLIGHT, going, [range()])
    fresh(OTHER_HIGHLIGHT, going, [range()])
    fresh(HIGHLIGHT, staying, [range()])

    taken(going)

    const drawn = held.get(HIGHLIGHT) as unknown as { ranges: readonly unknown[] }
    expect(drawn.ranges).toHaveLength(1)
    expect(held.has(OTHER_HIGHLIGHT)).toBe(false)
  })

  it('is nothing, where the book had put nothing anywhere', async () => {
    const { unhighlight: taken } = await import('./highlight')
    registry()

    expect(() => taken({ book: 'never was' })).not.toThrow()
  })
})
