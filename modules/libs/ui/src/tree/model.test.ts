/**
 * What is drawn, what a press makes the selection, where a key goes, and where
 * a drag lands. Plain values, so the awkward cases are cheap: a row that holds
 * nothing, a row named as open that cannot hold, and a drag let go past the
 * last row.
 */
import { describe, expect, it } from 'vitest'
import {
  between,
  carried,
  carries,
  everyRow,
  flatten,
  holderOf,
  isTreeKey,
  landing,
  refuses,
  sameRows,
  selects,
  stepTo,
  PLAIN,
  TREE_KEYS,
  type Press,
  type Row,
  type RowId,
} from './model'

const ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    holds: true,
    rows: [
      {
        id: 'plans',
        name: 'Plans',
        holds: true,
        rows: [{ id: 'friday', name: 'Friday', holds: false }],
      },
      { id: 'notes', name: 'Notes', holds: false },
    ],
  },
  { id: 'empty', name: 'Empty', holds: true },
  { id: 'loose', name: 'Loose', holds: false },
]

const shownWith = (...open: readonly RowId[]) => flatten(ROWS, new Set(open))

const names = (...open: readonly RowId[]) => shownWith(...open).map((row) => row.id)

describe('what is drawn', () => {
  it('is the top level while nothing is open', () => {
    expect(names()).toStrictEqual(['work', 'empty', 'loose'])
  })

  it('is what an open row holds, in its place', () => {
    expect(names('work')).toStrictEqual(['work', 'plans', 'notes', 'empty', 'loose'])
    expect(names('work', 'plans')).toStrictEqual([
      'work',
      'plans',
      'friday',
      'notes',
      'empty',
      'loose',
    ])
  })

  it('leaves a row shut whose holder is shut', () => {
    expect(names('plans')).toStrictEqual(['work', 'empty', 'loose'])
  })

  it('leaves a row that cannot hold shut, named or not', () => {
    expect(names('loose')).toStrictEqual(['work', 'empty', 'loose'])
    expect(shownWith('loose')[2]?.open).toBe(false)
  })

  it('counts the level from one, down the levels it walked', () => {
    expect(shownWith('work', 'plans').map((row) => row.level)).toStrictEqual([1, 2, 3, 2, 1, 1])
  })

  it('says which row holds each, and nothing at the top', () => {
    expect(shownWith('work').map((row) => row.parent)).toStrictEqual([
      null,
      'work',
      'work',
      null,
      null,
    ])
  })

  it('marks the last of the rows its holder holds', () => {
    expect(shownWith('work').map((row) => row.last)).toStrictEqual([
      false,
      false,
      true,
      false,
      true,
    ])
  })

  it('tells a row that can hold from one that is holding', () => {
    const [, , , empty] = shownWith('work')
    expect(empty?.holds).toBe(true)
    expect(empty?.holding).toBe(false)
  })

  it('draws nothing from nothing', () => {
    expect(flatten([], new Set())).toStrictEqual([])
  })
})

describe('the keys a tree answers', () => {
  it('are the ones a step is asked for by', () => {
    expect(TREE_KEYS.every(isTreeKey)).toBe(true)
  })

  it('are no other key', () => {
    expect(isTreeKey('Enter')).toBe(false)
    expect(isTreeKey('a')).toBe(false)
  })
})

describe('where a key takes the keyboard', () => {
  const shown = shownWith('work')
  const step = (from: RowId | null, key: Parameters<typeof stepTo>[2]) =>
    stepTo(shown, from, key)

  it('moves a row down and a row up', () => {
    expect(step('work', 'ArrowDown')).toStrictEqual({ at: 'plans', turn: null })
    expect(step('plans', 'ArrowUp')).toStrictEqual({ at: 'work', turn: null })
  })

  it('stops at either end', () => {
    expect(step('loose', 'ArrowDown').at).toBe('loose')
    expect(step('work', 'ArrowUp').at).toBe('work')
  })

  it('lands on the first row from no row at all', () => {
    expect(step(null, 'ArrowUp')).toStrictEqual({ at: 'work', turn: null })
    expect(step('gone', 'End')).toStrictEqual({ at: 'work', turn: null })
  })

  it('goes to the ends', () => {
    expect(step('notes', 'Home').at).toBe('work')
    expect(step('notes', 'End').at).toBe('loose')
  })

  it('opens a shut row and stays on it', () => {
    expect(step('plans', 'ArrowRight')).toStrictEqual({
      at: 'plans',
      turn: { row: 'plans', open: true },
    })
  })

  it('descends into a row already open', () => {
    expect(step('work', 'ArrowRight')).toStrictEqual({ at: 'plans', turn: null })
  })

  it('stays where there is nothing to descend into', () => {
    expect(step('notes', 'ArrowRight')).toStrictEqual({ at: 'notes', turn: null })
    expect(stepTo(shownWith('work', 'empty'), 'empty', 'ArrowRight')).toStrictEqual({
      at: 'empty',
      turn: null,
    })
  })

  it('closes an open row and stays on it', () => {
    expect(step('work', 'ArrowLeft')).toStrictEqual({
      at: 'work',
      turn: { row: 'work', open: false },
    })
  })

  it('climbs to the holder from a row that is shut', () => {
    expect(step('notes', 'ArrowLeft')).toStrictEqual({ at: 'work', turn: null })
    expect(step('plans', 'ArrowLeft')).toStrictEqual({ at: 'work', turn: null })
  })

  it('stays at the top level, where there is nothing to climb to', () => {
    expect(step('loose', 'ArrowLeft')).toStrictEqual({ at: 'loose', turn: null })
  })

  it('lands nowhere while nothing is drawn', () => {
    expect(stepTo([], null, 'ArrowDown')).toStrictEqual({ at: null, turn: null })
    expect(stepTo([], 'work', 'End')).toStrictEqual({ at: null, turn: null })
  })
})

