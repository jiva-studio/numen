/**
 * Turn a graph into a frame: request, fit to the window, admit, place, route.
 *
 * Pure — the same inputs give the same numbers on any machine. Nothing here
 * reads the clock, measures text or touches the DOM: the width of a title is
 * supplied as a number by the renderer.
 */
import type { LabelType } from '../edge'
import { getExtent, type PlexFrame } from '../frame'
import { assertGraph } from '../graph'
import type { PlacedNode } from '../node'
import { admitNodes, countOverflow, requestNodes } from './layout/admission'
import { clusters } from './layout/clusters'
import { measureCrowding } from './layout/crowding'
import { applyRoom } from './look'
import { resolveOptions, type PlexOptions, type PlexOptionsInput, type Size } from './options'
import type { Spot } from './layout/freeSpot'
import {
  getFootprint,
  getFootprintCenterY,
  NO_ROOM,
  type NodeRoom,
  type Placement,
  type NodesByRelation,
  type PlacementContext,
  type Rooms,
  type Widths,
} from './layout/placement'
import { routeEdges, createRouting } from './edges/routing'
import { findSpacing } from './layout/spacing'
import { separateTitles } from './edges/titles'
import type { PlexGraph, PlexNode } from '../../../types'

/** `room` is the space a note keeps over and under its pill; a note without one keeps none. */
export interface ArrangeInput {
  readonly options?: PlexOptionsInput | undefined
  readonly placement?: Placement | undefined
  readonly measure?: ((node: PlexNode) => number) | undefined
  readonly measureLabel?: ((label: string) => number) | undefined
  readonly labelType?: LabelType | undefined
  readonly room?: Rooms | undefined
}

export function arrangePlex(
  graph: PlexGraph,
  { options, placement = clusters, measure, measureLabel, labelType, room }: ArrangeInput = {},
): PlexFrame {
  const request = applyRoom(resolveOptions(options))
  const focusNode = assertGraph(graph)
  const requested = requestNodes(graph.nodes, request)
  const rooms = createRoomLookup(graph.nodes, room)

  // Lays the nodes out at the given options, each keeping the room it is given: the focus, then the placement.
  // A caller that does not need every box passes `shouldStop`.
  const lay = (
    laid: PlexOptions,
    nodesByRelation: NodesByRelation,
    kept: Fit['rooms'],
    limits: Pick<PlacementContext, 'shouldStop' | 'fits'> = {},
  ) => {
    const width = createWidthLookup(laid, measure, request)
    const focus: PlacedNode = {
      ...focusNode,
      x: 0,
      y: 0,
      width: width(focusNode),
      height: laid.focusSize.height,
      order: 0,
      opacity: 1,
    }
    return {
      focus,
      placed: placement.place(nodesByRelation, focus, {
        options: laid,
        width,
        room: (node) => kept.get(node.id) ?? NO_ROOM,
        ...limits,
      }),
    }
  }

  // Packed as closely as this window needs and no closer, so a box narrows
  // and a gap closes before a note is given up. The names are left out of it:
  // a pill is the same size with names as without, and the names then cost notes.
  const resolved = measureCrowding(request, (packed) =>
    isHeld(
      lay(packed, requested, NO_ROOMS, {
        shouldStop: stopAtOutside(packed.viewport, packed.margin, NO_ROOMS),
      }).placed,
      packed.viewport,
      { margin: packed.margin, rooms: NO_ROOMS },
    ),
  )

  // Admission is settled at that packing, and the gaps then expand into the
  // remaining space. Both are measured against the arrangement itself, so a
  // placement of any shape fits the window.
  const { viewport, margin } = resolved
  const admitted = viewport
    ? admitNodes(
        requested,
        (nodesByRelation, shouldStop) =>
          lay(resolved, nodesByRelation, rooms, { shouldStop, fits: getFits(resolved) }).placed,
        (node) => isInside(node, viewport, { margin, rooms }),
      )
    : requested
  const overflow = countOverflow(graph.nodes, admitted)

  const spacing = findSpacing(resolved, (candidate, window) =>
    isHeld(
      lay({ ...resolved, ...candidate }, admitted, rooms, {
        shouldStop: stopAtOutside(window, margin, rooms),
      }).placed,
      window,
      { margin, rooms },
    ),
  )
  const { focus, placed } = lay({ ...resolved, ...spacing }, admitted, rooms, {
    fits: getFits(resolved),
  })
  const nodes = [focus, ...placed]

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const edgeRouting = createRouting(resolved, measureLabel, labelType)
  const edges = separateTitles(routeEdges(graph.edges, byId, edgeRouting), nodes, edgeRouting)

  return { nodes, edges, extent: getExtent(nodes), overflow }
}

function createWidthLookup(
  options: PlexOptions,
  measure: ((node: PlexNode) => number) | undefined,
  settings: PlexOptions,
): Widths {
  return (node) => {
    const isFocus = node.relation === 'focus'
    if (!measure) return isFocus ? options.focusSize.width : options.nodeSize.width
    const widest = isFocus ? settings.focusSize.width : settings.nodeSize.width
    // The focus pill is the box its title would have as a neighbor, made larger.
    const box = Math.max(options.minWidth, measure(node))
    return Math.min(widest, isFocus ? box * options.focusScale : box)
  }
}

/** What a box must keep clear of the window: the margin, and the room each note keeps around its pill. */
interface Fit {
  readonly margin: number
  readonly rooms: ReadonlyMap<string, NodeRoom>
}

const NO_ROOMS: ReadonlyMap<string, NodeRoom> = new Map()

function createRoomLookup(
  nodes: readonly PlexNode[],
  room: Rooms | undefined,
): ReadonlyMap<string, NodeRoom> {
  const rooms = new Map<string, NodeRoom>()
  if (!room) return rooms
  for (const node of nodes) {
    if (node.relation !== 'focus') rooms.set(node.id, room(node))
  }
  return rooms
}

/** Whether a footprint keeps a margin from the edge of the window. */
function isFootprintInside(spot: Spot, viewport: Size, margin: number): boolean {
  return (
    Math.abs(spot.x) + spot.width / 2 <= viewport.width / 2 - margin &&
    Math.abs(spot.y) + spot.height / 2 <= viewport.height / 2 - margin
  )
}

/**
 * Whether a footprint lies inside the window, which is what a group is given to grow sideways within.
 * Admission and the final layout pass it. The probes of the packing and of the gaps lay out without it; with it they pack differently, and arrange.frames.test.ts records the packing they choose.
 */
function getFits(options: PlexOptions): PlacementContext['fits'] {
  const { viewport, margin } = options
  if (!viewport) return undefined
  return (spot) => isFootprintInside(spot, viewport, margin)
}

function isInside(node: PlacedNode, viewport: Size, { margin, rooms }: Fit): boolean {
  const room = rooms.get(node.id) ?? NO_ROOM
  return isFootprintInside(
    { x: node.x, y: getFootprintCenterY(node.y, room), ...getFootprint(node, room) },
    viewport,
    margin,
  )
}

function stopAtOutside(
  viewport: Size | undefined,
  margin: number,
  rooms: Fit['rooms'],
): PlacementContext['shouldStop'] {
  return viewport ? (node) => !isInside(node, viewport, { margin, rooms }) : undefined
}

function isHeld(nodes: readonly PlacedNode[], viewport: Size | undefined, fit: Fit): boolean {
  return !viewport || nodes.every((node) => isInside(node, viewport, fit))
}
