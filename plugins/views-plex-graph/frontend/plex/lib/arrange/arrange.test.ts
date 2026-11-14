/** The arrangement, tested without rendering it. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { graphs } from '../../fixtures/graphs'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import { clusters } from './layout/clusters'
import type { Placement } from './layout/placement'
import { build } from '../../fixtures/build'
import type { PlexGraph, PlexNode, PlexRelation, PlexNodeRelation } from '../../../types'

const every = Object.entries(graphs)

const byTitle = (node: PlexNode): number => 40 + node.title.length * 7

const createGraph = (relation: PlexRelation, ...titles: string[]): PlexGraph => ({
  nodes: [
    { id: 'focus', title: 'Focus', relation: 'focus' },
    ...titles.map((title, index) => ({ id: `${relation}-${index}`, title, relation })),
  ],
  edges: [],
})

const getNodesByRelation = (layout: { nodes: readonly PlacedNode[] }, relation: PlexNodeRelation) =>
  layout.nodes.filter((node) => node.relation === relation)

const getFocus = (layout: { nodes: readonly PlacedNode[] }) => {
  const focus = layout.nodes.find((node) => node.relation === 'focus')
  if (!focus) throw new Error('unreachable: every layout places a focus')
  return focus
}

describe('the contract on the graph', () => {
  it('refuses a graph with no focus', () => {
    const nodes = [{ id: 'a', title: 'A', relation: 'child' as const }]
    expect(() => arrangePlex({ nodes, edges: [] })).toThrow(/needs a node with the focus/)
  })

  it('refuses a graph with two nodes in focus', () => {
    const nodes = [
      { id: 'a', title: 'A', relation: 'focus' as const },
      { id: 'b', title: 'B', relation: 'focus' as const },
    ]
    expect(() => arrangePlex({ nodes, edges: [] })).toThrow(/one focus, not 2/)
  })

  it('refuses the same identifier in two relations', () => {
    const nodes = [
      { id: 'a', title: 'A', relation: 'focus' as const },
      { id: 'b', title: 'B', relation: 'parent' as const },
      { id: 'b', title: 'B again', relation: 'child' as const },
    ]
    expect(() => arrangePlex({ nodes, edges: [] })).toThrow(/appears more than once: b/)
  })
})

describe('where the relations sit', () => {
  const layout = arrangePlex(graphs.typical)
  const focus = getFocus(layout)

  it('puts the focus at the origin', () => {
    expect(focus.x).toBe(0)
    expect(focus.y).toBe(0)
  })

  it('puts every parent clear above the focus', () => {
    const parents = getNodesByRelation(layout, 'parent')
    expect(parents.length).toBeGreaterThan(0)
    for (const parent of parents) {
      expect(parent.y + parent.height / 2).toBeLessThan(-focus.height / 2)
    }
  })

  it('puts every child clear below the focus', () => {
    const children = getNodesByRelation(layout, 'child')
    expect(children.length).toBeGreaterThan(0)
    for (const child of children) {
      expect(child.y - child.height / 2).toBeGreaterThan(focus.height / 2)
    }
  })

  it('puts every jump clear to the left and every sibling clear to the right', () => {
    for (const jump of getNodesByRelation(layout, 'jump')) {
      expect(jump.x + jump.width / 2).toBeLessThan(-focus.width / 2)
    }
    for (const sibling of getNodesByRelation(layout, 'sibling')) {
      expect(sibling.x - sibling.width / 2).toBeGreaterThan(focus.width / 2)
    }
    expect(getNodesByRelation(layout, 'jump').length).toBeGreaterThan(0)
    expect(getNodesByRelation(layout, 'sibling').length).toBeGreaterThan(0)
  })
})

describe('a box is as wide as its title needs', () => {
  const { minWidth, nodeSize, focusSize, focusScale } = DEFAULT_OPTIONS

  it('draws every box at its widest when nothing was measured', () => {
    const layout = arrangePlex(graphs.typical)
    expect(getFocus(layout).width).toBe(focusSize.width)
    for (const node of getNodesByRelation(layout, 'child')) expect(node.width).toBe(nodeSize.width)
  })

  it('gives each box the width it was measured at', () => {
    const measurable = createGraph('child', 'Domain', 'Use case', 'Aggregate', 'Repository')
    const layout = arrangePlex(measurable, { measure: byTitle })
    for (const node of getNodesByRelation(layout, 'child')) expect(node.width).toBe(byTitle(node))
  })

  it('sizes the focus to its title as any other box, made larger by the focus scale', () => {
    const layout = arrangePlex(createGraph('child', 'Port'), { measure: byTitle })
    const focus = getFocus(layout)
    expect(focus.width).toBeCloseTo(byTitle(focus) * focusScale, 6)
    expect(focus.width).toBeLessThan(focusSize.width)
  })

  it('keeps a measured title whole in a crowded window, up to the widest box', () => {
    const crowd = build('A note', { parent: 4, child: 12, jump: 4, sibling: 4 })
    const viewport = { width: 560, height: 1000 }
    const layout = arrangePlex(crowd, { options: { viewport }, measure: () => 120 })
    const shown = layout.nodes.filter((node) => node.relation !== 'focus')

    expect(shown.length).toBeGreaterThan(0)
    for (const node of shown) expect(node.width).toBe(120)
  })

  it('clamps a title past the maximum, which each relation has its own of', () => {
    const layout = arrangePlex(graphs.typical, { measure: () => 4000 })
    expect(getFocus(layout).width).toBe(focusSize.width)
    for (const node of getNodesByRelation(layout, 'child')) expect(node.width).toBe(nodeSize.width)
  })

  it('gives a title that measures next to nothing the minimum', () => {
    const layout = arrangePlex(createGraph('child', '', 'A'), { measure: () => 0 })
    expect(getFocus(layout).width).toBeCloseTo(minWidth * focusScale, 6)
    for (const node of getNodesByRelation(layout, 'child')) expect(node.width).toBe(minWidth)
  })

  it('centers the first row of a group on the axis of its side, whatever the widths', () => {
    const mixed = createGraph('child', 'A', 'Domain', 'Composition root', 'Port', 'Use case')
    const children = getNodesByRelation(arrangePlex(mixed, { measure: byTitle }), 'child')
    const firstRow = children.filter((node) => node.y === children[0]!.y).map((node) => node.x)

    expect(new Set(children.map((node) => node.width)).size).toBeGreaterThan(1)
    expect(Math.min(...firstRow) + Math.max(...firstRow)).toBeCloseTo(0, 9)
  })

  it('shows as much as it would have shown unmeasured', () => {
    // How much is admitted is settled from the widest a box gets, so what a
    // title measures never changes the count or what the overflow reports.
    const measured = arrangePlex(graphs.overcrowded, { measure: byTitle })
    const defaultLayout = arrangePlex(graphs.overcrowded)

    expect(measured.overflow).toStrictEqual(defaultLayout.overflow)
    expect(measured.nodes.map((node) => node.id)).toStrictEqual(
      defaultLayout.nodes.map((node) => node.id),
    )
  })
})

describe('a label is as long as its words', () => {
  /** One line, running sideways, with words written along it. */
  const labelled: PlexGraph = {
    nodes: [
      { id: 'focus', title: 'Here', relation: 'focus' },
      { id: 'aside', title: 'Aside', relation: 'jump' },
    ],
    edges: [{ from: 'aside', to: 'focus', labels: [{ text: 'the scene in the assembly' }] }],
  }

  const getWords = (measureLabel?: (label: string) => number) =>
    arrangePlex(labelled, { measureLabel }).edges[0]!.titles[0]?.words

  it('asks the measurer for the words a line carries, and for no others', () => {
    const request: string[] = []
    arrangePlex(labelled, {
      measureLabel: (label) => {
        request.push(label)
        return 0
      },
    })

    // Called once to truncate the text to the line, and once to find a position
    // for it along the line.
    expect(request).toStrictEqual(['the scene in the assembly', 'the scene in the assembly'])
  })

  it('cuts words too long for their line', () => {
    const words = getWords((label) => 6 * [...label].length)
    expect(words).toMatch(/^the .+…$/)
    expect(words!.length).toBeLessThan('the scene in the assembly'.length)
  })

  it('writes no words at all where the line holds none of them', () => {
    expect(getWords(() => 4000)).toBeUndefined()
  })

  it('leaves words that fit as they were written', () => {
    expect(getWords(() => 1)).toBe('the scene in the assembly')
  })

  it('draws the whole label where nothing measured it', () => {
    expect(getWords()).toBe('the scene in the assembly')
  })
})

