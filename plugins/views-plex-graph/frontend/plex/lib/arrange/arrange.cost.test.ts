/** How many layouts arranging a large graph takes: admission and crowding are probes, and each probe is one layout. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { clusters } from './layout/clusters'
import type { Placement } from './layout/placement'
import type { PlexEdge, PlexGraph, PlexNode, PlexRelation } from '../../../types'

const RELATIONS: readonly PlexRelation[] = ['parent', 'child', 'jump', 'reference', 'sibling']

const createGraph = (count: number): PlexGraph => {
  const nodes: PlexNode[] = [{ id: 'focus', title: 'Focus', relation: 'focus' }]
  const edges: PlexEdge[] = []
  for (let index = 0; index < count; index++) {
    const relation = RELATIONS[index % RELATIONS.length]!
    const id = `${relation}-${index}`
    nodes.push({ id, title: id, relation })
    edges.push(relation === 'parent' ? { from: id, to: 'focus' } : { from: 'focus', to: id })
  }
  return { nodes, edges }
}

const arrangeCounting = (graph: PlexGraph) => {
  let layouts = 0
  const counting: Placement = {
    name: clusters.name,
    place: (nodes, focus, context) => {
      layouts++
      return clusters.place(nodes, focus, context)
    },
  }
  const frame = arrangePlex(graph, {
    placement: counting,
    options: { viewport: { width: 1000, height: 600 } },
    measure: (node) => 60 + (node.id.length % 7) * 20,
    room: (node) => ({ top: 0, bottom: node.id.length % 2 === 0 ? 14 : 0, width: 100 }),
  })
  return { frame, layouts }
}

describe('the number of layouts', () => {
  it('stays bounded for a graph of 200 notes in a window that holds a fraction of it', () => {
    const { frame, layouts } = arrangeCounting(createGraph(200))
    console.log('LAYOUTS 200', layouts, frame.nodes.length)
    expect(Object.keys(frame.overflow).length).toBeGreaterThan(0)
    expect(layouts).toBeLessThanOrEqual(60)
  })

  it('stays bounded when only the visible notes of that graph are arranged again', () => {
    const graph = createGraph(200)
    const visibleIds = new Set(arrangeCounting(graph).frame.nodes.map((node) => node.id))
    const { frame, layouts } = arrangeCounting({
      nodes: graph.nodes.filter((node) => visibleIds.has(node.id)),
      edges: graph.edges.filter((edge) => visibleIds.has(edge.from) && visibleIds.has(edge.to)),
    })
    console.log('LAYOUTS shown', layouts)
    expect(Object.keys(frame.overflow)).toStrictEqual([])
    expect(layouts).toBeLessThanOrEqual(60)
  })
})
