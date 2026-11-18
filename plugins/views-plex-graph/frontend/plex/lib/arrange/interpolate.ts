/**
 * One frame between two arrangements. Pure, so a moment partway through a
 * movement is a value with a test.
 *
 * Nodes are matched by identifier alone; nothing here depends on what a node
 * represents.
 */
import { edgeKey, type PlacedEdge } from '../edge'
import type { PlexFrame } from '../frame'
import type { PlacedNode, Position } from '../node'
import type { Size } from './options'
import { clamp01, lerp, lerpExtent } from './math'
import { resolveOptions, type PlexOptionsInput } from './options'
import { createRouting, type Routing } from './edges/routing'
import { routeEdges } from './edges/routeEdges'
import type { PlexEdge } from '../../../types'

/** What a settled arrangement specifies for the title an edge carries. */
type Title = Pick<PlacedEdge, 'titles'>

const getTitles = (edge: PlacedEdge): Title => ({ titles: edge.titles })

interface Origin extends Position {
  readonly width?: number | undefined
  readonly height?: number | undefined
}

function getOrigin(to: PlexFrame, was: ReadonlyMap<string, PlacedNode>): Origin {
  const newFocus = to.nodes.find((node) => node.relation === 'focus')
  const cameFrom = newFocus ? was.get(newFocus.id) : undefined
  return {
    x: cameFrom?.x ?? 0,
    y: cameFrom?.y ?? 0,
    width: cameFrom?.width,
    height: cameFrom?.height,
  }
}

function getSize(node: PlacedNode): Size {
  return { width: node.width, height: node.height }
}

function getCentre(to: PlexFrame): Position {
  const centre = to.nodes.find((node) => node.relation === 'focus')
  return { x: centre?.x ?? 0, y: centre?.y ?? 0 }
}

function tweenNode(start: PlacedNode, target: PlacedNode, t: number): PlacedNode {
  return {
    ...target,
    x: lerp(start.x, target.x, t),
    y: lerp(start.y, target.y, t),
    width: lerp(start.width, target.width, t),
    height: lerp(start.height, target.height, t),
    opacity: lerp(start.opacity, target.opacity, t),
    settled: getSize(target),
  }
}

function openNode(target: PlacedNode, origin: Origin, t: number): PlacedNode {
  return {
    ...target,
    x: lerp(origin.x, target.x, t),
    y: lerp(origin.y, target.y, t),
    width: lerp(origin.width ?? target.width, target.width, t),
    height: lerp(origin.height ?? target.height, target.height, t),
    settled: getSize(target),
  }
}

function closeNode(start: PlacedNode, origin: Origin, t: number): PlacedNode {
  return {
    ...start,
    x: lerp(start.x, origin.x, t),
    y: lerp(start.y, origin.y, t),
    opacity: lerp(start.opacity, 0, t),
    settled: start.settled ?? getSize(start),
  }
}

function tweenNodes(from: PlexFrame, to: PlexFrame, t: number): PlacedNode[] {
  const was = new Map(from.nodes.map((node) => [node.id, node]))
  const will = new Set(to.nodes.map((node) => node.id))
  const origin = getOrigin(to, was)

  const nodes: PlacedNode[] = []

  for (const target of to.nodes) {
    const start = was.get(target.id)
    nodes.push(start ? tweenNode(start, target, t) : openNode(target, origin, t))
  }

  for (const removed of from.nodes) {
    if (!will.has(removed.id)) nodes.push(closeNode(removed, getCentre(to), t))
  }

  return nodes
}

interface EdgeTweenContext {
  readonly byId: ReadonlyMap<string, PlacedNode>
  readonly routing: Routing
  readonly entryOpacity: number
  readonly exitOpacity: number
}

function tweenEdges(from: PlexFrame, to: PlexFrame, context: EdgeTweenContext): PlacedEdge[] {
  const { byId, routing, entryOpacity, exitOpacity } = context
  const wasEdge = new Set(from.edges.map(edgeKey))
  const willEdge = new Set(to.edges.map(edgeKey))

  const both: PlexEdge[] = [...to.edges]
  for (const edge of from.edges) {
    if (!willEdge.has(edgeKey(edge))) both.push(edge)
  }

  const edgeOpacity = (edge: PlexEdge): number => {
    const key = edgeKey(edge)
    if (wasEdge.has(key) && willEdge.has(key)) return 1
    return willEdge.has(key) ? entryOpacity : exitOpacity
  }

  // A title belongs to a settled arrangement: the words it was cut to, the way
  // round they are read and where along the line they sit come from where the
  // edge is going, or from where it is leaving for an edge that only rests
  // there. A title held this way stays put for the length of a movement.
  const resolved = new Map<string, Title>()
  for (const edge of from.edges) resolved.set(edgeKey(edge), getTitles(edge))
  for (const edge of to.edges) resolved.set(edgeKey(edge), getTitles(edge))

  return routeEdges(both, byId, { routing, opacity: edgeOpacity }).map((edge) => ({
    ...edge,
    ...resolved.get(edgeKey(edge)),
  }))
}

export type InterpolationOptions = PlexOptionsInput & {
  readonly progress?: number
}

export function interpolatePlex(
  from: PlexFrame,
  to: PlexFrame,
  input: number | InterpolationOptions,
): PlexFrame {
  const progress = typeof input === 'number' ? input : (input.progress ?? 0)
  const options = typeof input === 'number' ? undefined : input

  const t = clamp01(progress)
  if (t <= 0) return from
  if (t >= 1) return to

  const resolved = resolveOptions(options)
  const { arriveAfter, leaveBefore } = resolved.motion
  const incoming = clamp01((t - arriveAfter) / (1 - arriveAfter))
  const leaving = 1 - clamp01(t / leaveBefore)

  const nodes = tweenNodes(from, to, t)
  const byId = new Map(nodes.map((node) => [node.id, node]))

  return {
    nodes,
    edges: tweenEdges(from, to, {
      byId,
      routing: createRouting(resolved),
      entryOpacity: incoming,
      exitOpacity: leaving,
    }),
    extent: lerpExtent(from.extent, to.extent, t),
    overflow: to.overflow,
  }
}