describe('order within a group follows the order given', () => {
  it('numbers each relation outward from the focus', () => {
    const children = getNodesByRelation(arrangePlex(graphs.crowded), 'child')
    expect(children.map((node) => node.order)).toStrictEqual(children.map((_, index) => index))
  })

  it('keeps the relative order when one more node arrives', () => {
    const before = graphs.typical
    const after: PlexGraph = {
      nodes: [...before.nodes, { id: 'child-6', title: 'One more', relation: 'child' }],
      edges: [...before.edges, { from: 'focus', to: 'child-6' }],
    }
    const ids = (n: PlexGraph) => getNodesByRelation(arrangePlex(n), 'child').map((node) => node.id)
    expect(ids(after).slice(0, ids(before).length)).toStrictEqual(ids(before))
  })
})

describe('what does not fit is reported and not dropped silently', () => {
  it('counts the nodes past the cap of a group, per relation', () => {
    const capacity = DEFAULT_OPTIONS.maxPerCluster
    const { overflow, nodes } = arrangePlex(graphs.overcrowded)

    expect(overflow.child).toBe(200 - capacity)
    expect(overflow.jump).toBe(40 - capacity)
    expect(overflow.parent).toBeUndefined() // nine parents are under the cap
    expect(nodes.filter((node) => node.relation === 'child')).toHaveLength(capacity)
  })

  it('reports nothing when everything fits', () => {
    expect(arrangePlex(graphs.typical).overflow).toStrictEqual({})
  })

  it('drops the edges that lead to a node which did not fit', () => {
    const { nodes, edges } = arrangePlex(graphs.overcrowded)
    const placedIds = new Set(nodes.map((node) => node.id))
    for (const edge of edges) {
      expect(placedIds.has(edge.from)).toBe(true)
      expect(placedIds.has(edge.to)).toBe(true)
    }
    expect(edges.length).toBeLessThan(graphs.overcrowded.edges.length)
  })
})

