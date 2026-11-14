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
import { measureCrowding, packOptions } from './layout/crowding'
import { applyRoom } from './look'
import { resolveOptions, type PlexOptions, type PlexOptionsInput, type Size } from './options'
import type { Spot } from './layout/spotIndex'
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
import type { PlexGraph, PlexNode, PlexRelation } from '../../../types'

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
    roomsById: Fit['rooms'],
    {
      isLineKept = false,
      ...limits
    }: Pick<PlacementContext, 'shouldStop' | 'fits'> & {
      readonly isLineKept?: boolean
    } = {},
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
        room: (node) => roomsById.get(node.id) ?? NO_ROOM,
        ...(isLineKept ? { edges: graph.edges } : {}),
        ...limits,
      }),
    }
  }

  // Packed as closely as this window needs and no closer, so a gap closes, and without `measure` a box
  // narrows, before a note is given up. With `measure` a box keeps the width of its title: packing does not narrow it. The probes of the packing and of the gaps lay out
  // without `fits`; with it they pack differently, and arrange.frames.test.ts records the packing
  // they choose. The names are left out of it:
  // a pill is the same size with names as without, and the names then cost notes.
  const crowded = measureCrowding(request, (packed) =>
    isInsideWindow(
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
  const { viewport, margin } = crowded
  const admitAt = (packing: PlexOptions) =>
    viewport
      ? admitNodes(
          requested,
          (nodesByRelation, shouldStop) =>
            lay(packing, nodesByRelation, rooms, { shouldStop, fits: getFits(packing) }).placed,
          (node) => isInside(node, viewport, { margin, rooms }),
        )
      : requested
  // The probes of the packing leave out the rooms, so a packing they find loose can still cut notes that the tightest packing holds.
  const tightest = packOptions(request, 0)
  const admittedCrowded = admitAt(crowded)
  const isTightestBetter =
    viewport !== undefined &&
    crowded.gap > tightest.gap &&
    countNodes(admittedCrowded) < countNodes(requested)
  const admittedTightest = isTightestBetter ? admitAt(tightest) : admittedCrowded
  const isTightestChosen = countNodes(admittedTightest) > countNodes(admittedCrowded)
  const resolved = isTightestChosen ? tightest : crowded
  const admitted = isTightestChosen ? admittedTightest : admittedCrowded
  const admittedOverflow = countOverflow(graph.nodes, admitted)

  // Wider gaps stretch the arrangement and do not rearrange it: a note stays on its side of the focus.
  let sides: ReadonlyMap<string, string> | undefined
  const spacing = findSpacing(resolved, (candidate, window) => {
    sides ??= getSides(lay(resolved, admitted, rooms).placed)
    const laid = lay({ ...resolved, ...candidate }, admitted, rooms, {
      shouldStop: stopAtOutside(window, margin, rooms),
    }).placed
    return isInsideWindow(laid, window, { margin, rooms }) && isOnSides(laid, sides)
  })
  const fit: Fit = { margin, rooms }
  // The lines are kept apart only here and only in a window: what the window holds is settled without them, and a layout
  // that keeps them apart but leaves the window gives way to the one that does not.
  const spacedOptions = { ...resolved, ...spacing }
  const spaced = lay(spacedOptions, admitted, rooms, {
    fits: getFits(resolved),
    isLineKept: viewport !== undefined,
  })
  // The probes of the gaps lay out without `fits`, so the spaced layout can leave the window where the one admission verified does not.
  const getFinalLayout = () => {
    if (isInsideWindow(spaced.placed, viewport, fit)) return spaced
    const spacedPlain = lay(spacedOptions, admitted, rooms, { fits: getFits(resolved) })
    if (isInsideWindow(spacedPlain.placed, viewport, fit)) return spacedPlain
    return lay(resolved, admitted, rooms, { fits: getFits(resolved) })
  }
  const finalLayout = getFinalLayout()
  const { focus } = finalLayout
  const placed = viewport
    ? finalLayout.placed.filter((node) => isInside(node, viewport, fit))
    : finalLayout.placed
  const overflow = addDropped(admittedOverflow, finalLayout.placed, placed)
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
    const measured = measure(node)
    if (!Number.isFinite(measured)) {
      return isFocus ? options.focusSize.width : options.nodeSize.width
    }
    const box = Math.max(options.minWidth, measured)
    return Math.min(widest, isFocus ? box * options.focusScale : box)
  }
}

/** Which side of the focus a note is on, across and up or down, by id. */
function getSides(nodes: readonly PlacedNode[]): ReadonlyMap<string, string> {
  const side = (value: number) => (Math.abs(value) < 0.5 ? 0 : Math.sign(value))
  return new Map(nodes.map((node) => [node.id, `${side(node.x)} ${side(node.y)}`]))
}

function isOnSides(nodes: readonly PlacedNode[], sides: ReadonlyMap<string, string>): boolean {
  const now = getSides(nodes)
  return [...now].every(([id, side]) => sides.get(id) === side)
}

function countNodes(nodesByRelation: NodesByRelation): number {
  return Object.values(nodesByRelation).reduce((total, nodes) => total + nodes.length, 0)
}

function addDropped(
  overflow: Partial<Record<PlexRelation, number>>,
  laid: readonly PlacedNode[],
  inside: readonly PlacedNode[],
): Partial<Record<PlexRelation, number>> {
  const insideIds = new Set(inside.map((node) => node.id))
  const total = { ...overflow }
  for (const node of laid) {
    if (insideIds.has(node.id) || node.relation === 'focus') continue
    total[node.relation] = (total[node.relation] ?? 0) + 1
  }
  return total
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

function isFootprintInside(spot: Spot, viewport: Size, margin: number): boolean {
  return (
    Math.abs(spot.x) + spot.width / 2 <= viewport.width / 2 - margin &&
    Math.abs(spot.y) + spot.height / 2 <= viewport.height / 2 - margin
  )
}

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

function isInsideWindow(
  nodes: readonly PlacedNode[],
  viewport: Size | undefined,
  fit: Fit,
): boolean {
  return !viewport || nodes.every((node) => isInside(node, viewport, fit))
}
