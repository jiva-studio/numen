/** The rows of the headings of a note: their words, their lines and how far in each is set. */
import { describe, expect, it } from 'vitest'
import { getRows } from './rows'

const heading = (text: string, level: number, line: number) => ({ text, level, line })

describe('the rows of a list', () => {
  it('are the headings in the order they were given, each with its words and its line', () => {
    expect(
      getRows([heading('First', 1, 3), heading('Second', 1, 9)], 10).map(({ text, line }) => [text, line]),
    ).toStrictEqual([
      ['First', 3],
      ['Second', 9],
    ])
  })

  it('set each level in by its rank among the levels the note uses', () => {
    expect(
      getRows([heading('A', 1, 0), heading('B', 2, 1), heading('C', 3, 2)], 10).map((row) => row.indent),
    ).toStrictEqual([0, 10, 20])
  })

  it('set a note that starts at the second level as one that starts at the first', () => {
    expect(getRows([heading('A', 2, 0), heading('B', 3, 1)], 10).map((row) => row.indent)).toStrictEqual([0, 10])
  })

  it('set no heading in further than three steps', () => {
    const levels = [1, 2, 3, 4, 5, 6].map((level, index) => heading(`H${level}`, level, index))

    expect(getRows(levels, 10).map((row) => row.indent)).toStrictEqual([0, 10, 20, 30, 30, 30])
  })

  it('are none for a note with no heading', () => {
    expect(getRows([], 10)).toStrictEqual([])
  })
})