describe('the extent', () => {
  const hasEveryNode = ({ nodes, extent }: PlexFrame) => {
    for (const node of nodes) {
      expect(node.x - node.width / 2).toBeGreaterThanOrEqual(extent.minX)
      expect(node.x + node.width / 2).toBeLessThanOrEqual(extent.maxX)
      expect(node.y - node.height / 2).toBeGreaterThanOrEqual(extent.minY)
      expect(node.y + node.height / 2).toBeLessThanOrEqual(extent.maxY)
    }
  }

  it.each(every)('contains every node of %s', (_name, graph: PlexGraph) => {
    hasEveryNode(arrangePlex(graph))
  })

  it.each(every)('contains every node of %s, sized to its titles', (_name, graph: PlexGraph) => {
    hasEveryNode(arrangePlex(graph, { measure: byTitle }))
  })

  it('is the focus box alone when the note remains alone', () => {
    const { extent } = arrangePlex(graphs.solitary)
    const { width, height } = DEFAULT_OPTIONS.focusSize
    expect(extent).toStrictEqual({
      minX: -width / 2,
      minY: -height / 2,
      maxX: width / 2,
      maxY: height / 2,
    })
  })
})

describe('the options are honoured', () => {
  it('lets the caller change how many are shown in a group', () => {
    const narrow = arrangePlex(graphs.crowded, { options: { maxPerCluster: 3 } })
    expect(getNodesByRelation(narrow, 'child')).toHaveLength(3)
  })

  it('lets the caller send children up and parents down', () => {
    const upside = arrangePlex(graphs.typical, {
      options: {
        direction: {
          parent: 'down',
          child: 'up',
          jump: 'left',
          reference: 'left',
          sibling: 'right',
        },
      },
    })
    for (const child of getNodesByRelation(upside, 'child')) expect(child.y).toBeLessThan(0)
    for (const parent of getNodesByRelation(upside, 'parent')) expect(parent.y).toBeGreaterThan(0)
  })

  it('takes a curve setting without receiving the size of a box', () => {
    // Tuning a curve should not need to know the size of a box.
    const straighter = arrangePlex(graphs.typical, {
      options: { routing: { curvature: 0.01 } },
    })
    const edge = straighter.edges[0]!
    expect(edge.control1.y - edge.fromPoint.y).toBe(DEFAULT_OPTIONS.routing.minReach)
  })
})

