/**
 * The lines of a book's contents.
 *
 * Finding the line that holds an offset is the decision under test.
 */
import { describe, expect, it } from 'vitest'
import { findLineAt } from './contents'
import type { ContentsEntry } from '../../../types'

/** A book naming four places, in the order its text runs. */
const PARTS: readonly ContentsEntry[] = [
  { title: 'Ādi Parva', at: 0, level: 0 },
  { title: 'Sabhā Parva', at: 4_000, level: 0 },
  { title: 'Сказание о сожжении леса', at: 4_800, level: 1 },
  { title: 'Vana Parva', at: 9_000, level: 0 },
]

describe('the line an offset falls on', () => {
  it('is the last one beginning at or before it', () => {
    expect(findLineAt(PARTS, 4_799)).toBe(1)
    expect(findLineAt(PARTS, 4_800)).toBe(2)
    expect(findLineAt(PARTS, 100_000)).toBe(3)
  })

  it('is none where the offset precedes every name', () => {
    // A book whose first name starts after its opening pages leaves those pages
    // under no name, and the panel marks nothing.
    expect(findLineAt([{ title: 'Chapter One', at: 500, level: 0 }], 12)).toBe(-1)
  })

  it('is none where the book is empty', () => {
    expect(findLineAt([], 0)).toBe(-1)
  })
})
