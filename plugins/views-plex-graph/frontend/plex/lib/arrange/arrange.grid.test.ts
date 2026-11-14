/**
 * Each side of the focus holds one block of the grid. The graphs are shaped like the ones the blocks were made for:
 * a note with many children, few jumps and many siblings; a note with a few children among jumps, references and siblings;
 * a note with only a parent, jumps and references; and a note whose siblings and jumps each fill a few columns.
 */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { build, type RelationCounts } from '../../fixtures/build'
import { getFootprint, getFootprintCenterY, type NodeRoom } from './layout/placement'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import type { PlexNode } from '../../../types'

/** A note with names under its pill, and a parent with the name of its own parent over it. */
const NAMES: NodeRoom = { top: 16, bottom: 60, width: 140 }
const PARENT_NAMES: NodeRoom = { top: 16, bottom: 45, width: 124 }
const NO_ROOM: NodeRoom = { top: 0, bottom: 0, width: 0 }

const getRoom = (node: PlexNode): NodeRoom => (node.relation === 'parent' ? PARENT_NAMES : NAMES)

interface Box {
  readonly id: string
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

const getBox = (node: PlacedNode): Box => {
  const room = node.relation === 'focus' ? NO_ROOM : getRoom(node)
  const { width, height } = getFootprint(node, room)
  const y = getFootprintCenterY(node.y, room)
  return {
    id: node.id,
    minX: node.x - width / 2,
    maxX: node.x + width / 2,
    minY: y - height / 2,
    maxY: y + height / 2,
  }
}

const isOverlapping = (a: Box, b: Box): boolean =>
  a.minX < b.maxX - 1e-6 &&
  b.minX < a.maxX - 1e-6 &&
  a.minY < b.maxY - 1e-6 &&
  b.minY < a.maxY - 1e-6

const getBounds = (boxes: readonly Box[]): Box => ({
  id: 'bounds',
  minX: Math.min(...boxes.map((box) => box.minX)),
  maxX: Math.max(...boxes.map((box) => box.maxX)),
  minY: Math.min(...boxes.map((box) => box.minY)),
  maxY: Math.max(...boxes.map((box) => box.maxY)),
})

const arrange = (counts: RelationCounts, viewport: { width: number; height: number }) => {
  const graph = build('A note', counts)
  return {
    graph,
    frame: arrangePlex(graph, {
      options: { viewport },
      measure: (node) => 40 + node.title.length * 7,
      room: getRoom,
    }),
  }
}

const ofRelation = (frame: PlexFrame, ...relations: string[]) =>
  frame.nodes.filter((node) => relations.includes(node.relation))

const getFocus = (frame: PlexFrame) => frame.nodes.find((node) => node.relation === 'focus')!

const getOverlaps = (frame: PlexFrame): string[] => {
  const boxes = frame.nodes.map(getBox)
  return boxes.flatMap((a, index) =>
    boxes
      .slice(index + 1)
      .filter((b) => isOverlapping(a, b))
      .map((b) => `${a.id} / ${b.id}`),
  )
}

const listInside = (notes: readonly PlacedNode[], area: Box): string[] =>
  notes.filter((note) => isOverlapping(getBox(note), area)).map((note) => note.id)

function isWholeBlock(notes: readonly PlacedNode[]): boolean {
  const columns = [...new Set(notes.map((note) => note.x))].sort(
    (a, b) => Math.abs(a) - Math.abs(b),
  )
  const rows = [...new Set(notes.map((note) => getBox(note).minY))].sort((a, b) => a - b)
  const isEven = rows.every(
    (row, index) => index < 2 || Math.abs(row - rows[index - 1]! - (rows[1]! - rows[0]!)) < 1e-6,
  )
  const isFilled = rows.every((row) => {
    const xs = notes.filter((note) => getBox(note).minY === row).map((note) => note.x)
    return xs.every((x) => columns.indexOf(x) < xs.length)
  })
  return isEven && isFilled
}

const MANY_CHILDREN = { parent: 1, child: 53, jump: 3, reference: 3, sibling: 39 }

describe('a note with many children, few jumps and many siblings in 1800×1900', () => {
  const { graph, frame } = arrange(MANY_CHILDREN, { width: 1800, height: 1900 })

  it('shows every note', () => {
    expect(frame.overflow).toStrictEqual({})
    expect(frame.nodes).toHaveLength(graph.nodes.length)
  })

  it('sets no jump inside the bounds of the children', () => {
    const children = getBounds(ofRelation(frame, 'child').map(getBox))
    expect(listInside(ofRelation(frame, 'jump'), children)).toStrictEqual([])
  })

  it('overlaps no two notes', () => {
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe('a note with many children, few jumps and many siblings in 1400×900', () => {
  const { frame } = arrange(MANY_CHILDREN, { width: 1400, height: 900 })

  it('shows every jump, left of the focus and outside the bounds of the children', () => {
    const focus = getFocus(frame)
    const jumps = ofRelation(frame, 'jump')
    const children = getBounds(ofRelation(frame, 'child').map(getBox))
    expect(jumps).toHaveLength(MANY_CHILDREN.jump)
    for (const jump of jumps) expect(getBox(jump).maxX).toBeLessThan(focus.x - focus.width / 2)
    expect(listInside(jumps, children)).toStrictEqual([])
  })

  it('overlaps no two notes', () => {
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe.each([
  { width: 1100, height: 1300 },
  { width: 1280, height: 1600 },
])(
  'a note with four children among jumps, references and siblings in $width×$height',
  (viewport) => {
    const { graph, frame } = arrange(
      { parent: 1, child: 4, jump: 4, reference: 3, sibling: 16 },
      viewport,
    )

    it('shows every note', () => {
      expect(frame.overflow).toStrictEqual({})
      expect(frame.nodes).toHaveLength(graph.nodes.length)
    })

    it('sets the children under the focus, centered on it, outside the bounds of the jumps', () => {
      const focus = getFocus(frame)
      const children = ofRelation(frame, 'child')
      const xs = children.map((child) => child.x)
      for (const child of children)
        expect(getBox(child).minY).toBeGreaterThan(focus.y + focus.height / 2)
      expect(Math.min(...xs) + Math.max(...xs)).toBeCloseTo(0, 6)
      const jumps = getBounds(ofRelation(frame, 'jump').map(getBox))
      expect(listInside(children, jumps)).toStrictEqual([])
    })

    it('overlaps no two notes', () => {
      expect(getOverlaps(frame)).toStrictEqual([])
    })
  },
)

describe.each([
  { width: 1100, height: 1300 },
  { width: 1280, height: 1600 },
])('a note with a parent, jumps and references in $width×$height', (viewport) => {
  const { frame } = arrange({ parent: 1, jump: 3, reference: 5 }, viewport)

  it('sets every jump left of the focus, its line leaving the focus on the left', () => {
    const focus = getFocus(frame)
    const left = focus.x - focus.width / 2
    const jumps = ofRelation(frame, 'jump')
    expect(jumps).toHaveLength(3)
    for (const jump of jumps) {
      expect(getBox(jump).maxX).toBeLessThan(left)
      const edge = frame.edges.find((one) => one.from === jump.id || one.to === jump.id)!
      const atFocus = edge.from === focus.id ? edge.fromPoint : edge.toPoint
      expect(atFocus).toStrictEqual({ x: left, y: focus.y })
    }
  })

  it('overlaps no two notes', () => {
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe.each([
  { width: 1100, height: 1300 },
  { width: 1280, height: 1600 },
])('a note whose siblings and jumps fill columns in $width×$height', (viewport) => {
  const { graph, frame } = arrange(
    { parent: 1, child: 1, jump: 5, reference: 2, sibling: 13 },
    viewport,
  )

  it('shows every note', () => {
    expect(frame.overflow).toStrictEqual({})
    expect(frame.nodes).toHaveLength(graph.nodes.length)
  })

  it('keeps the siblings in columns with no note off its column and no hole', () => {
    expect(isWholeBlock(ofRelation(frame, 'sibling'))).toBe(true)
  })

  it('keeps the jumps and references in columns with no note off its column and no hole', () => {
    expect(isWholeBlock(ofRelation(frame, 'jump', 'reference'))).toBe(true)
  })

  it('overlaps no two notes', () => {
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe('a note whose children sit between many siblings and a few jumps in 1825×1890', () => {
  const { graph, frame } = arrange(
    { parent: 1, child: 13, jump: 3, reference: 2, sibling: 39 },
    { width: 1825, height: 1890 },
  )

  it('shows every note', () => {
    expect(frame.overflow).toStrictEqual({})
    expect(frame.nodes).toHaveLength(graph.nodes.length)
  })

  it('sets the first row of children right under the focus', () => {
    const focus = getFocus(frame)
    const children = ofRelation(frame, 'child')
    const top = Math.min(...children.map((child) => getBox(child).minY))
    expect(top).toBeLessThanOrEqual(focus.y + focus.height / 2 + DEFAULT_OPTIONS.focusGap + 1e-6)
  })

  it('keeps every side note out of the columns of the children at their height', () => {
    const children = getBounds(ofRelation(frame, 'child').map(getBox))
    const sides = ofRelation(frame, 'jump', 'reference', 'sibling')
    expect(listInside(sides, children)).toStrictEqual([])
  })
})

describe('a block of short titles', () => {
  const graph = build('A note', { child: 2, sibling: 9 })
  const shortTitles = {
    ...graph,
    nodes: graph.nodes.map((node, index) =>
      node.relation === 'focus' ? node : { ...node, title: `T${index}` },
    ),
  }
  const width = (node: PlexNode) => 40 + node.title.length * 7
  const withNames = (node: PlexNode): NodeRoom => (node.relation === 'child' ? NAMES : NO_ROOM)
  const frame = arrangePlex(shortTitles, {
    options: { viewport: { width: 1800, height: 1400 } },
    measure: width,
    room: withNames,
  })
  const { gap, lineGap, nodeSize } = DEFAULT_OPTIONS
  const siblings = ofRelation(frame, 'sibling')
  const steps = (values: readonly number[]) => {
    const sorted = [...new Set(values.map((value) => Math.round(value * 1e6) / 1e6))].sort(
      (a, b) => a - b,
    )
    return sorted.slice(1).map((value, index) => value - sorted[index]!)
  }

  it('steps its columns by the widest title and the gap, not by the widest note elsewhere', () => {
    const widest = Math.max(...siblings.map((sibling) => sibling.width))
    expect(steps(siblings.map((sibling) => sibling.x)).length).toBeGreaterThan(0)
    for (const step of steps(siblings.map((sibling) => sibling.x)))
      expect(step).toBeLessThanOrEqual(widest + gap + 1e-6)
  })

  it('steps its rows by the height of a note and the line gap, the names of other notes left out', () => {
    expect(steps(siblings.map((sibling) => sibling.y)).length).toBeGreaterThan(0)
    for (const step of steps(siblings.map((sibling) => sibling.y)))
      expect(step).toBeLessThanOrEqual(nodeSize.height + lineGap + 1e-6)
  })
})

const getRowY = (node: PlacedNode): number => {
  const box = getBox(node)
  return (box.minY + box.maxY) / 2
}

describe.each([
  { reference: 3, jump: 12 },
  { reference: 12, jump: 3 },
])('a note with $reference references and $jump jumps in 1400×900', (counts) => {
  const { frame } = arrange(counts, { width: 1400, height: 900 })
  const focus = getFocus(frame)
  const references = ofRelation(frame, 'reference')
  const jumps = ofRelation(frame, 'jump')

  it('sets the lowest row of references on the row of the focus and every other one over it', () => {
    expect(references).toHaveLength(counts.reference)
    expect(Math.max(...references.map(getRowY))).toBeCloseTo(focus.y, 6)
  })

  it('sets every jump under the row of the focus, its first row the next row under the references', () => {
    expect(jumps).toHaveLength(counts.jump)
    for (const jump of jumps) expect(getRowY(jump)).toBeGreaterThan(focus.y)
    const rows = [...new Set([...references, ...jumps].map((note) => getRowY(note).toFixed(6)))]
      .map(Number)
      .sort((first, second) => first - second)
    const steps = rows.slice(1).map((row, index) => row - rows[index]!)
    for (const step of steps) expect(step).toBeCloseTo(steps[0]!, 6)
  })

  it('overlaps no two notes', () => {
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe('a note with only jumps in 1400×900', () => {
  const { frame } = arrange({ jump: 12 }, { width: 1400, height: 900 })

  it('centers the jumps on the row of the focus', () => {
    const jumps = getBounds(ofRelation(frame, 'jump').map(getBox))
    expect((jumps.minY + jumps.maxY) / 2).toBeCloseTo(getFocus(frame).y, 6)
  })
})
