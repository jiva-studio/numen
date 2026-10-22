/**
 * What is drawn, what a press makes the selection, where a key goes, and where
 * a drag lands. Plain values, so the awkward cases are cheap: a row that holds
 * nothing, a row named as open that cannot hold, and a drag let go past the
 * last row.
 */
import { describe, expect, it, vi } from 'vitest'
import { flatten, getMarkOf, type Row, type RowId, type RowMarker } from './row'
import {
  everyRow,
  getRowsBetween,
  resolveSelection,
  isSameSelection,
  PLAIN,
  type Press,
} from './select'
import { isTreeKey, stepTo, TREE_KEYS } from './step'
import { getDraggedRows, dragLabel } from './drag'
import { findHolder, isRefused, getRowDropTarget } from './drop'

const ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    hasChildren: true,
    rows: [
      {
        id: 'plans',
        name: 'Plans',
        hasChildren: true,
        rows: [{ id: 'friday', name: 'Friday', hasChildren: false }],
      },
      { id: 'notes', name: 'Notes', hasChildren: false },
    ],
  },
  { id: 'empty', name: 'Empty', hasChildren: true },
  { id: 'loose', name: 'Loose', hasChildren: false },
]

const getShownRows = (...open: readonly RowId[]) => flatten(ROWS, new Set(open))

const getRowIds = (...open: readonly RowId[]) => getShownRows(...open).map((row) => row.id)

describe('what is drawn', () => {
  it('is the top level while nothing is open', () => {
    expect(getRowIds()).toStrictEqual(['work', 'empty', 'loose'])
  })

  it('is what an open row holds, in its place', () => {
    expect(getRowIds('work')).toStrictEqual(['work', 'plans', 'notes', 'empty', 'loose'])
    expect(getRowIds('work', 'plans')).toStrictEqual([
      'work',
      'plans',
      'friday',
      'notes',
      'empty',
      'loose',
    ])
  })

  it('leaves a row shut whose holder is shut', () => {
    expect(getRowIds('plans')).toStrictEqual(['work', 'empty', 'loose'])
  })

  it('leaves a row that cannot hold shut, named or not', () => {
    expect(getRowIds('loose')).toStrictEqual(['work', 'empty', 'loose'])
    expect(getShownRows('loose')[2]?.isOpen).toBe(false)
  })

  it('counts the level from one, down the levels it walked', () => {
    expect(getShownRows('work', 'plans').map((row) => row.level)).toStrictEqual([1, 2, 3, 2, 1, 1])
  })

  it('says which row holds each, and nothing at the top', () => {
    expect(getShownRows('work').map((row) => row.parent)).toStrictEqual([
      null,
      'work',
      'work',
      null,
      null,
    ])
  })

  it('marks the last of the rows its holder holds', () => {
    expect(getShownRows('work').map((row) => row.isLast)).toStrictEqual([
      false,
      false,
      true,
      false,
      true,
    ])
  })

  it('notifies a row that can hold from one that is holding', () => {
    const [, , , empty] = getShownRows('work')
    expect(empty?.hasChildren).toBe(true)
    expect(empty?.hasRows).toBe(false)
  })

  it('draws nothing from nothing', () => {
    expect(flatten([], new Set())).toStrictEqual([])
  })

  it('notifies a row that is holding', () => {
    expect(getShownRows('work').map((row) => row.hasRows)).toStrictEqual([
      true,
      true,
      false,
      false,
      false,
    ])
  })
})

describe('what a row is marked with', () => {
  const marker = (answer: string | null | undefined): RowMarker => ({
    attribute: 'data-place',
    getAttributeValue: () => answer as string | null,
  })

  it('is the attribute set to what the row is answered with', () => {
    expect(getMarkOf(marker('work'), 'notes')).toStrictEqual({ 'data-place': 'work' })
  })

  it('is the attribute set to an empty answer', () => {
    expect(getMarkOf(marker(''), null)).toStrictEqual({ 'data-place': '' })
  })

  it('is requested about the row it marks', () => {
    const getAttributeValue = vi.fn<(row: RowId | null) => string | null>(() => 'x')
    getMarkOf({ attribute: 'data-place', getAttributeValue }, 'notes')
    expect(getAttributeValue).toHaveBeenCalledWith('notes')
  })

  it('is nothing where the row is answered with null', () => {
    expect(getMarkOf(marker(null), 'notes')).toStrictEqual({})
  })

  it('is nothing where the row is answered with nothing', () => {
    expect(getMarkOf(marker(undefined), 'notes')).toStrictEqual({})
  })

  it('is nothing where there is no marker', () => {
    expect(getMarkOf(undefined, 'notes')).toStrictEqual({})
  })
})

