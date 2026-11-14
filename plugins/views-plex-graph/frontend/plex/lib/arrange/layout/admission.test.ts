/** How much of a graph the window holds: cut before the first box outside, halved when even the first is outside. */
import { describe, expect, it } from 'vitest'
import { admitNodes, countOverflow } from './admission'
import type { NodesByRelation } from './placement'
import { createRandom } from '../../../fixtures/random'
import { RELATION_NAMES } from '../../relation'
import type { PlacedNode } from '../../node'
import type { PlexNode, PlexRelation } from '../../../../types'

const FIRST_BOX_AT_LEAST = 100
const STEP = 10
/** A box is inside the window while it is no deeper than this. */
const DEEPEST = 200

const createNodes = (relation: PlexRelation, count: number, prefix = relation): PlexNode[] =>
  Array.from({ length: count }, (_, index) => ({
    id: `${prefix}-${index}`,
    title: `${prefix} ${index}`,
    relation,
  }))

const toPlaced = (node: PlexNode, y: number, order: number): PlacedNode => ({
  ...node,
  x: 0,
  y,
  width: 100,
  height: 30,
  order,
  opacity: 1,
})

const lay = (nodesByRelation: NodesByRelation, shouldStop: (node: PlacedNode) => boolean) => {
  const placed: PlacedNode[] = []
  const queues = RELATION_NAMES.map((relation) => nodesByRelation[relation] ?? [])
  const rounds = Math.max(0, ...queues.map((queue) => queue.length))
  for (let round = 0; round < rounds; round++) {
    for (const queue of queues) {
      const node = queue[round]
      if (!node) continue
      const y = FIRST_BOX_AT_LEAST + queue.length * STEP + round * STEP
      const box = toPlaced(node, y, round)
      placed.push(box)
      if (shouldStop(box)) return placed
    }
  }
  return placed
}

const isInside = (node: PlacedNode) => node.y <= DEEPEST

const admit = (requested: NodesByRelation) => admitNodes(requested, lay, isInside)

const countOf = (nodes: NodesByRelation, relation: PlexRelation) => nodes[relation]?.length ?? 0

describe('the boundary of what a window holds', () => {
  // The first box of n notes lies at 100 + 10n and the last at 100 + 20n - 10, so five notes are held whole and the sixth box is outside.
  const WHOLE = 5
  // At ten notes the first box is still inside, at eleven it is outside.
  const FIRST_INSIDE = 10

  it('admits the whole group at the number of notes whose last box is inside', () => {
    const requested = createNodes('child', WHOLE)

    expect(admit({ child: requested }).child).toStrictEqual(requested)
  })

  // Six notes: the sixth box is outside, the group is cut before it. Ten: the second box is outside, one note is held and the notes are taken back. Eleven and more: the first box is outside, the group is halved and the notes are taken back.
  it.each([WHOLE + 1, FIRST_INSIDE, FIRST_INSIDE + 1, FIRST_INSIDE + 3, 40])(
    'admits the notes the window holds, and no more, of a group of %i',
    (count) => {
      const requested = createNodes('child', count)

      expect(admit({ child: requested }).child).toStrictEqual(requested.slice(0, WHOLE))
    },
  )

  it('keeps the order the notes arrived in, as a prefix', () => {
    const requested = createNodes('child', 30)

    const visible = admit({ child: requested }).child ?? []

    expect(visible.length).toBeGreaterThan(0)
    expect(visible).toStrictEqual(requested.slice(0, visible.length))
  })
})

describe('the order of the notes inside a relation', () => {
  const random = createRandom(1837)
  const shuffle = <T>(items: readonly T[]): T[] => {
    const shuffled = [...items]
    for (let at = shuffled.length - 1; at > 0; at--) {
      const other = Math.floor(random() * (at + 1))
      ;[shuffled[at], shuffled[other]] = [shuffled[other]!, shuffled[at]!]
    }
    return shuffled
  }

  it('does not change how many notes of each relation are shown', () => {
    for (let round = 0; round < 60; round++) {
      const requested: NodesByRelation = {
        parent: createNodes('parent', Math.floor(random() * 25)),
        child: createNodes('child', Math.floor(random() * 25)),
        jump: createNodes('jump', Math.floor(random() * 25)),
      }
      const permuted: NodesByRelation = {
        parent: shuffle(requested.parent ?? []),
        child: shuffle(requested.child ?? []),
        jump: shuffle(requested.jump ?? []),
      }

      const counts = (nodes: NodesByRelation) =>
        RELATION_NAMES.map((relation) => countOf(nodes, relation))

      expect(counts(admit(permuted)), `round ${round}`).toStrictEqual(counts(admit(requested)))
    }
  })

  it('shows the notes that come first, wherever they are in the set', () => {
    const requested = createNodes('child', 25)
    const permuted = shuffle(requested)

    const visible = admit({ child: permuted }).child ?? []

    expect(visible).toStrictEqual(permuted.slice(0, visible.length))
  })
})

describe('the count of notes left out', () => {
  it('is what arrived less what was admitted, for each relation with any left', () => {
    const nodes = [...createNodes('parent', 4), ...createNodes('child', 7)]
    const admitted: NodesByRelation = {
      parent: nodes.slice(0, 4),
      child: nodes.filter((node) => node.relation === 'child').slice(0, 3),
    }

    expect(countOverflow(nodes, admitted)).toStrictEqual({ child: 4 })
  })
})
