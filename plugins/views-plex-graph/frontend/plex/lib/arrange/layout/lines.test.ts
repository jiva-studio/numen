/** A note is not set where its line would run along another line or through another note. */
import { describe, expect, it } from 'vitest'
import { clusters } from './clusters'
import { findFreeSpot, SpotIndex, type Spot } from './freeSpot'
import { LineIndex } from './lines'
import type { PlacementContext } from './placement'
import { resolveOptions } from '../options'
import type { PlacedNode } from '../../node'
import type { PlexNode } from '../../../../types'

const SIZE = { width: 100, height: 30 }
const GAPS = { gap: 16, lineGap: 20 }
const at = (x: number, y: number): Spot => ({ x, y, ...SIZE })

const FOCUS: Spot = { x: 0, y: 0, width: 140, height: 40 }

function createLines(links: readonly [string, string][]) {
  const index = new SpotIndex([FOCUS])
  const lines = new LineIndex(
    links.map(([from, to]) => ({ from, to })),
    'focus',
    index,
  )
  lines.place('focus', { x: 0, y: 0 }, FOCUS)
  const put = (id: string, spot: Spot) => {
    index.add(spot)
    lines.place(id, { x: spot.x, y: spot.y }, spot)
  }
  return { index, lines, put }
}

describe('the lines of a note stay apart', () => {
  it('when its line would continue the line of its neighbor straight on', () => {
    const { lines, put } = createLines([
      ['middle', 'focus'],
      ['top', 'middle'],
    ])
    put('middle', at(0, -100))

    expect(lines.isClear('top', { x: 0, y: -200 }, at(0, -200))).toBe(false)
    expect(lines.isClear('top', { x: 160, y: -200 }, at(160, -200))).toBe(true)
  })

  it('when its line would run back along the line of its neighbor', () => {
    const { lines, put } = createLines([
      ['middle', 'focus'],
      ['near', 'middle'],
    ])
    put('middle', at(0, -200))

    expect(lines.isClear('near', { x: 0, y: -100 }, at(0, -100))).toBe(false)
  })

  it('but a parent and a child may sit on opposite sides of the focus', () => {
    const { lines, put } = createLines([
      ['parent', 'focus'],
      ['focus', 'child'],
    ])
    put('parent', at(0, -100))

    expect(lines.isClear('child', { x: 0, y: 100 }, at(0, 100))).toBe(true)
  })

  it('when its line would run through another note', () => {
    const { lines, put } = createLines([['far', 'focus']])
    put('front', at(0, -100))

    expect(lines.isClear('far', { x: 0, y: -200 }, at(0, -200))).toBe(false)
    expect(lines.isClear('far', { x: 300, y: -200 }, at(300, -200))).toBe(true)
  })

  it('when the note itself would lie on a line that is already drawn', () => {
    const { lines, put } = createLines([['far', 'focus']])
    put('far', at(0, -300))

    expect(lines.isClear('other', { x: 0, y: -150 }, at(0, -150))).toBe(false)
    expect(lines.isClear('other', { x: 200, y: -150 }, at(200, -150))).toBe(true)
  })
})

describe('a free spot', () => {
  const request = {
    start: { x: 0, y: -100 },
    middle: { x: 0, y: -100 },
    direction: { x: 0, y: -1 },
    size: SIZE,
    group: [] as Spot[],
    gaps: GAPS,
  }

  it('that keeps its lines apart is taken in place of a nearer one that does not', () => {
    const { index } = createLines([])
    const spot = findFreeSpot(index, {
      ...request,
      isLineClear: (candidate) => candidate.x !== 0,
    })

    expect(spot.x).not.toBe(0)
  })

  it('that does not is still taken when no spot keeps its lines apart', () => {
    const { index } = createLines([])

    expect(findFreeSpot(index, { ...request, isLineClear: () => false })).toStrictEqual(
      findFreeSpot(index, request),
    )
  })

  it('that does not is still taken when the window holds no other spot', () => {
    const { index } = createLines([])
    const only = (spot: Spot) => spot.x === 0 && spot.y === -100
    const spot = findFreeSpot(index, { ...request, fits: only, isLineClear: () => false })

    expect(spot).toMatchObject({ x: 0, y: -100 })
  })
})

describe('a window that holds one column', () => {
  const options = resolveOptions({ viewport: { width: 176, height: 900 } })
  const node = (id: string): PlexNode => ({ id, title: id, relation: 'parent' })
  const focus: PlacedNode = {
    id: 'focus',
    title: 'Focus',
    relation: 'focus',
    x: 0,
    y: 0,
    width: 140,
    height: 40,
    order: 0,
    opacity: 1,
  }
  const context: PlacementContext = {
    options,
    width: () => options.nodeSize.width,
    room: () => ({ top: 0, bottom: 0, width: 0 }),
    fits: (spot) => Math.abs(spot.x) + spot.width / 2 <= 88 - options.margin,
    edges: [
      { from: 'chapter', to: 'focus' },
      { from: 'part', to: 'chapter' },
    ],
  }

  it('still places both notes of a line of parents', () => {
    const placed = clusters.place({ parent: [node('chapter'), node('part')] }, focus, context)

    expect(placed.map((box) => box.id)).toStrictEqual(['chapter', 'part'])
  })
})