describe('the keys a tree answers', () => {
  it('are the ones a step is requested for by', () => {
    expect(TREE_KEYS.every(isTreeKey)).toBe(true)
  })

  it('are no other key', () => {
    expect(isTreeKey('Enter')).toBe(false)
    expect(isTreeKey('a')).toBe(false)
  })
})

describe('where a key takes the keyboard', () => {
  const visibleRows = getShownRows('work')
  const step = (from: RowId | null, key: Parameters<typeof stepTo>[2]) => stepTo(visibleRows, from, key)

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
      turn: { row: 'plans', isOpen: true },
    })
  })

  it('descends into a row already open', () => {
    expect(step('work', 'ArrowRight')).toStrictEqual({ at: 'plans', turn: null })
  })

  it('stays where there is nothing to descend into', () => {
    expect(step('notes', 'ArrowRight')).toStrictEqual({ at: 'notes', turn: null })
    expect(stepTo(getShownRows('work', 'empty'), 'empty', 'ArrowRight')).toStrictEqual({
      at: 'empty',
      turn: null,
    })
  })

  it('closes an open row and stays on it', () => {
    expect(step('work', 'ArrowLeft')).toStrictEqual({
      at: 'work',
      turn: { row: 'work', isOpen: false },
    })
  })

  it('climbs to the holder from a row that is shut', () => {
    expect(step('notes', 'ArrowLeft')).toStrictEqual({ at: 'work', turn: null })
    expect(step('plans', 'ArrowLeft')).toStrictEqual({ at: 'work', turn: null })
  })

  it('stays at the top level, where there is nothing to climb to', () => {
    expect(step('loose', 'ArrowLeft')).toStrictEqual({ at: 'loose', turn: null })
  })

  it('lands on the first row from no row at all, whichever key is pressed', () => {
    expect(step(null, 'ArrowDown')).toStrictEqual({ at: 'work', turn: null })
    expect(step(null, 'ArrowRight')).toStrictEqual({ at: 'work', turn: null })
  })

  it('lands nowhere while nothing is drawn', () => {
    expect(stepTo([], null, 'ArrowDown')).toStrictEqual({ at: null, turn: null })
    expect(stepTo([], 'work', 'End')).toStrictEqual({ at: null, turn: null })
  })
})

describe('where a drag lands', () => {
  // work, plans, notes, empty, loose — one row every 24.
  const visibleRows = getShownRows('work')
  const HEIGHT = 24
  const getDropTargetAt = (y: number, ...dragging: readonly RowId[]) =>
    getRowDropTarget(visibleRows, dragging.length ? dragging : ['friday'], { y, height: HEIGHT })

  it('goes into a row that holds, over the middle of it', () => {
    expect(getDropTargetAt(12)).toStrictEqual({ into: 'work' })
  })

  it('goes between, at either end of a row', () => {
    expect(getDropTargetAt(2)).toStrictEqual({ before: 'work' })
    expect(getDropTargetAt(20)).toStrictEqual({ before: 'plans' })
  })

  it('has no middle over a row that cannot hold', () => {
    expect(getDropTargetAt(52)).toStrictEqual({ before: 'notes' })
    expect(getDropTargetAt(60)).toStrictEqual({ before: 'empty' })
    expect(getDropTargetAt(71)).toStrictEqual({ before: 'empty' })
  })

  it('lands at the top level past the last row, which is the tree’s own area', () => {
    expect(getDropTargetAt(118)).toStrictEqual({ into: null })
    expect(getDropTargetAt(200)).toStrictEqual({ into: null })
  })

  it('lands at the top level under the last row, where nothing arrives after it', () => {
    expect(getDropTargetAt(112)).toStrictEqual({ into: null })
  })

  it('lands nowhere above the first row', () => {
    expect(getDropTargetAt(-4)).toBeNull()
  })

  it('lands before the first row at its very top', () => {
    expect(getDropTargetAt(0)).toStrictEqual({ before: 'work' })
  })

  it('lands nowhere while the rows have no height', () => {
    expect(getRowDropTarget(visibleRows, ['friday'], { y: 12, height: 0 })).toBeNull()
  })

  it('lands nowhere where it would name a row being dragged', () => {
    expect(getDropTargetAt(12, 'work')).toBeNull()
    expect(getDropTargetAt(20, 'plans')).toBeNull()
  })

  it('lands nowhere where it would name any of several being dragged', () => {
    expect(getDropTargetAt(12, 'notes', 'work')).toBeNull()
    expect(getDropTargetAt(20, 'loose', 'plans')).toBeNull()
  })
})

