/**
 * The marks a book puts into the browser's own highlight registry, and what it
 * takes back out.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { HIGHLIGHT, OTHER_HIGHLIGHT, highlight, unhighlight } from './highlight'

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
const books = [first, second, moving, kept, going, staying, only]

beforeEach(() => {
  registry()
})

// The module keeps what each book put in it; every book is taken back out so
// the next test reads a registry nothing was put into before it.
afterEach(() => {
  for (const book of books) unhighlight(book)
})

describe('the entries the names name', () => {
  it('are the two a book is drawn with', () => {
    expect(HIGHLIGHT).toBe('numen-book')
    expect(OTHER_HIGHLIGHT).toBe('numen-book-other-highlight')
  })
})

describe('the runs of one book in one entry', () => {
  it('stand in the registry under the entry name', () => {
    const held = registry()

    highlight(HIGHLIGHT, first, [range()])

    expect(held.get(HIGHLIGHT)).toBeInstanceOf(Highlight)
  })

  it('are the ranges of each book alongside the ranges of the others', () => {
    const held = registry()
    const one = range()
    const other = range()

    highlight(HIGHLIGHT, first, [one])
    highlight(HIGHLIGHT, second, [other])

    const drawn = held.get(HIGHLIGHT) as unknown as { ranges: readonly unknown[] }
    expect(drawn.ranges).toStrictEqual([one, other])
  })

  it('replace what the same book had there, and leave the rest', () => {
    const held = registry()
    const standing = range()
    const freshRange = range()
    highlight(HIGHLIGHT, kept, [standing])
    highlight(HIGHLIGHT, moving, [range()])

    highlight(HIGHLIGHT, moving, [freshRange])

    const drawn = held.get(HIGHLIGHT) as unknown as { ranges: readonly unknown[] }
    expect(drawn.ranges).toStrictEqual([standing, freshRange])
  })

  it('are taken out of the registry where no book holds any', () => {
    const held = registry()

    highlight(HIGHLIGHT, only, [range()])
    highlight(HIGHLIGHT, only, [])

    expect(held.has(HIGHLIGHT)).toBe(false)
  })
})

describe('what a book took with it when it went', () => {
  it('is taken back out of every entry, and the rest is left standing', () => {
    const held = registry()
    highlight(HIGHLIGHT, going, [range()])
    highlight(OTHER_HIGHLIGHT, going, [range()])
    highlight(HIGHLIGHT, staying, [range()])

    unhighlight(going)

    const drawn = held.get(HIGHLIGHT) as unknown as { ranges: readonly unknown[] }
    expect(drawn.ranges).toHaveLength(1)
    expect(held.has(OTHER_HIGHLIGHT)).toBe(false)
  })

  it('is nothing, where the book had put nothing anywhere', () => {
    expect(() => unhighlight({ book: 'never was' })).not.toThrow()
  })
})

describe('a browser that cannot draw a range', () => {
  it('is left alone where it has no CSS at all', () => {
    Reflect.deleteProperty(globalThis, 'CSS')

    expect(() => highlight(HIGHLIGHT, only, [range()])).not.toThrow()
  })

  it('is left alone where its CSS has no highlight registry', () => {
    registry()
    globalThis.CSS = {} as unknown as typeof CSS

    expect(() => highlight(HIGHLIGHT, only, [range()])).not.toThrow()
  })

  it('is left alone where it has no Highlight to make', () => {
    const held = registry()
    Reflect.deleteProperty(globalThis, 'Highlight')

    expect(() => highlight(HIGHLIGHT, only, [range()])).not.toThrow()
    expect(held.size).toBe(0)
  })
})
