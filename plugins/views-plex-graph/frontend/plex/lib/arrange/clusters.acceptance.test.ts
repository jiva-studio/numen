/** Each relation grows as a compact cluster from its side of the focus, into whatever space is left. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getBox, isOverlapping } from './layout/boxes'
import { measureCrowding, packOptions } from './layout/crowding'
import { DEFAULT_OPTIONS, resolveOptions, type PlexOptions, type PlexOptionsInput } from './options'
import { graphs } from '../../fixtures/graphs'
import { build } from '../../fixtures/build'
import { ring } from '../../fixtures/ring'
import { RELATION_NAMES } from '../relation'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import type { Placement } from './layout/placement'
import type { PlexGraph, PlexNode, PlexNodeRelation, PlexRelation } from '../../../types'

type Input = PlexOptionsInput & { maxPerCluster?: number }
type Window = { width: number; height: number }

const byTitle = (node: PlexNode): number => 40 + node.title.length * 7
const { margin } = DEFAULT_OPTIONS

const arrange = (
  graph: PlexGraph,
  options: Input = {},
  extra: { measure?: (node: PlexNode) => number; placement?: Placement } = {},
): PlexFrame => arrangePlex(graph, { options: options as PlexOptionsInput, ...extra })

const inWindow = (viewport: Window, options: Input = {}) => ({ ...options, viewport })

const getFocus = (frame: PlexFrame): PlacedNode => frame.nodes.find((n) => n.relation === 'focus')!
const getShownNodes = (frame: PlexFrame, relation: PlexNodeRelation): PlacedNode[] =>
  frame.nodes.filter((node) => node.relation === relation)
const getArrivedNodes = (graph: PlexGraph, relation: PlexRelation): PlexNode[] =>
  graph.nodes.filter((node) => node.relation === relation)
const mean = (values: readonly number[]): number =>
  values.reduce((sum, value) => sum + value, 0) / values.length

const getOverlaps = (frame: PlexFrame): string[] =>
  frame.nodes.flatMap((a, index) =>
    frame.nodes
      .slice(index + 1)
      .filter((b) => isOverlapping(getBox(a, 0), getBox(b, 0)))
      .map((b) => `${a.id} / ${b.id}`),
  )

const getOutside = (nodes: readonly PlacedNode[], viewport: Window): string[] =>
  nodes
    .filter(
      (node) =>
        Math.abs(node.x) + node.width / 2 > viewport.width / 2 - margin + 1e-9 ||
        Math.abs(node.y) + node.height / 2 > viewport.height / 2 - margin + 1e-9,
    )
    .map((node) => node.id)

function countDistinctValues(values: readonly number[], tolerance: number): number {
  const sorted = [...values].sort((a, b) => a - b)
  return sorted.reduce(
    (count, value, index) =>
      index === 0 || value - sorted[index - 1]! > tolerance ? count + 1 : count,
    0,
  )
}

function getSmallestSeparation(nodes: readonly PlacedNode[]): number {
  let smallest = Infinity
  nodes.forEach((a, index) => {
    for (const b of nodes.slice(index + 1)) {
      const sideways = Math.abs(a.x - b.x) - (a.width + b.width) / 2
      const lengthways = Math.abs(a.y - b.y) - (a.height + b.height) / 2
      smallest = Math.min(smallest, Math.max(sideways, lengthways))
    }
  })
  return smallest
}

const WINDOWS: readonly (Window | undefined)[] = [
  undefined,
  { width: 2400, height: 1400 },
  { width: 1200, height: 800 },
  { width: 700, height: 600 },
  { width: 420, height: 320 },
]

describe('no two boxes overlap', () => {
  const cases = Object.entries(graphs).flatMap(([name, graph]) =>
    [undefined, byTitle].flatMap((measure) =>
      [0, 20].flatMap((room) =>
        WINDOWS.map((viewport) => ({ name, graph, measure, room, viewport })),
      ),
    ),
  )

  it.each(cases)(
    '$name, measured $measure, room $room, window $viewport',
    ({ graph, measure, room, viewport }) => {
      const frame = arrange(
        graph,
        { room, ...(viewport ? { viewport } : {}) },
        measure ? { measure } : {},
      )
      expect(getOverlaps(frame)).toStrictEqual([])
    },
  )
})

describe('each relation is placed on its own side of the focus', () => {
  const full = build('A note', { parent: 8, child: 20, jump: 8, sibling: 8 })
  const window = { width: 1600, height: 1000 }

  it.each([
    ['typical', graphs.typical],
    ['eight, twenty, eight and eight', full],
  ])('puts every neighbor on its side in %s', (_name, graph) => {
    const frame = arrange(graph, inWindow(window))
    const focus = getFocus(frame)
    const top = focus.y - focus.height / 2
    const bottom = focus.y + focus.height / 2
    const left = focus.x - focus.width / 2
    const right = focus.x + focus.width / 2

    expect(getShownNodes(frame, 'parent').length).toBeGreaterThan(0)
    for (const node of getShownNodes(frame, 'parent'))
      expect(node.y + node.height / 2).toBeLessThan(top)
    for (const node of getShownNodes(frame, 'child'))
      expect(node.y - node.height / 2).toBeGreaterThan(bottom)
    for (const node of getShownNodes(frame, 'sibling'))
      expect(node.x - node.width / 2).toBeGreaterThan(right)
    for (const node of getShownNodes(frame, 'jump'))
      expect(node.x + node.width / 2).toBeLessThan(left)
  })

  it.each([
    ['typical', graphs.typical],
    ['eight, twenty, eight and eight', full],
  ])('centers the jumps on the row of the focus in %s', (_name, graph) => {
    const frame = arrange(graph, inWindow(window))
    const { nodeSize, lineGap } = DEFAULT_OPTIONS
    expect(
      Math.abs(mean(getShownNodes(frame, 'jump').map((node) => node.y)) - getFocus(frame).y),
    ).toBeLessThan(nodeSize.height + lineGap)
  })

  it('leans the jumps down and the references up where they share the left side', () => {
    const frame = arrange(build('A note', { jump: 6, reference: 6 }), inWindow(window))
    const focusY = getFocus(frame).y
    expect(mean(getShownNodes(frame, 'jump').map((node) => node.y))).toBeGreaterThan(focusY)
    expect(mean(getShownNodes(frame, 'reference').map((node) => node.y))).toBeLessThan(focusY)
  })
})

describe('a group is never nearer the focus than its start', () => {
  const DIRECTIONS: Record<PlexRelation, readonly [number, number]> = {
    parent: [0, -1],
    child: [0, 1],
    jump: [-1, 0],
    reference: [-1, 0],
    sibling: [1, 0],
  }
  const along = (node: PlacedNode, [dx, dy]: readonly [number, number]) => node.x * dx + node.y * dy

  it.each(
    RELATION_NAMES.flatMap((relation) => [1, 4, 9, 20].map((count) => ({ relation, count }))),
  )('$count $relation: no node is nearer than the first one alone', ({ relation, count }) => {
    const alone = getShownNodes(arrange(build('A note', { [relation]: 1 })), relation)[0]!
    const group = getShownNodes(arrange(build('A note', { [relation]: count })), relation)
    expect(group).toHaveLength(count)
    const start = along(alone, DIRECTIONS[relation])
    for (const node of group)
      expect(along(node, DIRECTIONS[relation])).toBeGreaterThanOrEqual(start - 0.5)
  })
})

describe('a large group is a block, not a line', () => {
  it.each([undefined, { width: 1600, height: 1000 }])(
    'stacks nine children in rows of at most five, window %o',
    (viewport) => {
      const frame = arrange(build('A note', { child: 9 }), viewport ? { viewport } : {})
      const rows = new Map<number, number>()
      for (const child of getShownNodes(frame, 'child')) {
        const row = Math.round(child.y)
        rows.set(row, (rows.get(row) ?? 0) + 1)
      }
      expect(rows.size).toBeGreaterThanOrEqual(2)
      expect(Math.max(...rows.values())).toBeLessThanOrEqual(5)
    },
  )

  it.each([undefined, { width: 1600, height: 1000 }])(
    'spreads nine siblings over two columns or more, window %o',
    (viewport) => {
      const frame = arrange(build('A note', { sibling: 9 }), viewport ? { viewport } : {})
      expect(
        countDistinctValues(
          getShownNodes(frame, 'sibling').map((node) => node.x),
          10,
        ),
      ).toBeGreaterThanOrEqual(2)
    },
  )
})

describe('a larger group is placed further out', () => {
  it('moves the middle of the children away from the focus as they grow in number', () => {
    const middles = [1, 4, 9, 16].map((child) =>
      mean(getShownNodes(arrange(build('A note', { child })), 'child').map((node) => node.y)),
    )
    for (let i = 1; i < middles.length; i++) expect(middles[i]!).toBeGreaterThan(middles[i - 1]!)
  })
})

describe('groups take turns', () => {
  it('leaves the first note of another group where it would be alone, however many children there are', () => {
    const crowd = arrange(build('A note', { child: 20, jump: 1, sibling: 1, parent: 1 }))
    const alone = arrange(build('A note', { jump: 1, sibling: 1, parent: 1 }))
    for (const id of ['focus/parent-0', 'focus/jump-0', 'focus/sibling-0']) {
      const one = crowd.nodes.find((node) => node.id === id)!
      const other = alone.nodes.find((node) => node.id === id)!
      expect([id, one.x, one.y]).toStrictEqual([id, other.x, other.y])
    }
  })
})

describe('a group fills the free space next to another group', () => {
  const window = { width: 1600, height: 1000 }

  it('shows twenty-four children and spreads them sideways instead of running down one column', () => {
    const frame = arrange(build('A note', { child: 24 }), inWindow(window))
    const focus = getFocus(frame)
    const children = getShownNodes(frame, 'child')
    expect(frame.overflow.child).toBeUndefined()
    expect(children).toHaveLength(24)
    expect(
      children.some((child) => Math.abs(child.x - focus.x) > focus.width / 2 + child.width),
    ).toBe(true)
  })

  it('shows the children and the jumps without overlapping a box', () => {
    const frame = arrange(build('A note', { child: 16, jump: 16 }), inWindow(window))
    const children = getShownNodes(frame, 'child')
    const jumps = getShownNodes(frame, 'jump')
    expect(children.length).toBeGreaterThan(0)
    expect(jumps.length).toBeGreaterThan(0)
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe('a long group spreads instead of running into one column', () => {
  it('shows at least twenty of thirty jumps over three columns and three rows', () => {
    const frame = arrange(build('A note', { jump: 30 }), inWindow({ width: 1600, height: 1000 }))
    const jumps = getShownNodes(frame, 'jump')
    expect(jumps.length).toBeGreaterThanOrEqual(20)
    expect(
      countDistinctValues(
        jumps.map((node) => node.x),
        10,
      ),
    ).toBeGreaterThanOrEqual(3)
    expect(
      countDistinctValues(
        jumps.map((node) => node.y),
        10,
      ),
    ).toBeGreaterThanOrEqual(3)
  })
})

describe('what is left out is counted, and only the last notes go', () => {
  const tiny = { width: 400, height: 400 }
  const many = build('A note', { parent: 9, child: 40, jump: 9, sibling: 9 })

  it.each([
    ['overcrowded', graphs.overcrowded],
    ['nine, forty, nine and nine', many],
  ])('shows the first of each relation in arrival order in %s', (_name, graph) => {
    const frame = arrange(graph, inWindow(tiny))
    for (const relation of RELATION_NAMES) {
      const arrived = getArrivedNodes(graph, relation).map((node) => node.id)
      const shown = getShownNodes(frame, relation).map((node) => node.id)
      expect(shown).toStrictEqual(arrived.slice(0, shown.length))
      expect(shown.length + (frame.overflow[relation] ?? 0)).toBe(arrived.length)
    }
  })

  it.each([
    ['overcrowded', graphs.overcrowded],
    ['nine, forty, nine and nine', many],
  ])('leaves out at least one note and keeps every box in the window in %s', (_name, graph) => {
    const frame = arrange(graph, inWindow(tiny))
    expect(Object.values(frame.overflow).some((count) => count > 0)).toBe(true)
    expect(getOutside(frame.nodes, tiny)).toStrictEqual([])
  })

  it('draws no line to a note that is not shown', () => {
    const frame = arrange(many, inWindow(tiny))
    const shown = new Set(frame.nodes.map((node) => node.id))
    for (const edge of frame.edges) {
      expect(shown.has(edge.from), edge.from).toBe(true)
      expect(shown.has(edge.to), edge.to).toBe(true)
    }
  })

  it('shows twenty of thirty children without a window, and counts ten', () => {
    const frame = arrange(build('A note', { child: 30 }))
    expect(getShownNodes(frame, 'child')).toHaveLength(20)
    expect(frame.overflow).toStrictEqual({ child: 10 })
  })

  it('shows as many as the caller allows without a window', () => {
    const frame = arrange(build('A note', { child: 30 }), { maxPerCluster: 5 })
    expect(getShownNodes(frame, 'child')).toHaveLength(5)
    expect(frame.overflow).toStrictEqual({ child: 25 })
  })
})

describe('boxes narrow and gaps close before a note is left out', () => {
  const graph = build('A note', { parent: 4, child: 12, jump: 4, sibling: 4 })
  const windows = [600, 560, 540].map((width) => ({ width, height: 1000 }))

  const createHolder = (viewport: Window) => (packed: PlexOptions) => {
    const frame = arrangePlex(graph, { options: { ...packed, viewport: undefined } })
    return (
      Object.keys(frame.overflow).length === 0 && getOutside(frame.nodes, viewport).length === 0
    )
  }

  const getTightness = (packed: PlexOptions): number =>
    Math.round(
      ((packed.nodeSize.width - packed.minWidth) /
        (DEFAULT_OPTIONS.nodeSize.width - packed.minWidth)) *
        128,
    ) / 128

  it('has a window that cannot hold the graph at the settings', () => {
    const needs = windows.filter(
      (viewport) => !createHolder(viewport)(resolveOptions({ viewport })),
    )
    expect(needs.length).toBeGreaterThan(0)
  })

  it.each(windows)(
    'finds the loosest packing that holds everything in $width×$height',
    (viewport) => {
      const options = resolveOptions({ viewport })
      const canHoldAll = createHolder(viewport)
      const packed = measureCrowding(options, canHoldAll)

      if (canHoldAll(options)) {
        expect(packed).toStrictEqual(options)
        return
      }
      expect(canHoldAll(packed)).toBe(true)
      const tightness = getTightness(packed)
      expect(tightness).toBeLessThan(1)
      expect(canHoldAll(packOptions(options, tightness + 1 / 128))).toBe(false)
    },
  )

  it('returns a window that holds everything at the settings unchanged', () => {
    const options = resolveOptions({ viewport: { width: 2400, height: 1400 } })
    expect(measureCrowding(options, createHolder(options.viewport!))).toStrictEqual(options)
  })
})

describe('gaps open in a roomy window and the arrangement stays inside', () => {
  it('opens the space between neighboring children, by at most the spread', () => {
    const graph = build('A node', { child: 3 })
    const children = (frame: PlexFrame) => getShownNodes(frame, 'child')
    const settings = getSmallestSeparation(children(arrange(graph)))
    const opened = getSmallestSeparation(
      children(arrange(graph, inWindow({ width: 2400, height: 1400 }))),
    )
    const { gap, lineGap, spread } = DEFAULT_OPTIONS

    expect(opened).toBeGreaterThan(settings)
    expect(opened).toBeLessThanOrEqual(spread * Math.max(gap, lineGap) + 1e-6)
  })

  const crowd = build('A node', { parent: 6, child: 21, jump: 7, sibling: 7 })
  it.each([
    { width: 2400, height: 1400 },
    { width: 1970, height: 1440 },
    { width: 1200, height: 800 },
    { width: 700, height: 600 },
    { width: 560, height: 480 },
  ])('keeps every box inside a $width×$height window at a spread of four', (viewport) => {
    const frame = arrange(crowd, inWindow(viewport, { spread: 4 }))
    expect(getOutside(frame.nodes, viewport)).toStrictEqual([])
  })
})

describe('siblings join their parent', () => {
  const frame = arrange(graphs.typical, inWindow({ width: 1200, height: 800 }), {
    measure: byTitle,
  })
  const pairs = frame.edges.map((edge) => `${edge.from} ${edge.to}`)

  it('draws a line from a shown parent to each sibling, and none from the focus', () => {
    for (const sibling of getShownNodes(frame, 'sibling')) {
      const parents = frame.edges
        .filter((edge) => edge.to === sibling.id)
        .map((edge) => frame.nodes.find((node) => node.id === edge.from)?.relation)
      expect(parents).toStrictEqual(['parent'])
      expect(pairs).not.toContain(`focus ${sibling.id}`)
    }
  })

  it('joins every other neighbor to the focus', () => {
    const expected = graphs.typical.edges.map((edge) => `${edge.from} ${edge.to}`)
    expect([...pairs].sort()).toStrictEqual([...expected].sort())
  })
})

describe('the same input gives the same picture', () => {
  it.each(Object.entries(graphs))('arranges %s the same twice', (_name, graph) => {
    const options = inWindow({ width: 1200, height: 800 })
    expect(arrange(graph, options, { measure: byTitle })).toStrictEqual(
      arrange(graph, options, { measure: byTitle }),
    )
  })

  it('places a node the same whichever relation is listed first', () => {
    const { typical } = graphs
    const order = (node: PlexNode) =>
      node.relation === 'focus' ? -1 : -RELATION_NAMES.indexOf(node.relation)
    const shuffled = { ...typical, nodes: [...typical.nodes].sort((a, b) => order(a) - order(b)) }
    const positions = (frame: PlexFrame) =>
      frame.nodes
        .map((n) => [n.id, n.x, n.y, n.width, n.height] as const)
        .sort((a, b) => (a[0] < b[0] ? -1 : 1))
    const options = inWindow({ width: 1200, height: 800 })

    expect(positions(arrange(shuffled, options))).toStrictEqual(
      positions(arrange(typical, options)),
    )
  })
})

describe('the direction of a side is set by the options', () => {
  it('puts children above the focus and parents below when told so', () => {
    const frame = arrange(build('A note', { parent: 3, child: 3 }), {
      direction: { parent: 'down', child: 'up', jump: 'left', reference: 'left', sibling: 'right' },
    })
    const focus = getFocus(frame)
    for (const child of getShownNodes(frame, 'child'))
      expect(child.y + child.height / 2).toBeLessThan(focus.y - focus.height / 2)
    for (const parent of getShownNodes(frame, 'parent'))
      expect(parent.y - parent.height / 2).toBeGreaterThan(focus.y + focus.height / 2)
  })
})

describe('any placement strategy is admitted by the window', () => {
  const graph = graphs.overcrowded

  it('shows only what is inside a 700×600 window and counts the rest', () => {
    const viewport = { width: 700, height: 600 }
    const frame = arrange(graph, inWindow(viewport), { placement: ring })
    expect(
      getOutside(
        frame.nodes.filter((n) => n.relation !== 'focus'),
        viewport,
      ),
    ).toStrictEqual([])
    for (const relation of RELATION_NAMES) {
      expect(getShownNodes(frame, relation).length + (frame.overflow[relation] ?? 0)).toBe(
        getArrivedNodes(graph, relation).length,
      )
    }
  })

  it('still shows some of the 200 children in a 700×600 window with that strategy', () => {
    const frame = arrange(graph, inWindow({ width: 700, height: 600 }), { placement: ring })
    expect(getShownNodes(frame, 'child').length).toBeGreaterThan(0)
  })

  it('shows as many per relation as the cap allows without a window, whatever the strategy returns', () => {
    const frame = arrange(graph, {}, { placement: ring })
    for (const relation of RELATION_NAMES) {
      const arrived = getArrivedNodes(graph, relation).length
      expect(getShownNodes(frame, relation)).toHaveLength(
        Math.min(arrived, DEFAULT_MAX_PER_CLUSTER),
      )
    }
  })
})

const DEFAULT_MAX_PER_CLUSTER = 20

describe('a big group is cut to what fits, not dropped whole', () => {
  const forty = build('A note', { child: 40 })
  const cases: { name: string; graph: PlexGraph; viewport: Window }[] = [
    { name: '40 children', graph: forty, viewport: { width: 400, height: 400 } },
    { name: '40 children', graph: forty, viewport: { width: 700, height: 600 } },
    { name: 'overcrowded', graph: graphs.overcrowded, viewport: { width: 700, height: 600 } },
    { name: 'overcrowded', graph: graphs.overcrowded, viewport: { width: 400, height: 400 } },
  ]

  it.each(cases)('shows some children of $name in $viewport', ({ graph, viewport }) => {
    const frame = arrange(graph, inWindow(viewport))
    const shown = getShownNodes(frame, 'child')
    expect(shown.length).toBeGreaterThanOrEqual(1)
    expect(shown.length + (frame.overflow.child ?? 0)).toBe(getArrivedNodes(graph, 'child').length)
    expect(getOutside(shown, viewport)).toStrictEqual([])
    expect(getOverlaps(frame)).toStrictEqual([])
  })

  it.each([
    { width: 400, height: 400 },
    { width: 700, height: 600 },
    { width: 1200, height: 800 },
  ])('shows forty children no fewer than twenty in $width×$height', (viewport) => {
    const twenty = arrange(build('A note', { child: 20 }), inWindow(viewport))
    const more = arrange(forty, inWindow(viewport))
    expect(getShownNodes(more, 'child').length).toBeGreaterThanOrEqual(
      getShownNodes(twenty, 'child').length,
    )
  })
})

describe('with no gap no two boxes overlap', () => {
  it.each([1, 2, 5, 9, 16, 30])('arranges %i children at gap 0 and line gap 0', (child) => {
    const graph = build('A note', { parent: 3, child, jump: 4, sibling: 2 })
    for (const viewport of [undefined, { width: 1200, height: 800 }]) {
      const frame = arrange(graph, {
        gap: 0,
        lineGap: 0,
        room: 0,
        ...(viewport ? { viewport } : {}),
      })
      expect(getOverlaps(frame)).toStrictEqual([])
    }
  })
})
