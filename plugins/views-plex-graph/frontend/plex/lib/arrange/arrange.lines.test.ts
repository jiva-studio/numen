/** A note is set where its line does not lie along another line or run through another note. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { createRandom } from '../../fixtures/random'
import type { PlacedNode } from '../node'
import type { PlexEdge, PlexGraph, PlexNode, PlexRelation } from '../../../types'

/** The sine of the angle below which two lines read as one line. */
const PARALLEL_SINE = 0.12

const VIEWPORT = { width: 1200, height: 900 }

const getSine = (apex: PlacedNode, first: PlacedNode, second: PlacedNode): number => {
  const ux = first.x - apex.x
  const uy = first.y - apex.y
  const vx = second.x - apex.x
  const vy = second.y - apex.y
  return Math.abs(ux * vy - uy * vx) / (Math.hypot(ux, uy) * Math.hypot(vx, vy))
}

const isInside = (box: PlacedNode, x: number, y: number): boolean =>
  Math.abs(x - box.x) < box.width / 2 && Math.abs(y - box.y) < box.height / 2

const isThrough = (from: PlacedNode, to: PlacedNode, box: PlacedNode): boolean => {
  for (let step = 1; step < 50; step++) {
    const t = step / 50
    if (isInside(box, from.x + (to.x - from.x) * t, from.y + (to.y - from.y) * t)) return true
  }
  return false
}

const getEnds = (frame: ReturnType<typeof arrangePlex>, edge: PlexEdge) => {
  const byId = new Map(frame.nodes.map((node) => [node.id, node]))
  return [byId.get(edge.from), byId.get(edge.to)] as const
}

const listCorners = (frame: ReturnType<typeof arrangePlex>, graph: PlexGraph) => {
  const byId = new Map(frame.nodes.map((node) => [node.id, node]))
  const corners: { apex: PlacedNode; first: PlacedNode; second: PlacedNode }[] = []
  for (const apex of frame.nodes) {
    if (apex.relation === 'focus') continue
    const ends = graph.edges
      .filter((edge) => edge.from === apex.id || edge.to === apex.id)
      .map((edge) => byId.get(edge.from === apex.id ? edge.to : edge.from))
      .filter((node): node is PlacedNode => node !== undefined)
    for (let index = 0; index < ends.length; index++) {
      for (const other of ends.slice(index + 1)) {
        corners.push({ apex, first: ends[index]!, second: other })
      }
    }
  }
  return corners
}

function createChain(relation: 'parent' | 'child', first: string, second: string): PlexGraph {
  const nodes: PlexNode[] = [
    { id: 'focus', title: 'Mechanics', relation: 'focus' },
    { id: first, title: first, relation },
    { id: second, title: second, relation },
  ]
  const edges: PlexEdge[] =
    relation === 'parent'
      ? [
          { from: first, to: 'focus' },
          { from: second, to: first },
        ]
      : [
          { from: 'focus', to: first },
          { from: first, to: second },
        ]
  return { nodes, edges }
}

describe.each([
  ['parent', 'Chapter', 'Part'],
  ['child', 'Section', 'Paragraph'],
] as const)('a %s of a %s', (relation, near, far) => {
  it.each([
    ['the near note first', near, far],
    ['the far note first', far, near],
  ])('is not on one line with the line to the focus, %s', (_, first, second) => {
    const graph = createChain(relation, near, far)
    const ordered = {
      nodes: [
        graph.nodes[0]!,
        ...[first, second].map((id) => graph.nodes.find((n) => n.id === id)!),
      ],
      edges: graph.edges,
    }
    const frame = arrangePlex(ordered, { options: { viewport: VIEWPORT } })
    const byId = new Map(frame.nodes.map((node) => [node.id, node]))

    expect(frame.overflow).toStrictEqual({})
    expect(getSine(byId.get(near)!, byId.get('focus')!, byId.get(far)!)).toBeGreaterThanOrEqual(
      PARALLEL_SINE,
    )
  })
})

describe('a window that holds nothing but one column', () => {
  it('still shows both notes of a line of parents', () => {
    const frame = arrangePlex(createChain('parent', 'Chapter', 'Part'), {
      options: { viewport: { width: 176, height: 900 } },
    })

    expect(frame.nodes.map((node) => node.id).sort()).toStrictEqual(['Chapter', 'Part', 'focus'])
    expect(frame.overflow).toStrictEqual({})
  })
})