describe('the row a drop target puts what is retained inside', () => {
  const visibleRows = getShownRows('work')

  it('is the row itself, going into one', () => {
    expect(findHolder(visibleRows, { into: 'work' })).toBe('work')
  })

  it('is what holds the row it arrives before', () => {
    expect(findHolder(visibleRows, { before: 'notes' })).toBe('work')
  })

  it('is nothing at the top level', () => {
    expect(findHolder(visibleRows, { before: 'empty' })).toBeNull()
    expect(findHolder(visibleRows, { before: 'gone' })).toBeNull()
  })
})

describe('what rows being dragged are rejected', () => {
  it('is one of themselves', () => {
    expect(isRefused(ROWS, ['work'], 'work')).toBe(true)
    expect(isRefused(ROWS, ['loose', 'work'], 'work')).toBe(true)
  })

  it('is anything one of them holds, however deep', () => {
    expect(isRefused(ROWS, ['work'], 'plans')).toBe(true)
    expect(isRefused(ROWS, ['work'], 'friday')).toBe(true)
    expect(isRefused(ROWS, ['loose', 'work'], 'friday')).toBe(true)
  })

  it('is nothing that holds one of them', () => {
    expect(isRefused(ROWS, ['plans'], 'work')).toBe(false)
  })

  it('is nothing alongside them', () => {
    expect(isRefused(ROWS, ['work'], 'empty')).toBe(false)
    expect(isRefused(ROWS, ['loose'], 'friday')).toBe(false)
    expect(isRefused(ROWS, ['empty'], 'plans')).toBe(false)
    expect(isRefused(ROWS, ['notes', 'loose'], 'empty')).toBe(false)
  })

  it('is nothing at the top level', () => {
    expect(isRefused(ROWS, ['work'], null)).toBe(false)
  })

  it('is nothing at all where nothing is being dragged', () => {
    expect(isRefused(ROWS, [], 'work')).toBe(false)
  })
})

describe('the rows between two rows', () => {
  const visibleRows = getShownRows('work')

  it('are the ones drawn from the first to the second, both among them', () => {
    expect(getRowsBetween(visibleRows, 'work', 'notes')).toStrictEqual(['work', 'plans', 'notes'])
  })

  it('are the same rows in the same order the other way round', () => {
    expect(getRowsBetween(visibleRows, 'notes', 'work')).toStrictEqual(['work', 'plans', 'notes'])
  })

  it('are the one row where both ends are it', () => {
    expect(getRowsBetween(visibleRows, 'plans', 'plans')).toStrictEqual(['plans'])
  })

  it('span whatever a folder boundary puts between them', () => {
    expect(getRowsBetween(visibleRows, 'plans', 'loose')).toStrictEqual([
      'plans',
      'notes',
      'empty',
      'loose',
    ])
  })

  it('are the row reached alone, measured from a row that is not drawn', () => {
    expect(getRowsBetween(visibleRows, 'friday', 'notes')).toStrictEqual(['notes'])
  })

  it('are none at all where the row reached is not drawn', () => {
    expect(getRowsBetween(visibleRows, 'work', 'friday')).toStrictEqual([])
  })
})

describe('the rows between two rows, one of them not drawn', () => {
  const visibleRows = getShownRows('work')

  it('is none where the row reached is not drawn', () => {
    expect(getRowsBetween(visibleRows, 'work', 'gone')).toStrictEqual([])
    expect(getRowsBetween(visibleRows, 'gone', 'gone')).toStrictEqual([])
  })
})

