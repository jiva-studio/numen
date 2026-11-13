/** Random windows, counts, widths and rooms: what the arrangement shows is what fits, and a note is not cut while there is room. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { clusters } from './layout/clusters'
import { admitNodes, countOverflow, requestNodes } from './layout/admission'
import { SpotIndex } from './layout/freeSpot'
import { getFootprint, getFootprintCenterY, type NodeRoom } from './layout/placement'
import { applyRoom } from './look'
import { resolveOptions } from './options'
import { RELATION_NAMES } from '../relation'
import type { PlacedNode } from '../node'
import { createRandom } from '../../fixtures/random'
import type { PlexGraph, PlexNode, PlexRelation } from '../../../types'

const ROUNDS = 120
const RELATIONS_DRAWN: readonly PlexRelation[] = ['parent', 'child', 'jump', 'reference']

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
  const widths = new Map<string, number>()
  const rooms = new Map<string, NodeRoom>()
  for (const relation of RELATIONS_DRAWN) {
    for (let index = whole(0, 22); index > 0; index--) {
      const id = `${relation}-${index}`
      nodes.push({ id, title: id, relation })
      widths.set(id, whole(60, 190))
      const bottom = whole(0, 3) * 14
      rooms.set(id, { top: 0, bottom, width: bottom === 0 ? 0 : whole(80, 170) })
    }
  }
  return { graph: { nodes, edges: [] }, viewport, widths, rooms }
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

describe('what is shown of a graph', () => {
  it.each(cases.map((one, round) => [round, one] as const))(
    'adds up to what arrived, and fits the window, in case %i',
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
      const index = new SpotIndex([])
      const overlaps = boxes.filter((box) => {
        const isFree = index.isFree(
          { ...box, width: box.width - 2, height: box.height - 2 },
          { gap: 0, lineGap: 0 },
        )
        index.add(box)
        return !isFree
      })
      expect(overlaps).toStrictEqual([])
    },
  )
})

/** The arrangement with the packing and the spacing left out, so the window decides what is admitted, as it does in the arrangement. */
function admitAndPlace(one: Case) {
  const options = applyRoom(resolveOptions({ viewport: one.viewport }))
  const { viewport, margin } = options
  const fits = (spot: { x: number; y: number; width: number; height: number }) =>
    Math.abs(spot.x) + spot.width / 2 <= viewport!.width / 2 - margin &&
    Math.abs(spot.y) + spot.height / 2 <= viewport!.height / 2 - margin
  const isInside = (node: PlacedNode) => fits(footprintOf(node, one.rooms))
  const focus: PlacedNode = {
    id: 'focus',
    title: 'Focus',
    relation: 'focus',
    x: 0,
    y: 0,
    width: options.focusSize.width,
    height: options.focusSize.height,
    order: 0,
    opacity: 1,
  }
  const place = (
    nodesByRelation: Parameters<typeof clusters.place>[0],
    shouldStop?: (node: PlacedNode) => boolean,
  ) =>
    clusters.place(nodesByRelation, focus, {
      options,
      width: (node) => Math.min(one.widths.get(node.id) ?? 100, options.nodeSize.width),
      room: (node) => one.rooms.get(node.id) ?? { top: 0, bottom: 0, width: 0 },
      shouldStop,
      fits,
    })

  const requested = requestNodes(one.graph.nodes, options)
  const admitted = admitNodes(
    requested,
    (nodesByRelation, shouldStop) => place(nodesByRelation, shouldStop),
    isInside,
  )
  return { admitted, place, isInside, overflow: countOverflow(one.graph.nodes, admitted) }
}

describe('a note that is cut', () => {
  it('leaves no room for it: laying the next note of its relation as well puts a box outside the window', () => {
    const cutWhileRoomLeft: string[] = []
    let cuts = 0
    cases.forEach((one, round) => {
      const { admitted, place, isInside, overflow } = admitAndPlace(one)
      for (const relation of RELATION_NAMES) {
        if (!overflow[relation]) continue
        cuts++
        const members = one.graph.nodes.filter((node) => node.relation === relation)
        const next = members[admitted[relation]?.length ?? 0]!
        const more = { ...admitted, [relation]: [...(admitted[relation] ?? []), next] }
        if (place(more).every(isInside))
          cutWhileRoomLeft.push(`case ${round} ${relation} ${members.length}`)
      }
    })

    expect(cuts).toBeGreaterThan(20)
    expect(cutWhileRoomLeft).toStrictEqual([])
  })
})