describe('a focus with children and children of children, in a window with room to spare', () => {
  const names = ['dd', 'dddd', 'Test 222', 'Chapter 1: Observing the armies']
  const graph: PlexGraph = {
    nodes: [
      { id: 'focus', title: 'Bhagavad-gita', relation: 'focus' },
      ...names.map((title): PlexNode => ({ id: title, title, relation: 'child' })),
      { id: 'uuu', title: 'uuu', relation: 'child' },
      { id: 'ddd', title: 'ddd', relation: 'child' },
    ],
    edges: [
      ...names.map((title): PlexEdge => ({ from: 'focus', to: title })),
      { from: 'dddd', to: 'uuu' },
      { from: 'dd', to: 'ddd' },
    ],
  }

  it.each([
    [{ width: 843, height: 603 }, undefined],
    [{ width: 843, height: 603 }, (node: PlexNode) => 40 + node.title.length * 7],
    [{ width: 1600, height: 1000 }, undefined],
  ])('keeps the lines apart and off the other notes in %o', (viewport, measure) => {
    const frame = arrangePlex(graph, { options: { viewport }, ...(measure ? { measure } : {}) })
    const byId = new Map(frame.nodes.map((node) => [node.id, node]))
    const focus = byId.get('focus')!

    expect(frame.overflow).toStrictEqual({})
    for (const title of [...names, 'uuu', 'ddd']) {
      expect(byId.get(title)!.y, `${title} below the focus`).toBeGreaterThan(focus.y)
    }
    expect(
      getSine(byId.get('dddd')!, focus, byId.get('uuu')!),
      'uuu on the line of dddd',
    ).toBeGreaterThanOrEqual(PARALLEL_SINE)
    expect(
      getSine(byId.get('dd')!, focus, byId.get('ddd')!),
      'ddd on the line of dd',
    ).toBeGreaterThanOrEqual(PARALLEL_SINE)
    for (const edge of graph.edges) {
      const [from, to] = getEnds(frame, edge)
      for (const box of frame.nodes) {
        if (box === from || box === to) continue
        expect(isThrough(from!, to!, box), `${edge.from}-${edge.to} through ${box.id}`).toBe(false)
      }
    }
    for (const first of names) {
      for (const second of names.slice(names.indexOf(first) + 1)) {
        expect(
          getSine(focus, byId.get(first)!, byId.get(second)!),
          `${first} and ${second}`,
        ).toBeGreaterThanOrEqual(PARALLEL_SINE)
      }
    }
  })
})

describe('graphs of random notes with notes joined to them', () => {
  const random = createRandom(2026)
  const whole = (low: number, high: number) => low + Math.floor(random() * (high - low + 1))
  const RELATIONS: readonly PlexRelation[] = ['parent', 'child', 'jump']

  const createGraph = (): PlexGraph => {
    const nodes: PlexNode[] = [{ id: 'focus', title: 'Optics', relation: 'focus' }]
    const edges: PlexEdge[] = []
    for (const relation of RELATIONS) {
      for (let index = whole(0, 2); index > 0; index--) {
        const id = `${relation}-${index}`
        nodes.push({ id, title: id, relation })
        edges.push(relation === 'child' ? { from: 'focus', to: id } : { from: id, to: 'focus' })
        if (relation === 'jump') continue
        for (let more = whole(0, 2); more > 0; more--) {
          const next = `${id}-${more}`
          nodes.push({ id: next, title: next, relation })
          edges.push(relation === 'child' ? { from: id, to: next } : { from: next, to: id })
        }
      }
    }
    return { nodes, edges }
  }

  const graphs = Array.from({ length: 80 }, createGraph)

  it.each(graphs.map((graph, round) => [round, graph] as const))(
    'no line lies along another and none runs through a note, in case %i',
    (_, graph) => {
      const frame = arrangePlex(graph, { options: { viewport: VIEWPORT } })

      expect(frame.overflow).toStrictEqual({})
      for (const { apex, first, second } of listCorners(frame, graph)) {
        expect(getSine(apex, first, second)).toBeGreaterThanOrEqual(PARALLEL_SINE)
      }
      for (const edge of graph.edges) {
        const [from, to] = getEnds(frame, edge)
        for (const box of frame.nodes) {
          if (box === from || box === to) continue
          expect(isThrough(from!, to!, box)).toBe(false)
        }
      }
    },
  )
})
