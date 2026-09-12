/**
 * The lines of a book's contents.
 *
 * Which line holds an offset and which a typed word finds are decisions, and
 * they are the ones asked about here.
 */
import { describe, expect, it } from 'vitest'
import { matching, standingIn, type ContentsEntry } from './contents'

/** A book naming four places, in the order its text runs. */
const PARTS: readonly ContentsEntry[] = [
  { title: 'Ādi Parva', at: 0, level: 0 },
  { title: 'Sabhā Parva', at: 4_000, level: 0 },
  { title: 'Сказание о сожжении леса', at: 4_800, level: 1 },
  { title: 'Vana Parva', at: 9_000, level: 0 },
]

describe('the line an offset falls on', () => {
  it('is the last one beginning at or before it', () => {
    expect(standingIn(PARTS, 4_799)).toBe(1)
    expect(standingIn(PARTS, 4_800)).toBe(2)
    expect(standingIn(PARTS, 100_000)).toBe(3)
  })

  it('is none where the offset precedes every name', () => {
    // A book whose first name stands past its opening pages leaves them under
    // no name at all, and the panel marks nothing.
    expect(standingIn([{ title: 'Chapter One', at: 500, level: 0 }], 12)).toBe(-1)
  })

  it('is none where the book names nothing', () => {
    expect(standingIn([], 0)).toBe(-1)
  })
})

describe('the lines a typed word finds', () => {
  it('is every line where nothing is typed', () => {
    expect(matching(PARTS, '   ')).toStrictEqual(PARTS)
  })

  it('finds a name however it was typed', () => {
    expect(matching(PARTS, 'parva').map((one) => one.title)).toStrictEqual([
      'Ādi Parva',
      'Sabhā Parva',
      'Vana Parva',
    ])
  })

  it('finds a name written in another script', () => {
    expect(matching(PARTS, 'СОЖЖЕНИИ')).toHaveLength(1)
  })

  it('finds nothing where nothing is named that', () => {
    expect(matching(PARTS, 'Bhagavad')).toStrictEqual([])
  })
})
