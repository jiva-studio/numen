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
import {
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

export interface ArrangeInput {
  readonly options?: PlexOptionsInput | undefined
  readonly placement?: Placement | undefined
  readonly measure?: ((node: PlexNode) => number) | undefined
  readonly measureLabel?: ((label: string) => number) | undefined
  readonly labelType?: LabelType | undefined
  /** The space each note keeps over and under its pill; absent, a note keeps none. */
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

  // Lays the nodes out at the given options: the focus, then the placement.
  // A caller that does not need every box passes `shouldStop`.
  const lay = (
    laid: PlexOptions,
    nodesByRelation: NodesByRelation,
    shouldStop?: PlacementContext['shouldStop'],
  ) => {
    const width = createWidthLookup(laid, measure)
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
        room: (node) => rooms.get(node.id) ?? NO_ROOM,
        shouldStop,
      }),
    }
  }

  // Packed as closely as this window needs and no closer, so a box narrows
  // and a gap closes before a note is given up.
  const resolved = measureCrowding(request, (packed) =>
    isHeld(
      lay(packed, requested, stopAtOutside(packed.viewport, packed.margin, rooms)).placed,
      packed.viewport,
      { margin: packed.margin, rooms },
    ),
  )

  // Admission is settled at that packing, and the gaps then expand into the
  // remaining space. Both are measured against the arrangement itself, so a
  // placement of any shape fits the window.
  const { viewport, margin } = resolved
  const admitted = viewport
    ? admitNodes(
        requested,
        (nodesByRelation, shouldStop) => lay(resolved, nodesByRelation, shouldStop).placed,
        (node) => isInside(node, viewport, { margin, rooms }),
      )
    : requested
  const overflow = countOverflow(graph.nodes, admitted)

  const spacing = findSpacing(resolved, (candidate, window) =>
    isHeld(
      lay({ ...resolved, ...candidate }, admitted, stopAtOutside(window, margin, rooms)).placed,
      window,
      { margin, rooms },
    ),
  )
  const { focus, placed } = lay({ ...resolved, ...spacing }, admitted)
  const nodes = [focus, ...placed]

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const edgeRouting = createRouting(resolved, measureLabel, labelType)
  const edges = separateTitles(routeEdges(graph.edges, byId, edgeRouting), nodes, edgeRouting)

  return { nodes, edges, extent: getExtent(nodes), overflow }
}

function createWidthLookup(
  options: PlexOptions,
  measure: ((node: PlexNode) => number) | undefined,
): Widths {
  return (node) => {
    const isFocus = node.relation === 'focus'
    const widest = isFocus ? options.focusSize.width : options.nodeSize.width
    if (!measure) return widest
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

/** Whether the footprint of a note, its pill and the room around it, lies inside the window less the margin. */
function isInside(node: PlacedNode, viewport: Size, { margin, rooms }: Fit): boolean {
  const { top, bottom, width } = rooms.get(node.id) ?? NO_ROOM
  const halfWidth = Math.max(node.width, width) / 2
  const isHeldAcross = Math.abs(node.x) + halfWidth <= viewport.width / 2 - margin
  if (top === 0 && bottom === 0) {
    return isHeldAcross && Math.abs(node.y) + node.height / 2 <= viewport.height / 2 - margin
  }
  return (
    isHeldAcross &&
    node.y - node.height / 2 - top >= -viewport.height / 2 + margin &&
    node.y + node.height / 2 + bottom <= viewport.height / 2 - margin
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
