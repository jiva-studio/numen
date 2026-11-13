/** The names under and over a pill: wrapped into rows, given room, and placed. */
import { describe, expect, it } from 'vitest'
import { getNamesRoom, placeNames } from './names'
import type { PlexNames } from '../../types'

const FONT_SIZE = 10
const SPACING = FONT_SIZE * 0.9
const ROW_HEIGHT = FONT_SIZE * 1.3

const named = (...titles: string[]) => titles.map((title) => ({ id: `id:${title}`, title }))

const createNames = (above: string[], below: string[]): PlexNames => ({
  above: named(...above),
  below: named(...below),
})

/** Every title is 40 wide, so two fit a row of 100 and three do not. */
const request = { fontSize: FONT_SIZE, width: 100, measure: () => 40 }

describe('the room of the names of a note', () => {
  it('is none for a note with no names', () => {
    const room = getNamesRoom(createNames([], []), request)

    expect(room.above).toHaveLength(0)
    expect(room.below).toHaveLength(0)
    expect(room.top).toBe(0)
    expect(room.bottom).toBe(0)
    expect(room.width).toBe(0)
  })

  it('wraps names into rows no wider than the width of the node', () => {
    const room = getNamesRoom(createNames([], ['a', 'b', 'c']), request)

    expect(room.below).toHaveLength(2)
    expect(room.width).toBeCloseTo(40 + SPACING + 40)
    expect(room.width).toBeLessThanOrEqual(100)
  })

  it('takes the height of its rows under the note and none over it', () => {
    const room = getNamesRoom(createNames([], ['a', 'b', 'c']), request)

    expect(room.rowHeight).toBeCloseTo(ROW_HEIGHT)
    expect(room.spacing).toBeCloseTo(SPACING)
    expect(room.bottom).toBeCloseTo(2 * ROW_HEIGHT + ROW_HEIGHT * 0.15)
    expect(room.top).toBe(0)
  })

  it('takes the height of its rows over the note for the names of its parents', () => {
    const room = getNamesRoom(createNames(['a'], []), request)

    expect(room.above).toHaveLength(1)
    expect(room.top).toBeCloseTo(ROW_HEIGHT + ROW_HEIGHT * 0.15)
    expect(room.bottom).toBe(0)
  })

  it('is wider for wider names, and never wider than the node', () => {
    const wide = getNamesRoom(createNames([], ['a']), { ...request, measure: () => 500 })

    expect(wide.width).toBeLessThanOrEqual(100)
  })

  it('is the same for the same names', () => {
    expect(getNamesRoom(createNames(['a'], ['b', 'c']), request)).toEqual(
      getNamesRoom(createNames(['a'], ['b', 'c']), request),
    )
  })
})

describe('the names placed around a pill', () => {
  const pill = { x: 200, y: 100, height: 36 }

  it('places the names of the parents over the pill and the rest under it, centered on it', () => {
    const placed = placeNames(getNamesRoom(createNames(['up'], ['one', 'two']), request), pill)

    const up = placed.find((one) => one.title === 'up')!
    const one = placed.find((name) => name.title === 'one')!
    const two = placed.find((name) => name.title === 'two')!
    expect(up.y + up.height / 2).toBeLessThanOrEqual(pill.y - pill.height / 2)
    expect(one.y - one.height / 2).toBeGreaterThanOrEqual(pill.y + pill.height / 2)
    expect(two.y - two.height / 2).toBeGreaterThanOrEqual(pill.y + pill.height / 2)
    const row = [one, two]
    const left = Math.min(...row.map((name) => name.x - name.width / 2))
    const right = Math.max(...row.map((name) => name.x + name.width / 2))
    expect((left + right) / 2).toBeCloseTo(pill.x)
  })

  it('puts the first row under the pill nearest to it and the last row over the pill nearest to it', () => {
    const placed = placeNames(
      getNamesRoom(createNames(['u1', 'u2', 'u3'], ['d1', 'd2', 'd3']), request),
      pill,
    )

    const at = (title: string) => placed.find((one) => one.title === title)!
    expect(at('d1').y).toBeLessThan(at('d3').y)
    expect(at('u3').y).toBeGreaterThan(at('u1').y)
  })

  it('keeps the order of the names along a row', () => {
    const placed = placeNames(getNamesRoom(createNames([], ['a', 'b']), request), pill)

    const a = placed.find((one) => one.title === 'a')!
    const b = placed.find((one) => one.title === 'b')!
    expect(a.x).toBeLessThan(b.x)
    expect(a.id).toBe('id:a')
  })

  it('every name is inside its room', () => {
    const room = getNamesRoom(createNames(['u'], ['a', 'b', 'c']), request)

    for (const one of placeNames(room, pill)) {
      expect(one.y - one.height / 2).toBeGreaterThanOrEqual(
        pill.y - pill.height / 2 - room.top - 1e-6,
      )
      expect(one.y + one.height / 2).toBeLessThanOrEqual(
        pill.y + pill.height / 2 + room.bottom + 1e-6,
      )
      expect(Math.abs(one.x - pill.x) + one.width / 2).toBeLessThanOrEqual(room.width / 2 + 1e-6)
    }
  })

  it('cuts a name wider than the node with an ellipsis', () => {
    const room = getNamesRoom(createNames([], ['An extremely long name of a note']), {
      ...request,
      measure: (title) => title.length * 10,
    })

    const [one] = placeNames(room, pill)

    expect(one!.width).toBeLessThanOrEqual(100)
    expect(one!.title.endsWith('…')).toBe(true)
    expect(one!.title.length).toBeLessThan('An extremely long name of a note'.length)
  })

  it('places nothing for a note with no names', () => {
    expect(placeNames(getNamesRoom(createNames([], []), request), pill)).toEqual([])
  })
})
