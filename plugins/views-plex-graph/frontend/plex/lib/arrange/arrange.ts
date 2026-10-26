/**
 * Turn a neighbourhood into a frame: canPlace, place, route.
 *
 * Pure — the same inputs give the same numbers on any machine. Nothing here
 * reads the clock, measures text or touches the DOM: how wide a title is
 * arrives as a number, from whoever is drawing it.
 */
import { getExtent, type PlexFrame } from '../frame'
import { assertNeighbourhood, type PlexNeighbourhood } from '../neighbourhood'
import type { PlacedNode, PlexNode } from '../node'
import { RELATION_NAMES, type PlexRelation } from '../relation'
import { measureCrowding } from './layout/crowding'
import { getLimits, type Limits } from './layout/limits'
import { resolveOptions, type PlexOptions, type PlexOptionsInput, type Size } from './options'
import {
  rowsAndColumns,
  type Placement,
  type NodesByRelation,
  type Widths,
} from './layout/placement'
import { routeEdges, createRouting } from './edges/routing'
import { findSpacing, type Spacing } from './layout/spacing'
import { separateTitles } from './edges/titles'

export interface ArrangeInput {
  readonly options?: PlexOptionsInput | undefined
  /** Defaults to rows and columns — the arrangement a plex is recognised by. */
  readonly placement?: Placement | undefined
  /**
   * The width a node's box needs to hold its title, padding included. Text is
   * measured where the plex is drawn; here it arrives as a number. Without one
   * every box is drawn at its widest.
   */
  readonly measure?: ((node: PlexNode) => number) | undefined
  /**
   * The width a label's words need on their line, measured where the plex is
   * drawn. Without one a label is written at whatever length it has.
   */
  readonly measureLabel?: ((label: string) => number) | undefined
  /**
   * How deep a label's line stands, across the line it is set on, measured
   * where the plex is drawn. A label is set in a type of its own, and one
   * given no depth takes up none.
   */
  readonly labelDepth?: number | undefined
}

export function arrangePlex(
  neighbourhood: PlexNeighbourhood,
  { options, placement = rowsAndColumns, measure, measureLabel, labelDepth }: ArrangeInput = {},
): PlexFrame {
  const request = resolveOptions(options)
  const focusNode = assertNeighbourhood(neighbourhood)
  const counts = countRelations(neighbourhood.nodes)

  // Packed as closely as this window needs and no closer, so a box narrows
  // and a gap closes before a relation is given up.
  const resolved = measureCrowding(request, counts)
  const getWidth = createWidthLookup(resolved, measure)

  const focus: PlacedNode = {
    ...focusNode,
    x: 0,
    y: 0,
    width: getWidth(focusNode),
    height: resolved.focusSize.height,
    order: 0,
    opacity: 1,
  }

  const limits = getLimits(resolved, counts)

  const { nodesByRelation, overflow } = canPlace(neighbourhood.nodes, limits)

  // How much is admitted is settled at that packing, and the gaps then open
  // into the room that is left. The opening is measured against the
  // arrangement itself, so a placement of any shape keeps the window.
  const lay = (spacing: Spacing) =>
    placement.place(nodesByRelation, focus, {
      options: { ...resolved, ...spacing },
      limits,
      width: getWidth,
    })
  const spacing = findSpacing(resolved, (candidate, viewport) =>
    isInsideViewport(lay(candidate), viewport, resolved.margin),
  )

  const nodes = [focus, ...lay(spacing)]

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const edgeRouting = createRouting(resolved, measureLabel, labelDepth)
  const edges = separateTitles(
    routeEdges(neighbourhood.edges, byId, edgeRouting),
    nodes,
    edgeRouting,
  )

  return { nodes, edges, extent: getExtent(nodes), overflow }
}

/**
 * How wide each box is drawn: what the measurer asks for, held between the
 * minimum and the widest the node's relation allows. The focus is measured as
 * every other node is, against its own width.
 */
function createWidthLookup(
  options: PlexOptions,
  measure: ((node: PlexNode) => number) | undefined,
): Widths {
  return (node) => {
    const widest = node.relation === 'focus' ? options.focusSize.width : options.nodeSize.width
    if (!measure) return widest
    return Math.min(widest, Math.max(options.minWidth, measure(node)))
  }
}

/** Whether every node stays inside the window, the margin kept clear. */
function isInsideViewport(nodes: readonly PlacedNode[], viewport: Size, margin: number): boolean {
  const halfWidth = viewport.width / 2 - margin
  const halfHeight = viewport.height / 2 - margin
  return nodes.every(
    (node) =>
      Math.abs(node.x) + node.width / 2 <= halfWidth &&
      Math.abs(node.y) + node.height / 2 <= halfHeight,
  )
}

function countRelations(nodes: readonly PlexNode[]): Record<PlexRelation, number> {
  return Object.fromEntries(
    RELATION_NAMES.map((relation) => [
      relation,
      nodes.filter((node) => node.relation === relation).length,
    ]),
  ) as Record<PlexRelation, number>
}

/**
 * How much of each relation the picture holds. Settled before a strategy is
 * called, so the overflow count means the same thing whichever one draws it.
 */
function canPlace(
  nodes: readonly PlexNode[],
  limits: Limits,
): { nodesByRelation: NodesByRelation; overflow: Partial<Record<PlexRelation, number>> } {
  const nodesByRelation: Partial<Record<PlexRelation, readonly PlexNode[]>> = {}
  const overflow: Partial<Record<PlexRelation, number>> = {}

  for (const relation of RELATION_NAMES) {
    const ofRelation = nodes.filter((node) => node.relation === relation)
    if (ofRelation.length === 0) continue

    const capacity = limits[relation].perLine * limits[relation].lines
    const visibleRelations = ofRelation.slice(0, capacity)
    if (ofRelation.length > visibleRelations.length)
      overflow[relation] = ofRelation.length - visibleRelations.length
    nodesByRelation[relation] = visibleRelations
  }

  return { nodesByRelation, overflow }
}