describe('where a drag lands', () => {
  // work, plans, notes, empty, loose — one row every 24.
  const shown = shownWith('work')
  const HEIGHT = 24
  const at = (y: number, ...dragging: readonly RowId[]) =>
    landing(shown, dragging.length ? dragging : ['friday'], y, HEIGHT)

  it('goes into a row that holds, over the middle of it', () => {
    expect(at(12)).toStrictEqual({ into: 'work' })
  })

  it('goes between, at either end of a row', () => {
    expect(at(2)).toStrictEqual({ before: 'work' })
    expect(at(20)).toStrictEqual({ before: 'plans' })
  })

  it('has no middle over a row that cannot hold', () => {
    expect(at(52)).toStrictEqual({ before: 'notes' })
    expect(at(60)).toStrictEqual({ before: 'empty' })
    expect(at(71)).toStrictEqual({ before: 'empty' })
  })

  it('lands at the top level past the last row, which is the tree’s own area', () => {
    expect(at(118)).toStrictEqual({ into: null })
    expect(at(200)).toStrictEqual({ into: null })
  })

  it('lands at the top level under the last row, where nothing comes after it', () => {
    expect(at(112)).toStrictEqual({ into: null })
  })

  it('lands nowhere above the first row', () => {
    expect(at(-4)).toBeNull()
  })

  it('lands nowhere while the rows have no height', () => {
    expect(landing(shown, ['friday'], 12, 0)).toBeNull()
  })

  it('lands nowhere where it would name a row being dragged', () => {
    expect(at(12, 'work')).toBeNull()
    expect(at(20, 'plans')).toBeNull()
  })

  it('lands nowhere where it would name any of several being dragged', () => {
    expect(at(12, 'notes', 'work')).toBeNull()
    expect(at(20, 'loose', 'plans')).toBeNull()
  })
})

describe('the row a landing puts what is held inside', () => {
  const shown = shownWith('work')

  it('is the row itself, going into one', () => {
    expect(holderOf(shown, { into: 'work' })).toBe('work')
  })

  it('is what holds the row it comes before', () => {
    expect(holderOf(shown, { before: 'notes' })).toBe('work')
  })

  it('is nothing at the top level', () => {
    expect(holderOf(shown, { before: 'empty' })).toBeNull()
    expect(holderOf(shown, { before: 'gone' })).toBeNull()
  })
})

describe('what rows being dragged are refused', () => {
  it('is one of themselves', () => {
    expect(refuses(ROWS, ['work'], 'work')).toBe(true)
    expect(refuses(ROWS, ['loose', 'work'], 'work')).toBe(true)
  })

  it('is anything one of them holds, however deep', () => {
    expect(refuses(ROWS, ['work'], 'plans')).toBe(true)
    expect(refuses(ROWS, ['work'], 'friday')).toBe(true)
    expect(refuses(ROWS, ['loose', 'work'], 'friday')).toBe(true)
  })

  it('is nothing that holds one of them', () => {
    expect(refuses(ROWS, ['plans'], 'work')).toBe(false)
  })

  it('is nothing beside them', () => {
    expect(refuses(ROWS, ['work'], 'empty')).toBe(false)
    expect(refuses(ROWS, ['notes', 'loose'], 'empty')).toBe(false)
  })

  it('is nothing at the top level', () => {
    expect(refuses(ROWS, ['work'], null)).toBe(false)
  })

  it('is nothing at all where nothing is being dragged', () => {
    expect(refuses(ROWS, [], 'work')).toBe(false)
  })
})

describe('the rows between two rows', () => {
  const shown = shownWith('work')

  it('are the ones drawn from the first to the second, both among them', () => {
    expect(between(shown, 'work', 'notes')).toStrictEqual(['work', 'plans', 'notes'])
  })

  it('are the same rows in the same order the other way round', () => {
    expect(between(shown, 'notes', 'work')).toStrictEqual(['work', 'plans', 'notes'])
  })

  it('are the one row where both ends are it', () => {
    expect(between(shown, 'plans', 'plans')).toStrictEqual(['plans'])
  })

  it('span whatever a folder boundary puts between them', () => {
    expect(between(shown, 'plans', 'loose')).toStrictEqual(['plans', 'notes', 'empty', 'loose'])
  })

  it('are the row reached alone, measured from a row that is not drawn', () => {
    expect(between(shown, 'friday', 'notes')).toStrictEqual(['notes'])
  })

  it('are none at all where the row reached is not drawn', () => {
    expect(between(shown, 'work', 'friday')).toStrictEqual([])
  })
})

