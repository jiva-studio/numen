/**
 * What is drawn, where a key goes, and where a drag lands. Plain values, so
 * the awkward cases are cheap: a row that holds nothing, a row named as open
 * that cannot hold, and a drag let go past the last row.
 */
import { describe, expect, it } from 'vitest'
import {
  flatten,
  holderOf,
  isTreeKey,
  landing,
  refuses,
  stepTo,
  TREE_KEYS,
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
  const at = (y: number, dragging: RowId = 'friday') => landing(shown, dragging, y, HEIGHT)

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

  it('lands nowhere past the last row', () => {
    expect(at(118)).toBeNull()
    expect(at(200)).toBeNull()
  })

  it('lands nowhere above the first row', () => {
    expect(at(-4)).toBeNull()
  })

  it('lands nowhere while the rows have no height', () => {
    expect(landing(shown, 'friday', 12, 0)).toBeNull()
  })

  it('lands nowhere where it would name the row being dragged', () => {
    expect(at(12, 'work')).toBeNull()
    expect(at(20, 'plans')).toBeNull()
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

describe('what a row is refused', () => {
  it('is itself', () => {
    expect(refuses(ROWS, 'work', 'work')).toBe(true)
  })

  it('is anything it holds, however deep', () => {
    expect(refuses(ROWS, 'work', 'plans')).toBe(true)
    expect(refuses(ROWS, 'work', 'friday')).toBe(true)
  })

  it('is nothing that holds it', () => {
    expect(refuses(ROWS, 'plans', 'work')).toBe(false)
  })

  it('is nothing beside it', () => {
    expect(refuses(ROWS, 'work', 'empty')).toBe(false)
    expect(refuses(ROWS, 'notes', 'empty')).toBe(false)
  })

  it('is nothing at the top level', () => {
    expect(refuses(ROWS, 'work', null)).toBe(false)
  })
})
