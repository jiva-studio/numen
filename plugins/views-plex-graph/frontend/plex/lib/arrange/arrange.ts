/**
 * Turn a neighborhood into a frame: canPlace, place, route.
 *
 * Pure — the same inputs give the same numbers on any machine. Nothing here
 * reads the clock, measures text or touches the DOM: the width of a title is
 * supplied as a number by the renderer.
 */
import type { LabelType } from '../edge'
import { getExtent, type PlexFrame } from '../frame'
import { assertNeighborhood } from '../neighborhood'
import type { PlacedNode } from '../node'
import { RELATION_NAMES } from '../relation'
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
import type { PlexGraph, PlexNode, PlexRelation } from '../../../types'

export interface ArrangeInput {
  readonly options?: PlexOptionsInput | undefined
  readonly placement?: Placement | undefined
  readonly measure?: ((node: PlexNode) => number) | undefined
  readonly measureLabel?: ((label: string) => number) | undefined
  readonly labelType?: LabelType | undefined
}

export function arrangePlex(
  neighborhood: PlexGraph,
  { options, placement = rowsAndColumns, measure, measureLabel, labelType }: ArrangeInput = {},
): PlexFrame {
  const request = resolveOptions(options)
  const focusNode = assertNeighborhood(neighborhood)
  const counts = countRelations(neighborhood.nodes)

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

  const { nodesByRelation, overflow } = canPlace(neighborhood.nodes, limits)

  // Admission is settled at that packing, and the gaps then expand into the
  // remaining space. The expansion is measured against the arrangement itself, so
  // a placement of any shape fits the window.
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
  const edgeRouting = createRouting(resolved, measureLabel, labelType)
  const edges = separateTitles(
    routeEdges(neighborhood.edges, byId, edgeRouting),
    nodes,
    edgeRouting,
  )

  return { nodes, edges, extent: getExtent(nodes), overflow }
}

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