describe('what a press makes the selection', () => {
  const shown = shownWith('work')
  const JOINING: Press = { joining: true, reaching: false }
  const REACHING: Press = { joining: false, reaching: true }

  it('is the row alone, pressed plainly', () => {
    expect(selects(shown, ['plans', 'notes'], 'plans', 'loose', PLAIN)).toStrictEqual({
      rows: ['loose'],
      anchor: 'loose',
    })
  })

  it('takes a row in that stands outside it, joining', () => {
    expect(selects(shown, ['work'], 'work', 'notes', JOINING)).toStrictEqual({
      rows: ['work', 'notes'],
      anchor: 'notes',
    })
  })

  it('takes a row out that stands in it, joining', () => {
    expect(selects(shown, ['work', 'notes'], 'work', 'notes', JOINING)).toStrictEqual({
      rows: ['work'],
      anchor: 'notes',
    })
  })

  it('draws what it joined in the order the rows are drawn', () => {
    expect(selects(shown, ['loose', 'notes'], 'loose', 'work', JOINING).rows).toStrictEqual([
      'work',
      'notes',
      'loose',
    ])
  })

  it('reaches from the anchor to the row, in the order they are drawn', () => {
    expect(selects(shown, ['plans'], 'plans', 'empty', REACHING)).toStrictEqual({
      rows: ['plans', 'notes', 'empty'],
      anchor: 'plans',
    })
  })

  it('reaches back to the anchor from a row above it', () => {
    expect(selects(shown, ['notes'], 'notes', 'work', REACHING).rows).toStrictEqual([
      'work',
      'plans',
      'notes',
    ])
  })

  it('leaves the anchor where it stands, reaching again from it', () => {
    const once = selects(shown, ['work'], 'work', 'notes', REACHING)
    const twice = selects(shown, once.rows, once.anchor, 'loose', REACHING)

    expect(twice.anchor).toBe('work')
    expect(twice.rows).toStrictEqual(['work', 'plans', 'notes', 'empty', 'loose'])
  })

  it('reaches from the row itself where there is no anchor yet', () => {
    expect(selects(shown, [], null, 'notes', REACHING)).toStrictEqual({
      rows: ['notes'],
      anchor: 'notes',
    })
  })

  it('drops a row that is not drawn', () => {
    expect(selects(shownWith(), ['plans'], 'plans', 'loose', JOINING).rows).toStrictEqual(['loose'])
  })
})

describe('every row drawn, selected at once', () => {
  it('is all of them, in the order they are drawn', () => {
    expect(everyRow(shownWith('work'), 'notes')).toStrictEqual({
      rows: ['work', 'plans', 'notes', 'empty', 'loose'],
      anchor: 'notes',
    })
  })

  it('puts the anchor on the first row where there is none', () => {
    expect(everyRow(shownWith(), null).anchor).toBe('work')
  })

  it('is nothing at all where nothing is drawn', () => {
    expect(everyRow([], null)).toStrictEqual({ rows: [], anchor: null })
  })
})

describe('the rows a press carries', () => {
  it('are the selection, where the row stands in it', () => {
    expect(carries(['work', 'notes'], 'notes')).toStrictEqual(['work', 'notes'])
  })

  it('are the row alone, where it stands outside', () => {
    expect(carries(['work', 'notes'], 'loose')).toStrictEqual(['loose'])
    expect(carries([], 'loose')).toStrictEqual(['loose'])
  })
})

describe('two selections', () => {
  it('are the same holding the same rows in the same order', () => {
    expect(sameRows(['work', 'notes'], ['work', 'notes'])).toBe(true)
    expect(sameRows([], [])).toBe(true)
  })

  it('are not the same in another order, or of another length', () => {
    expect(sameRows(['work', 'notes'], ['notes', 'work'])).toBe(false)
    expect(sameRows(['work'], ['work', 'notes'])).toBe(false)
  })
})

describe('what follows the pointer', () => {
  const shown = shownWith('work')
  const at = { x: 40, y: 60 }
  const counted = (rows: number) => `${rows} rows`

  it('is the name of the one row carried', () => {
    expect(carried(shown, ['notes'], at, counted)).toStrictEqual({ says: 'Notes', at })
  })

  it('is how many are carried, where there are several', () => {
    expect(carried(shown, ['work', 'notes'], at, counted)?.says).toBe('2 rows')
  })

  it('is the identity of a row carried that is not drawn', () => {
    expect(carried(shown, ['friday'], at, counted)?.says).toBe('friday')
  })

  it('is nothing at all while nothing is carried', () => {
    expect(carried(shown, [], at, counted)).toBeNull()
  })
})
