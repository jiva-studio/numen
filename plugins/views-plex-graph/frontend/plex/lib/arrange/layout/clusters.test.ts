/** The result of the clusters placement. */
import { describe, expect, it } from 'vitest'
import { clusters } from './clusters'
import { DEFAULT_OPTIONS } from '../options'
import { NO_ROOM } from './placement'
import type { PlacedNode } from '../../node'
import type { PlexNode } from '../../../../types'

const node = (id: string, relation: PlexNode['relation']): PlexNode => ({ id, title: id, relation })

describe('the result of the placement', () => {
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
    { options: DEFAULT_OPTIONS, width: () => 120, room: () => NO_ROOM },
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
