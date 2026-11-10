/** The two invariants every neighbourhood holds. */
import { describe, expect, it } from 'vitest'
import { assertNeighbourhood } from './neighbourhood'
import type { PlexNode } from './node'
import type { PlexNodeRelation } from './relation'

const node = (id: string, relation: PlexNodeRelation): PlexNode => ({ id, title: id, relation })

describe('assertNeighbourhood', () => {
  it('gives back the node with the focus relation', () => {
    const focus = node('a', 'focus')
    expect(assertNeighbourhood({ nodes: [node('b', 'child'), focus], edges: [] })).toBe(focus)
  })

  it('throws where no node has the focus relation', () => {
    expect(() => assertNeighbourhood({ nodes: [node('b', 'child')], edges: [] })).toThrow(
      'needs a node with the focus relation',
    )
  })

  it('names every node that claims the focus, and how many there are', () => {
    const nodes = [node('a', 'focus'), node('b', 'focus'), node('c', 'focus')]
    expect(() => assertNeighbourhood({ nodes, edges: [] })).toThrow('has one focus, not 3: a, b, c')
  })

  it('names every node that appears more than once', () => {
    const nodes = [
      node('f', 'focus'),
      node('x', 'child'),
      node('y', 'child'),
      node('x', 'parent'),
      node('y', 'parent'),
    ]
    expect(() => assertNeighbourhood({ nodes, edges: [] })).toThrow('more than once: x, y')
  })
})