describe('the arrangement is a strategy, not a shape baked in', () => {
  it('uses clusters unless told otherwise', () => {
    const frame = arrangePlex(graphs.typical)
    expect(getNodesByRelation(frame, 'parent').every((n) => n.y < 0)).toBe(true)
  })

  it('lets another strategy place the nodes, and routes its edges anyway', () => {
    // The point is not the ring: a strategy sets coordinates and inherits
    // the limits, the routing and the extent.
    const ring: Placement = {
      name: 'ring',
      place(nodesByRelation) {
        const all = Object.values(nodesByRelation).flat()
        return all.map((node, index) => {
          const angle = (index / all.length) * Math.PI * 2
          return {
            ...node,
            x: Math.round(Math.cos(angle) * 300),
            y: Math.round(Math.sin(angle) * 300),
            width: 144,
            height: 36,
            order: index,
            opacity: 1,
          }
        })
      },
    }

    const frame = arrangePlex(graphs.typical, { placement: ring })
    const focus = getFocus(frame)

    expect(focus.x).toBe(0)
    expect(focus.y).toBe(0)
    expect(frame.nodes).toHaveLength(graphs.typical.nodes.length)
    // Edges still meet their boxes, without the strategy knowing they exist.
    expect(frame.edges).toHaveLength(graphs.typical.edges.length)
    for (const edge of frame.edges) {
      expect(Number.isFinite(edge.control1.x)).toBe(true)
    }
  })

  it('does not let a strategy decide how much is shown', () => {
    // Admission is settled by the window and the cap, so the count is the same
    // whichever arrangement drew it.
    const countPlacement: Placement = {
      name: 'counting',
      place: (nodesByRelation) =>
        Object.values(nodesByRelation)
          .flat()
          .map((node, index) => ({
            ...node,
            x: index * 200,
            y: 100,
            width: 144,
            height: 36,
            order: index,
            opacity: 1,
          })),
    }

    const frame = arrangePlex(graphs.overcrowded, { placement: countPlacement })
    expect(frame.overflow.child).toBe(200 - DEFAULT_OPTIONS.maxPerCluster)
  })

  it('passes a strategy only the relations that have nodes in them', () => {
    const handed: string[][] = []
    const recording: Placement = {
      name: 'recording',
      place: (nodesByRelation, focus, context) => {
        handed.push(Object.keys(nodesByRelation))
        return clusters.place(nodesByRelation, focus, context)
      },
    }

    const frame = arrangePlex(build('A node', { child: 2, jump: 1 }), { placement: recording })

    expect(handed.map((relations) => [...relations].sort())).toStrictEqual([['child', 'jump']])
    expect(frame.overflow).toStrictEqual({})
  })
})

describe('a window that holds the arrangement exactly', () => {
  it('opens the gaps that have room, the height being met to the pixel', () => {
    const { nodeSize, gap, margin } = DEFAULT_OPTIONS
    const two = createGraph('child', 'One', 'Two')
    const settings = getNodesByRelation(arrangePlex(two), 'child')
    const bottom = Math.max(...settings.map((node) => node.y + node.height / 2))
    const viewport = { width: 2000, height: 2 * (bottom + margin) }

    const frame = arrangePlex(two, { options: { viewport } })
    const [first, second] = getNodesByRelation(frame, 'child')

    expect(Math.max(...[first!, second!].map((node) => node.y + node.height / 2))).toBeCloseTo(
      bottom,
    )
    expect(Math.abs(second!.x - first!.x)).toBeGreaterThan(nodeSize.width + gap)
  })
})