describe('what a press makes the selection', () => {
  const visibleRows = getShownRows('work')
  const JOINING: Press = { isJoining: true, isExtending: false }
  const REACHING: Press = { isJoining: false, isExtending: true }
  const select = (rows: readonly RowId[], anchor: RowId | null, row: RowId, press = PLAIN) =>
    resolveSelection(visibleRows, { rows, anchor }, { row, press })

  it('is the row alone, pressed plainly', () => {
    expect(select(['plans', 'notes'], 'plans', 'loose')).toStrictEqual({
      rows: ['loose'],
      anchor: 'loose',
    })
  })

  it('takes a row in that remains outside it, joining', () => {
    expect(select(['work'], 'work', 'notes', JOINING)).toStrictEqual({
      rows: ['work', 'notes'],
      anchor: 'notes',
    })
  })

  it('takes a row out that remains in it, joining', () => {
    expect(select(['work', 'notes'], 'work', 'notes', JOINING)).toStrictEqual({
      rows: ['work'],
      anchor: 'notes',
    })
  })

  it('draws what it joined in the order the rows are drawn', () => {
    expect(select(['loose', 'notes'], 'loose', 'work', JOINING).rows).toStrictEqual([
      'work',
      'notes',
      'loose',
    ])
  })

  it('reaches from the anchor to the row, in the order they are drawn', () => {
    expect(select(['plans'], 'plans', 'empty', REACHING)).toStrictEqual({
      rows: ['plans', 'notes', 'empty'],
      anchor: 'plans',
    })
  })

  it('reaches back to the anchor from a row above it', () => {
    expect(select(['notes'], 'notes', 'work', REACHING).rows).toStrictEqual([
      'work',
      'plans',
      'notes',
    ])
  })

  it('leaves the anchor at its current position, reaching again from it', () => {
    const once = select(['work'], 'work', 'notes', REACHING)
    const twice = select(once.rows, once.anchor, 'loose', REACHING)

    expect(twice.anchor).toBe('work')
    expect(twice.rows).toStrictEqual(['work', 'plans', 'notes', 'empty', 'loose'])
  })

  it('reaches from the row itself where there is no anchor yet', () => {
    expect(select([], null, 'notes', REACHING)).toStrictEqual({
      rows: ['notes'],
      anchor: 'notes',
    })
  })

  it('drops a row that is not drawn', () => {
    expect(
      resolveSelection(
        getShownRows(),
        { rows: ['plans'], anchor: 'plans' },
        { row: 'loose', press: JOINING },
      ).rows,
    ).toStrictEqual(['loose'])
  })
})

describe('every row drawn, selected at once', () => {
  it('is all of them, in the order they are drawn', () => {
    expect(everyRow(getShownRows('work'), 'notes')).toStrictEqual({
      rows: ['work', 'plans', 'notes', 'empty', 'loose'],
      anchor: 'notes',
    })
  })

  it('puts the anchor on the first row where there is none', () => {
    expect(everyRow(getShownRows(), null).anchor).toBe('work')
  })

  it('is nothing at all where nothing is drawn', () => {
    expect(everyRow([], null)).toStrictEqual({ rows: [], anchor: null })
  })
})

describe('the rows a press drags', () => {
  it('are the selection, where the row remains in it', () => {
    expect(getDraggedRows(['work', 'notes'], 'notes')).toStrictEqual(['work', 'notes'])
  })

  it('are the row alone, at its current position outside', () => {
    expect(getDraggedRows(['work', 'notes'], 'loose')).toStrictEqual(['loose'])
    expect(getDraggedRows([], 'loose')).toStrictEqual(['loose'])
  })
})

describe('two selections', () => {
  it('are the same holding the same rows in the same order', () => {
    expect(isSameSelection(['work', 'notes'], ['work', 'notes'])).toBe(true)
    expect(isSameSelection([], [])).toBe(true)
  })

  it('are not the same in another order, or of another length', () => {
    expect(isSameSelection(['work', 'notes'], ['notes', 'work'])).toBe(false)
    expect(isSameSelection(['work'], ['work', 'notes'])).toBe(false)
  })
})

describe('what follows the pointer', () => {
  const visibleRows = getShownRows('work')
  const point = { x: 40, y: 60 }
  const formatCount = (rows: number) => `${rows} rows`
  const label = (rows: readonly RowId[]) =>
    dragLabel(visibleRows, rows, { at: point, getCountWords: formatCount })

  it('is the name of the one row dragged', () => {
    expect(label(['notes'])).toStrictEqual({ label: 'Notes', at: point })
  })

  it('is how many are dragged, where there are several', () => {
    expect(label(['work', 'notes'])?.label).toBe('2 rows')
  })

  it('is the identity of a row dragged that is not drawn', () => {
    expect(label(['friday'])?.label).toBe('friday')
  })

  it('is nothing at all while nothing is dragged', () => {
    expect(label([])).toBeNull()
  })
})
