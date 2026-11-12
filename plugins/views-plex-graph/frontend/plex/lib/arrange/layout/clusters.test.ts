/** The pieces of the clusters placement that have a shape of their own. */
import { describe, expect, it } from 'vitest'
import { clusters, getAnchor, getClusterDirection, takeTurns } from './clusters'
import { DEFAULT_OPTIONS } from '../options'
import type { PlacedNode } from '../../node'
import type { PlexNode } from '../../../../types'

const node = (id: string, relation: PlexNode['relation']): PlexNode => ({ id, title: id, relation })

describe('the direction of a group', () => {
  it.each([
    ['up', [0, -1]],
    ['down', [0, 1]],
    ['left', [-1, 0]],
    ['right', [1, 0]],
  ] as const)('is a unit vector straight %s when it does not lean', (side, [x, y]) => {
    expect(getClusterDirection(side, null)).toStrictEqual({ x, y })
  })

  it('leans a leftward group down or up by the same amount', () => {
    const down = getClusterDirection('left', 'down')
    const up = getClusterDirection('left', 'up')

    for (const vector of [down, up]) {
      expect(vector.x).toBeLessThan(0)
      expect(Math.hypot(vector.x, vector.y)).toBeCloseTo(1, 9)
    }
    expect(down.y).toBeGreaterThan(0)
    expect(up.y).toBeLessThan(0)
    expect(down.y).toBeCloseTo(-up.y, 9)
    expect(down.y / -down.x).toBeCloseTo(0.42, 9)
  })
})

describe('the nearest point a group may reach', () => {
  const focus = { width: 176, height: 44 }
  const { focusGap, lineGap } = DEFAULT_OPTIONS

  it('is the focus half-size and the focus gap away for a straight group', () => {
    expect(getAnchor({ x: 0, y: 1 }, focus, DEFAULT_OPTIONS)).toStrictEqual({
      x: 0,
      y: focus.height / 2 + focusGap,
    })
    expect(getAnchor({ x: -1, y: 0 }, focus, DEFAULT_OPTIONS)).toStrictEqual({
      x: -(focus.width / 2 + focusGap),
      y: 0,
    })
  })

  it('is clear of the focus on both axes for a leaning group', () => {
    const direction = getClusterDirection('left', 'down')
    expect(getAnchor(direction, focus, DEFAULT_OPTIONS)).toStrictEqual({
      x: -(focus.width / 2 + focusGap),
      y: focus.height / 2 + lineGap,
    })
  })
})

describe('the turns of the groups', () => {
  it('takes one note of each relation in turn, in the order of the relations', () => {
    const turns = takeTurns({
      sibling: [node('s0', 'sibling')],
      child: [node('c0', 'child'), node('c1', 'child'), node('c2', 'child')],
      parent: [node('p0', 'parent'), node('p1', 'parent')],
      jump: [node('j0', 'jump')],
    })
    expect(turns.map((each) => each.id)).toStrictEqual(['p0', 'c0', 'j0', 's0', 'p1', 'c1', 'c2'])
  })
})

describe('what comes back from the placement', () => {
  const focus: PlacedNode = {
    ...node('focus', 'focus'),
    x: 0,
    y: 0,
    width: 176,
    height: 44,
    order: 0,
    opacity: 1,
  }
  const placed = clusters.place(
    {
      sibling: [node('s0', 'sibling')],
      child: [node('c0', 'child'), node('c1', 'child')],
      parent: [node('p0', 'parent')],
    },
    focus,
    { options: DEFAULT_OPTIONS, width: () => 120 },
  )

  it('is called clusters', () => {
    expect(clusters.name).toBe('clusters')
  })

  it('gives the nodes grouped by relation in the order of the relations, each in the order it arrived', () => {
    expect(placed.map((each) => each.id)).toStrictEqual(['p0', 'c0', 'c1', 's0'])
  })

  it('draws each box as wide as the width lookup says and as tall as a neighbor, fully visible', () => {
    for (const each of placed) {
      expect(each.width).toBe(120)
      expect(each.height).toBe(DEFAULT_OPTIONS.nodeSize.height)
      expect(each.opacity).toBe(1)
    }
  })

  it('numbers the nodes of a relation by arrival', () => {
    expect(
      placed.filter((each) => each.relation === 'child').map((each) => each.order),
    ).toStrictEqual([0, 1])
  })
})
