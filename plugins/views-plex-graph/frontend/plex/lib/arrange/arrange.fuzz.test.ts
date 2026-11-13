/** Random windows, counts, widths and rooms: what the arrangement shows is what fits, and a note is not cut while there is room. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { SpotIndex } from './layout/freeSpot'
import { getFootprint, getFootprintCenterY, type NodeRoom } from './layout/placement'
import { resolveOptions } from './options'
import { RELATION_NAMES } from '../relation'
import type { PlacedNode } from '../node'
import { createRandom } from '../../fixtures/random'
import type { PlexEdge, PlexGraph, PlexNode, PlexRelation } from '../../../types'

const ROUNDS = 120
const RELATIONS_DRAWN: readonly PlexRelation[] = ['parent', 'child', 'jump', 'reference', 'sibling']
const RELATIONS_CUT: readonly PlexRelation[] = ['parent', 'child', 'jump', 'reference']

const random = createRandom(1837)
const between = (low: number, high: number) => low + random() * (high - low)
const whole = (low: number, high: number) => Math.floor(between(low, high + 1))

interface Case {
  readonly graph: PlexGraph
  readonly viewport: { width: number; height: number }
  readonly widths: ReadonlyMap<string, number>
  readonly rooms: ReadonlyMap<string, NodeRoom>
}

const createCase = (): Case => {
  const viewport = { width: whole(480, 1400), height: whole(360, 900) }
  const nodes: PlexNode[] = [{ id: 'focus', title: 'Focus', relation: 'focus' }]
  const edges: PlexEdge[] = []
  const widths = new Map<string, number>()
  const rooms = new Map<string, NodeRoom>()
  for (const relation of RELATIONS_DRAWN) {
    for (let index = whole(0, 22); index > 0; index--) {
      const id = `${relation}-${index}`
      nodes.push({ id, title: id, relation })
      edges.push(relation === 'parent' ? { from: id, to: 'focus' } : { from: 'focus', to: id })
      widths.set(id, whole(60, 190))
      const bottom = whole(0, 3) * 14
      rooms.set(id, { top: 0, bottom, width: bottom === 0 ? 0 : whole(80, 170) })
    }
  }
  return { graph: { nodes, edges }, viewport, widths, rooms }
}

const cases = Array.from({ length: ROUNDS }, createCase)

const arrangeCase = ({ graph, viewport, widths, rooms }: Case) =>
  arrangePlex(graph, {
    options: { viewport },
    measure: (node) => widths.get(node.id) ?? 100,
    room: (node) => rooms.get(node.id) ?? { top: 0, bottom: 0, width: 0 },
  })

const footprintOf = (node: PlacedNode, rooms: ReadonlyMap<string, NodeRoom>) => {
  const room = rooms.get(node.id) ?? { top: 0, bottom: 0, width: 0 }
  const { width, height } = getFootprint(node, room)
  return { x: node.x, y: getFootprintCenterY(node.y, room), width, height }
}

const countByRelation = (nodes: readonly { relation: string }[], relation: string) =>
  nodes.filter((node) => node.relation === relation).length

const getFocusBox = (frame: ReturnType<typeof arrangeCase>) => {
  const focus = frame.nodes.find((node) => node.relation === 'focus')!
  return { x: focus.x, y: focus.y, width: focus.width, height: focus.height }
}

describe('what is shown of a graph', () => {
  it.each(cases.map((one, round) => [round, one] as const))(
    'adds up to what arrived, fits the window and overlaps nothing, in case %i',
    (_, one) => {
      const frame = arrangeCase(one)
      const shown = frame.nodes.filter((node) => node.relation !== 'focus')

      for (const relation of RELATION_NAMES) {
        expect(countByRelation(shown, relation) + (frame.overflow[relation] ?? 0), relation).toBe(
          countByRelation(one.graph.nodes, relation),
        )
      }

      const { margin } = resolveOptions({ viewport: one.viewport })
      const boxes = shown.map((node) => footprintOf(node, one.rooms))
      for (const box of boxes) {
        expect(Math.abs(box.x) + box.width / 2).toBeLessThanOrEqual(
          one.viewport.width / 2 - margin + 1,
        )
        expect(Math.abs(box.y) + box.height / 2).toBeLessThanOrEqual(
          one.viewport.height / 2 - margin + 1,
        )
      }
      // The focus is a box like the others: no note lies over it.
      const index = new SpotIndex([])
      const overlaps = [getFocusBox(frame), ...boxes].filter((box) => {
        const isFree = index.isFree(
          { ...box, width: box.width - 2, height: box.height - 2 },
          { gap: 0, lineGap: 0 },
        )
        index.add(box)
        return !isFree
      })
      expect(overlaps).toStrictEqual([])

      // A line is drawn for each note that is shown, and for no note that is not.
      const shownIds = new Set(shown.map((node) => node.id))
      expect(frame.edges.length).toBe(shown.length)
      for (const edge of frame.edges) {
        expect(shownIds.has(edge.from === 'focus' ? edge.to : edge.from)).toBe(true)
        for (const end of [edge.fromPoint, edge.toPoint]) {
          expect(Math.abs(end.x)).toBeLessThanOrEqual(one.viewport.width / 2)
          expect(Math.abs(end.y)).toBeLessThanOrEqual(one.viewport.height / 2)
        }
      }
    },
  )
})

describe('a graph cut to what was shown', () => {
  it.each(cases.map((one, round) => [round, one] as const))('is shown whole, in case %i', (_, one) => {
    const shown = new Set(arrangeCase(one).nodes.map((node) => node.id))
    const kept: Case = {
      ...one,
      graph: {
        nodes: one.graph.nodes.filter((node) => shown.has(node.id)),
        edges: one.graph.edges.filter((edge) => shown.has(edge.from) && shown.has(edge.to)),
      },
    }

    expect(arrangeCase(kept).overflow).toStrictEqual({})
  })
})

describe('a note that is cut', () => {
  const cutNodes = (one: Case, relation: PlexRelation, keep: number): Case => {
    const members = one.graph.nodes.filter((node) => node.relation === relation)
    const left = new Set(members.slice(keep).map((node) => node.id))
    return {
      ...one,
      graph: {
        nodes: one.graph.nodes.filter((node) => !left.has(node.id)),
        edges: one.graph.edges.filter((edge) => !left.has(edge.from) && !left.has(edge.to)),
      },
    }
  }

  const cuts = RELATIONS_CUT.map((relation) => ({
    relation,
    rounds: cases.flatMap((one, round) => {
      const frame = arrangeCase(one)
      if (!frame.overflow[relation]) return []
      return [{ round, one, shown: countByRelation(frame.nodes, relation) }]
    }),
  }))

  it('is cut in a window of every kind of relation, and not in every window', () => {
    for (const { relation, rounds } of cuts) expect(rounds.length, relation).toBeGreaterThanOrEqual(5)
    const whole = cases.filter((one) => Object.keys(arrangeCase(one).overflow).length === 0)
    expect(whole.length).toBeGreaterThanOrEqual(10)
  })

  it.each(cuts.map(({ relation, rounds }) => [relation, rounds] as const))(
    'leaves no room for the next %s note: the arrangement of the graph with that note cut to shown + 1 cuts it again',
    (relation, rounds) => {
      const stillFits: string[] = []
      for (const { round, one, shown } of rounds) {
        const again = arrangeCase(cutNodes(one, relation, shown + 1))
        if (!again.overflow[relation]) stillFits.push(`case ${round} ${relation} ${shown}`)
      }
      expect(stillFits).toStrictEqual([])
    },
  )
})

describe('a window too small for the boxes', () => {
  const nodes: PlexNode[] = [
    { id: 'focus', title: 'Focus', relation: 'focus' },
    ...RELATIONS_CUT.flatMap((relation) =>
      [1, 2, 3].map((index) => ({ id: `${relation}-${index}`, title: `${relation} ${index}`, relation })),
    ),
  ]
  const graph: PlexGraph = { nodes, edges: [] }
  const arrangeIn = (viewport: { width: number; height: number }) =>
    arrangePlex(graph, { options: { viewport }, measure: () => 120 })

  // Rule: a note other than the focus is shown inside the window or counted in the overflow; the focus is drawn wherever the window is.
  it.each([
    ['below 480 by 360', { width: 300, height: 240 }],
    ['very small', { width: 160, height: 120 }],
    ['narrower than one box', { width: 90, height: 600 }],
    ['not as high as one box', { width: 800, height: 20 }],
    ['narrower than the focus', { width: 100, height: 400 }],
  ])('shows no note outside a window %s, apart from the focus', (_, viewport) => {
    const frame = arrangeIn(viewport)
    const { margin } = resolveOptions({ viewport })
    const shown = frame.nodes.filter((node) => node.relation !== 'focus')

    expect(frame.nodes.filter((node) => node.relation === 'focus')).toHaveLength(1)
    for (const node of shown) {
      expect(Math.abs(node.x) + node.width / 2).toBeLessThanOrEqual(viewport.width / 2 - margin + 1)
      expect(Math.abs(node.y) + node.height / 2).toBeLessThanOrEqual(viewport.height / 2 - margin + 1)
    }
    for (const relation of RELATION_NAMES) {
      expect(countByRelation(shown, relation) + (frame.overflow[relation] ?? 0), relation).toBe(
        countByRelation(nodes, relation),
      )
    }
  })

  it('still draws a focus that is wider than the window', () => {
    const viewport = { width: 100, height: 400 }
    const focus = arrangeIn(viewport).nodes.find((node) => node.relation === 'focus')!

    expect(focus.width).toBeGreaterThan(viewport.width)
  })
})
