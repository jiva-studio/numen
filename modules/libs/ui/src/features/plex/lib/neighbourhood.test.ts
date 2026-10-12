/** The two invariants every neighbourhood holds. */
import { describe, expect, it } from 'vitest'
import { assertNeighbourhood } from './neighbourhood'
import type { PlexNode } from './node'
import type { PlexSeat } from './seat'

const node = (id: string, seat: PlexSeat): PlexNode => ({ id, title: id, seat })

describe('assertNeighbourhood', () => {
  it('gives back the node with the focus seat', () => {
    const focus = node('a', 'focus')
    expect(assertNeighbourhood({ nodes: [node('b', 'child'), focus], edges: [] })).toBe(focus)
  })

  it('throws where no node has the focus seat', () => {
    expect(() => assertNeighbourhood({ nodes: [node('b', 'child')], edges: [] })).toThrow(
      'needs a node with the focus seat',
    )
  })

  it('names every node that claims the focus, and how many there are', () => {
    const nodes = [node('a', 'focus'), node('b', 'focus'), node('c', 'focus')]
    expect(() => assertNeighbourhood({ nodes, edges: [] })).toThrow(
      'has one focus, not 3: a, b, c',
    )
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
