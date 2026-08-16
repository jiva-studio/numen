/**
 * One frame between two arrangements. Pure, so a moment partway through a
 * movement is a value with a test rather than something to be caught.
 *
 * Nodes are matched by identifier alone, which is why this needs no knowledge
 * of what a node stands for.
 */
import {
  edgeKey,
  type PlacedEdge,
  type PlacedNode,
  type PlexEdge,
  type PlexFrame,
} from '../model'
import { clamp01, lerp, lerpExtent } from './math'
import { resolveOptions, type PlexOptionsInput } from './options'
import { routeEdges, routingFor } from './routing'

export function interpolatePlex(
  from: PlexFrame,
  to: PlexFrame,
  progress: number,
  options?: PlexOptionsInput,
): PlexFrame {
  const t = clamp01(progress)
  if (t <= 0) return from
  if (t >= 1) return to

  const resolved = resolveOptions(options)
  const was = new Map(from.nodes.map((node) => [node.id, node]))
  const will = new Map(to.nodes.map((node) => [node.id, node]))

  // Arrivals unfold from where the new focus started, rather than fading in on
  // top of the picture.
  const newFocus = to.nodes.find((node) => node.seat === 'focus')
  const cameFrom = newFocus ? was.get(newFocus.id) : undefined
  const source = { x: cameFrom?.x ?? 0, y: cameFrom?.y ?? 0 }

  const { arriveAfter, leaveBefore } = resolved.motion
  const arriving = clamp01((t - arriveAfter) / (1 - arriveAfter))
  const leaving = 1 - clamp01(t / leaveBefore)

  const nodes: PlacedNode[] = []

  for (const target of to.nodes) {
    const start = was.get(target.id)
    if (start) {
      nodes.push({
        ...target,
        x: lerp(start.x, target.x, t),
        y: lerp(start.y, target.y, t),
        width: lerp(start.width, target.width, t),
        height: lerp(start.height, target.height, t),
        opacity: 1,
      })
    } else {
      nodes.push({
        ...target,
        x: lerp(source.x, target.x, t),
        y: lerp(source.y, target.y, t),
        opacity: arriving,
      })
    }
  }

  // Departures stay put and fade; moving them too would be a third thing
  // travelling in a picture meant to be followed by eye.
  for (const gone of from.nodes) {
    if (will.has(gone.id)) continue
    nodes.push({ ...gone, opacity: leaving })
  }

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const wasEdge = new Set(from.edges.map(edgeKey))
  const willEdge = new Set(to.edges.map(edgeKey))

  const both: PlexEdge[] = [...to.edges]
  for (const edge of from.edges) {
    if (!willEdge.has(edgeKey(edge))) both.push(edge)
  }

  const edgeOpacity = (edge: PlexEdge): number => {
    const key = edgeKey(edge)
    if (wasEdge.has(key) && willEdge.has(key)) return 1
    return willEdge.has(key) ? arriving : leaving
  }

  const edges: PlacedEdge[] = routeEdges(
    both,
    byId,
    routingFor(resolved),
    edgeOpacity,
  )

  return {
    nodes,
    edges,
    extent: lerpExtent(from.extent, to.extent, t),
    overflow: to.overflow,
  }
}
