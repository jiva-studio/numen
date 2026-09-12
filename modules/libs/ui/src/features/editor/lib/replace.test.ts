import { describe, expect, it } from 'vitest'
import { EditorSelection, EditorState, Transaction } from '@codemirror/state'
import { replace } from './replace'

const DOC = 'one\ntwo\nthree\nfour'

/** A person can put a caret in several places at once. */
const SEVERAL = EditorState.allowMultipleSelections.of(true)

/** Where every end of every range lands, as a line and a column. */
const after = (doc: string, selection: EditorSelection, fresh: string) => {
  const state = EditorState.create({ doc, selection, extensions: [SEVERAL] })
  const put = state.update(replace(state, fresh)).state
  const place = (at: number) => {
    const line = put.doc.lineAt(at)
    return [line.number, at - line.from]
  }
  return {
    text: put.doc.toString(),
    ranges: put.selection.ranges.map((range) => [place(range.anchor), place(range.head)]),
    main: put.selection.mainIndex,
  }
}

const caret = (doc: string, line: number, column: number) =>
  EditorSelection.single(EditorState.create({ doc }).doc.line(line).from + column)

/** Every stretch of text the change puts something else in the place of. */
const applyChange = (doc: string, fresh: string) => {
  const state = EditorState.create({ doc })
  const found: [number, number, string][] = []
  state
    .update(replace(state, fresh))
    .changes.iterChanges((from, to, _at, _to, insert) =>
      void found.push([from, to, insert.toString()]),
    )
  return found
}

describe('a document put in over another', () => {
  it('is the text it was given', () => {
    expect(after(DOC, caret(DOC, 1, 0), 'fresh\ntext').text).toBe('fresh\ntext')
  })

  it('is no step to undo', () => {
    const state = EditorState.create({ doc: DOC })
    expect(state.update(replace(state, 'fresh')).annotation(Transaction.addToHistory)).toBe(false)
  })

  it('changes what differs and no more than that', () => {
    expect(applyChange(DOC, 'one\ntwo again\nthree\nfour')).toEqual([[7, 7, ' again']])
  })

  it('changes the lines that were added at the end', () => {
    expect(applyChange(DOC, `${DOC}\nfive`)).toEqual([[18, 18, '\nfive']])
  })

  it('changes what was taken from the middle', () => {
    expect(applyChange(DOC, 'one\nfour')).toEqual([[4, 14, '']])
  })

  it('changes everything when the text shares no line with it', () => {
    expect(applyChange(DOC, 'a\nb')).toEqual([[0, 18, 'a\nb']])
  })

  it('holds text that is nothing but one line', () => {
    expect(applyChange('one', 'two')).toEqual([[0, 3, 'two']])
    expect(applyChange('', 'one')).toEqual([[0, 0, 'one']])
    expect(applyChange('one', '')).toEqual([[0, 3, '']])
  })
})

describe('the caret', () => {
  it('comes back at its line and column', () => {
    const fresh = 'one is longer now\ntwo\nthree\nfour'
    expect(after(DOC, caret(DOC, 3, 4), fresh).ranges).toEqual([
      [
        [3, 4],
        [3, 4],
      ],
    ])
  })

  it('stops at the end of a line that is now shorter', () => {
    expect(after(DOC, caret(DOC, 3, 5), 'one\ntwo\nthr\nfour').ranges).toEqual([
      [
        [3, 3],
        [3, 3],
      ],
    ])
  })

  it('stops at the last line when the text lost the one it was on', () => {
    expect(after(DOC, caret(DOC, 4, 2), 'one\ntwo').ranges).toEqual([
      [
        [2, 2],
        [2, 2],
      ],
    ])
  })

  it('stays at the start of the text it was at the start of', () => {
    expect(after(DOC, caret(DOC, 1, 0), 'a whole other note').ranges).toEqual([
      [
        [1, 0],
        [1, 0],
      ],
    ])
  })
})

describe('a selection', () => {
  it('keeps both of its ends, each on its own line', () => {
    const doc = DOC
    const state = EditorState.create({ doc })
    const anchor = state.doc.line(2).from + 1
    const head = state.doc.line(4).from + 3
    const fresh = 'one\ntwo is longer now\nthree\nfour'

    expect(after(doc, EditorSelection.single(anchor, head), fresh).ranges).toEqual([
      [
        [2, 1],
        [4, 3],
      ],
    ])
  })

  it('keeps every range, and which of them is the main one', () => {
    const state = EditorState.create({ doc: DOC })
    const at = (line: number, column: number) => state.doc.line(line).from + column
    const selection = EditorSelection.create(
      [
        EditorSelection.range(at(1, 1), at(1, 2)),
        EditorSelection.range(at(3, 1), at(3, 3)),
        EditorSelection.range(at(4, 0), at(4, 2)),
      ],
      1,
    )

    const put = after(DOC, selection, 'one and more\ntwo\nthree and more\nfour')
    expect(put.ranges).toEqual([
      [
        [1, 1],
        [1, 2],
      ],
      [
        [3, 1],
        [3, 3],
      ],
      [
        [4, 0],
        [4, 2],
      ],
    ])
    expect(put.main).toBe(1)
  })

  it('holds ranges that fell together to one place', () => {
    const state = EditorState.create({ doc: DOC })
    const at = (line: number, column: number) => state.doc.line(line).from + column
    const selection = EditorSelection.create(
      [EditorSelection.cursor(at(3, 4)), EditorSelection.cursor(at(3, 5))],
      0,
    )

    expect(after(DOC, selection, 'one\ntwo\nthr\nfour').ranges).toEqual([
      [
        [3, 3],
        [3, 3],
      ],
    ])
  })
})

/**
 * A note's body always ends with a line break, so a replacement whose shared
 * lines reach the first line is the ordinary case and not a corner.
 */
describe('a document whose shared lines reach the first', () => {
  const shapes: [string, string][] = [
    ['', '# Title\n'],
    ['a\nb\n', 'x\na\nb\n'],
    ['draft\nHello\n', 'Hello\n'],
    ['gone\n', ''],
    ['one\ntwo\n', 'one\ntwo\nthree\n'],
    ['x\n', 'x\n'],
    ['a', 'b'],
  ]

  for (const [was, now] of shapes) {
    it(`is ${JSON.stringify(now)} after ${JSON.stringify(was)}`, () => {
      const state = EditorState.create({ doc: was })
      expect(state.update(replace(state, now)).state.doc.toString()).toBe(now)
    })
  }
})
