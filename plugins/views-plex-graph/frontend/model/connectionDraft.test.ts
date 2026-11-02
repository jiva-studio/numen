import { describe, expect, it } from 'vitest'
import type { Connection, ConnectionSide } from '@numen/plugin-sdk'
import {
  NEW_LINK_ROLE,
  addReverseSide,
  areConnectionsEqual,
  cycleDirection,
  getNextDirection,
  getRows,
  isConnectionEmpty,
  removeSide,
  setSideLabel,
} from './connectionDraft'

const connection = (more: Partial<Connection> = {}): Connection => ({
  first: 'Harbour.md',
  second: 'Lighthouse.md',
  role: 'child',
  type: '',
  firstSide: null,
  secondSide: null,
  ...more,
})

const side = (label: string, marker: ConnectionSide['marker'] = null): ConnectionSide => ({
  label,
  marker,
  reason: 'kept',
})

const undirected = connection({ firstSide: side('guides') })
const forward = connection({ firstSide: side('guides', 'arrow') })
const backward = connection({ secondSide: side('guides', 'arrow') })

describe('getRows', () => {
  it('gives no row for a connection with no record', () => {
    expect(getRows(connection())).toStrictEqual([])
  })

  it('gives the direction of one record from the note that writes it and its marker', () => {
    expect(getRows(undirected).map((row) => [row.key, row.direction])).toEqual([['first', 'none']])
    expect(getRows(forward).map((row) => [row.key, row.direction])).toEqual([['first', 'forward']])
    expect(getRows(backward).map((row) => [row.key, row.direction])).toEqual([
      ['second', 'backward'],
    ])
  })

  it('gives a record of the second note without a marker no direction', () => {
    const draft = connection({ secondSide: side('guides') })
    expect(getRows(draft).map((row) => row.direction)).toEqual(['none'])
  })

  it('gives the record of the first note before that of the second', () => {
    const both = connection({ firstSide: side('one', 'arrow'), secondSide: side('two', 'arrow') })
    expect(getRows(both).map((row) => [row.key, row.side.label, row.direction])).toEqual([
      ['first', 'one', 'forward'],
      ['second', 'two', 'backward'],
    ])
  })
})

describe('cycleDirection', () => {
  it('turns no direction into a record of the first note with an arrow', () => {
    expect(cycleDirection(undirected, 'first')).toStrictEqual(forward)
  })

  it('moves a forward record to the second note with its label and reason', () => {
    expect(cycleDirection(forward, 'first')).toStrictEqual(backward)
  })

  it('moves a backward record to the first note without an arrow', () => {
    expect(cycleDirection(backward, 'first')).toStrictEqual(undirected)
  })

  it('returns to the first direction after three steps and keeps the role', () => {
    const parent = connection({ role: 'parent', firstSide: side('above') })
    const steps = [cycleDirection(parent, 'first')]
    steps.push(cycleDirection(steps[0]!, 'first'))
    steps.push(cycleDirection(steps[1]!, 'first'))

    expect(steps.map((step) => getRows(step)[0]?.direction)).toEqual([
      'forward',
      'backward',
      'none',
    ])
    expect(steps.every((step) => step.role === 'parent')).toBe(true)
    expect(steps[2]).toStrictEqual(parent)
  })

  it('gives each of two undirected records its own arrow and takes it away again', () => {
    const both = connection({ firstSide: side('one'), secondSide: side('two') })

    const first = cycleDirection(both, 'first')
    expect(getRows(first).map((row) => [row.key, row.direction])).toEqual([
      ['first', 'forward'],
      ['second', 'none'],
    ])
    const second = cycleDirection(first, 'second')
    expect(getRows(second).map((row) => row.direction)).toEqual(['forward', 'backward'])
    expect(second.firstSide?.label).toBe('one')
    expect(second.secondSide?.label).toBe('two')
    expect(cycleDirection(cycleDirection(second, 'first'), 'second')).toStrictEqual(both)
  })

  it('leaves a draft with no record as it is', () => {
    const empty = connection()
    expect(cycleDirection(empty, 'first')).toBe(empty)
  })
})

describe('getNextDirection', () => {
  it('turns a lone record through the three directions', () => {
    expect(getNextDirection('none', 'first', 1)).toBe('forward')
    expect(getNextDirection('forward', 'first', 1)).toBe('backward')
    expect(getNextDirection('backward', 'second', 1)).toBe('none')
  })

  it('gives one of two records its own arrow or none', () => {
    expect(getNextDirection('none', 'first', 2)).toBe('forward')
    expect(getNextDirection('none', 'second', 2)).toBe('backward')
    expect(getNextDirection('forward', 'first', 2)).toBe('none')
    expect(getNextDirection('backward', 'second', 2)).toBe('none')
  })
})

describe('addReverseSide', () => {
  it('adds the record of the second note with an empty label and points both records', () => {
    expect(addReverseSide(undirected)).toStrictEqual(
      connection({
        firstSide: side('guides', 'arrow'),
        secondSide: { label: '', marker: 'arrow', reason: '' },
      }),
    )
  })

  it('adds the record of the first note when only the second writes one', () => {
    expect(addReverseSide(backward)).toStrictEqual(
      connection({
        firstSide: { label: '', marker: 'arrow', reason: '' },
        secondSide: side('guides', 'arrow'),
      }),
    )
  })

  it('adds an undirected record of the first note to a draft with none', () => {
    const draft = addReverseSide(connection())
    expect(draft.firstSide).toStrictEqual({ label: '', marker: null, reason: '' })
    expect(draft.secondSide).toBeNull()
    expect(draft.role).toBe('child')
  })

  it('gives a connection with no role the role of a new link', () => {
    expect(addReverseSide(connection({ role: null })).role).toBe(NEW_LINK_ROLE)
  })

  it('leaves a draft with two records as it is', () => {
    const both = addReverseSide(forward)
    expect(addReverseSide(both)).toBe(both)
  })
})

describe('removeSide and setSideLabel', () => {
  it('removes the record of one note and keeps the other', () => {
    const both = addReverseSide(forward)
    expect(removeSide(both, 'second')).toStrictEqual(forward)
    expect(getRows(removeSide(both, 'first')).map((row) => row.direction)).toEqual(['backward'])
  })

  it('replaces the label of one record and ignores a note that writes none', () => {
    expect(setSideLabel(forward, 'first', 'lights').firstSide?.label).toBe('lights')
    expect(setSideLabel(forward, 'second', 'lights')).toBe(forward)
  })

  it('is empty for a connection without records', () => {
    expect(isConnectionEmpty(connection())).toBe(true)
    expect(isConnectionEmpty(backward)).toBe(false)
  })
})

describe('areConnectionsEqual', () => {
  it('compares the role, the type and both records', () => {
    expect(areConnectionsEqual(forward, connection({ firstSide: side('guides', 'arrow') }))).toBe(
      true,
    )
    expect(areConnectionsEqual(forward, { ...forward, role: 'jump' })).toBe(false)
    expect(areConnectionsEqual(forward, { ...forward, type: 'requires' })).toBe(false)
    expect(areConnectionsEqual(forward, undirected)).toBe(false)
    expect(areConnectionsEqual(forward, backward)).toBe(false)
    expect(areConnectionsEqual(forward, setSideLabel(forward, 'first', 'x'))).toBe(false)
  })
})
