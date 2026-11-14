/** Each group keeps to its own side of the focus and grows about the middle of that side; a note leaves its lane only when the lane has no room. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getBox, isOverlapping } from './layout/boxes'
import { DEFAULT_OPTIONS } from './options'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import type { PlexEdge, PlexGraph, PlexNode, PlexRelation } from '../../../types'

const { nodeSize, gap, lineGap } = DEFAULT_OPTIONS
const ROW = nodeSize.height + lineGap
const COLUMN = nodeSize.width + gap

const createNodes = (relation: PlexRelation, count: number): PlexNode[] =>
  Array.from({ length: count }, (_, index) => ({
    id: `${relation}-${index}`,
    title: `${relation} ${index}`,
    relation,
  }))

const createGraph = (counts: Partial<Record<PlexRelation, number>>): PlexGraph => {
  const nodes: PlexNode[] = [{ id: 'focus', title: 'Flywheels', relation: 'focus' }]
  const edges: PlexEdge[] = []
  for (const [relation, count] of Object.entries(counts) as [PlexRelation, number][]) {
    const group = createNodes(relation, count)
    nodes.push(...group)
    for (const node of group) {
      if (relation === 'child') edges.push({ from: 'focus', to: node.id })
      else if (relation === 'sibling') edges.push({ from: 'parent-0', to: node.id })
      else edges.push({ from: node.id, to: 'focus' })
    }
  }
  return { nodes, edges }
}

const list = (frame: PlexFrame, relation: PlexRelation): PlacedNode[] =>
  frame.nodes.filter((node) => node.relation === relation)

const mean = (values: readonly number[]): number =>
  values.reduce((sum, value) => sum + value, 0) / values.length

const getOverlaps = (frame: PlexFrame): string[] =>
  frame.nodes.flatMap((a, index) =>
    frame.nodes
      .slice(index + 1)
      .filter((b) => isOverlapping(getBox(a, 0), getBox(b, 0)))
      .map((b) => `${a.id} / ${b.id}`),
  )

describe('a focus with forty siblings, twenty children, twelve jumps and three parents', () => {
  const graph = createGraph({ parent: 3, child: 20, jump: 12, sibling: 40 })
  const frame = arrangePlex(graph, { options: { viewport: { width: 1880, height: 1500 } } })
  const focus = frame.nodes.find((node) => node.relation === 'focus')!
  const jumps = list(frame, 'jump')
  const siblings = list(frame, 'sibling')
  const children = list(frame, 'child')

  it('shows every note, apart', () => {
    expect(frame.overflow).toStrictEqual({})
    expect(frame.nodes).toHaveLength(graph.nodes.length)
    expect(getOverlaps(frame)).toStrictEqual([])
  })

  it('puts every jump to the left of the focus and every sibling to the right', () => {
    for (const jump of jumps) expect(jump.x + jump.width / 2, jump.id).toBeLessThan(focus.x)
    for (const sibling of siblings) {
      expect(sibling.x - sibling.width / 2, sibling.id).toBeGreaterThan(focus.x)
    }
  })

  it('does not push a jump down past the children while the left side has room', () => {
    const lowestChild = Math.max(...children.map((child) => child.y))
    for (const jump of jumps) expect(jump.y, jump.id).toBeLessThanOrEqual(lowestChild)
  })

  it('centers the jumps and the siblings on the row of the focus', () => {
    expect(Math.abs(mean(jumps.map((jump) => jump.y)) - focus.y)).toBeLessThanOrEqual(ROW)
    expect(Math.abs(mean(siblings.map((sibling) => sibling.y)) - focus.y)).toBeLessThanOrEqual(ROW)
  })

  it('does not lay the siblings out as a staircase', () => {
    const byHeight = [...siblings].sort((a, b) => a.y - b.y)
    const half = byHeight.length / 2
    const upper = mean(byHeight.slice(0, half).map((sibling) => sibling.x))
    const lower = mean(byHeight.slice(half).map((sibling) => sibling.x))
    expect(Math.abs(upper - lower)).toBeLessThanOrEqual(COLUMN)
  })
})

describe('a focus with children, a parent and jumps in a small window', () => {
  const graph = createGraph({ parent: 1, child: 6, jump: 3 })
  const frame = arrangePlex(graph, { options: { viewport: { width: 843, height: 603 } } })
  const focus = frame.nodes.find((node) => node.relation === 'focus')!

  it('keeps each group on its own side of the focus', () => {
    expect(frame.overflow).toStrictEqual({})
    for (const child of list(frame, 'child')) expect(child.y, child.id).toBeGreaterThan(focus.y)
    for (const parent of list(frame, 'parent')) expect(parent.y, parent.id).toBeLessThan(focus.y)
    for (const jump of list(frame, 'jump')) expect(jump.x, jump.id).toBeLessThan(focus.x)
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe('jumps in a window with no room on the left of the focus', () => {
  const graph = createGraph({ jump: 4 })

  it('keep to the left where the window has room there', () => {
    const frame = arrangePlex(graph, { options: { viewport: { width: 1200, height: 900 } } })

    expect(frame.overflow).toStrictEqual({})
    for (const jump of list(frame, 'jump')) expect(jump.x, jump.id).toBeLessThan(0)
  })

  it('are still shown above or below the focus', () => {
    const frame = arrangePlex(graph, { options: { viewport: { width: 260, height: 900 } } })

    expect(list(frame, 'jump').length).toBeGreaterThan(0)
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

interface Band {
  readonly left: number
  readonly right: number
  readonly top: number
  readonly bottom: number
}

const getBand = (nodes: readonly PlacedNode[]): Band => ({
  left: Math.min(...nodes.map((node) => node.x - node.width / 2)),
  right: Math.max(...nodes.map((node) => node.x + node.width / 2)),
  top: Math.min(...nodes.map((node) => node.y - node.height / 2)),
  bottom: Math.max(...nodes.map((node) => node.y + node.height / 2)),
})

const isOutside = (node: PlacedNode, band: Band): boolean =>
  node.x - node.width / 2 >= band.right ||
  node.x + node.width / 2 <= band.left ||
  node.y - node.height / 2 >= band.bottom ||
  node.y + node.height / 2 <= band.top

const getDistance = (first: PlacedNode, second: PlacedNode): number =>
  Math.max(
    Math.abs(first.x - second.x) - (first.width + second.width) / 2,
    Math.abs(first.y - second.y) - (first.height + second.height) / 2,
  )

const GROUP_GAP = 2 * gap - 1

describe.each([
  { width: 1880, height: 1500 },
  { width: 1200, height: 900 },
])(
  'the territories of $width x $height with forty siblings, twenty children, twelve jumps and a parent',
  (viewport) => {
    const graph = createGraph({ parent: 1, child: 20, jump: 12, sibling: 40 })
    const frame = arrangePlex(graph, { options: { viewport } })

    it('counts every note as shown or not shown', () => {
      const hidden = Object.values(frame.overflow).reduce((sum, count) => sum + count, 0)
      expect(frame.nodes.length - 1 + hidden).toBe(graph.nodes.length - 1)
      expect(getOverlaps(frame)).toStrictEqual([])
    })
  },
)

describe('the territories of 1880 x 1500 with forty siblings, twenty children, twelve jumps and a parent', () => {
  const graph = createGraph({ parent: 1, child: 20, jump: 12, sibling: 40 })
  const frame = arrangePlex(graph, { options: { viewport: { width: 1880, height: 1500 } } })
  const siblings = list(frame, 'sibling')
  const children = list(frame, 'child')
  const parents = list(frame, 'parent')
  const jumps = list(frame, 'jump')

  it('keeps every sibling out of the band of the children and the band of the parents', () => {
    const childBand = getBand(children)
    const parentBand = getBand(parents)
    for (const sibling of siblings) {
      expect(isOutside(sibling, childBand), sibling.id).toBe(true)
      expect(isOutside(sibling, parentBand), sibling.id).toBe(true)
    }
  })

  it('keeps every jump out of the band of the children and the band of the parents', () => {
    const childBand = getBand(children)
    const parentBand = getBand(parents)
    for (const jump of jumps) {
      expect(isOutside(jump, childBand), jump.id).toBe(true)
      expect(isOutside(jump, parentBand), jump.id).toBe(true)
    }
  })

  it('keeps the group gap between a sibling and a child, and between a sibling and a parent', () => {
    for (const sibling of siblings) {
      for (const other of [...children, ...parents]) {
        expect(getDistance(sibling, other), `${sibling.id} / ${other.id}`).toBeGreaterThanOrEqual(
          GROUP_GAP,
        )
      }
    }
  })
})

describe('the border between the children and the siblings', () => {
  const viewport = { width: 1880, height: 1500 }
  const getFirstSiblingEdge = (children: number): number => {
    const frame = arrangePlex(createGraph({ child: children, sibling: 5 }), {
      options: { viewport },
    })
    return Math.min(...list(frame, 'sibling').map((node) => node.x - node.width / 2))
  }

  it('moves outward when the children need more room', () => {
    expect(getFirstSiblingEdge(20)).toBeGreaterThan(getFirstSiblingEdge(2))
  })
})

describe('a window too small to keep the groups apart', () => {
  const graph = createGraph({ parent: 2, child: 12, jump: 6, sibling: 12 })
  const frame = arrangePlex(graph, { options: { viewport: { width: 420, height: 360 } } })

  it('still places notes, apart, and counts the rest as not shown', () => {
    const hidden = Object.values(frame.overflow).reduce((sum, count) => sum + count, 0)
    expect(frame.nodes.length).toBeGreaterThan(1)
    expect(frame.nodes.length - 1 + hidden).toBe(graph.nodes.length - 1)
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})
