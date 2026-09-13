/**
 * One frame between two arrangements. Pure, so a moment partway through a
 * movement is a value with a test rather than something to be caught.
 *
 * Nodes are matched by identifier alone, which is why this needs no knowledge
 * of what a node stands for.
 */
import { edgeKey, type PlacedEdge, type PlexEdge } from '../edge'
import type { PlexFrame } from '../frame'
import type { PlacedNode, Position } from '../node'
import { clamp01, lerp, lerpExtent } from './math'
import { resolveOptions, type PlexOptionsInput } from './options'
import { routeEdges, routingFor, type Routing } from './routing'

/** What a settled arrangement says about the title an edge carries. */
type Title = Pick<PlacedEdge, 'heading' | 'words' | 'wordsAt'>

const titleOf = (edge: PlacedEdge): Title => ({
  heading: edge.heading,
  words: edge.words,
  wordsAt: edge.wordsAt,
})

/** Where an arriving node unfolds from: the place the new focus started at. */
function getOrigin(to: PlexFrame, was: ReadonlyMap<string, PlacedNode>): Position {
  const newFocus = to.nodes.find((node) => node.seat === 'focus')
  const cameFrom = newFocus ? was.get(newFocus.id) : undefined
  return { x: cameFrom?.x ?? 0, y: cameFrom?.y ?? 0 }
}

/** A node partway from where it stood to where it is going. */
function tweenNode(start: PlacedNode, target: PlacedNode, t: number): PlacedNode {
  return {
    ...target,
    x: lerp(start.x, target.x, t),
    y: lerp(start.y, target.y, t),
    width: lerp(start.width, target.width, t),
    height: lerp(start.height, target.height, t),
    opacity: 1,
  }
}

/** A node that was not there before, unfolding from the origin. */
function openNode(target: PlacedNode, origin: Position, t: number, opacity: number): PlacedNode {
  return {
    ...target,
    x: lerp(origin.x, target.x, t),
    y: lerp(origin.y, target.y, t),
    opacity,
  }
}

/** Every node of both arrangements, as it stands partway through the movement. */
function tweenNodes(
  from: PlexFrame,
  to: PlexFrame,
  t: number,
  entryOpacity: number,
  exitOpacity: number,
): PlacedNode[] {
  const was = new Map(from.nodes.map((node) => [node.id, node]))
  const will = new Set(to.nodes.map((node) => node.id))
  const origin = getOrigin(to, was)

  const nodes: PlacedNode[] = []

  for (const target of to.nodes) {
    const start = was.get(target.id)
    nodes.push(start ? tweenNode(start, target, t) : openNode(target, origin, t, entryOpacity))
  }

  // Departures stay put and fade; moving them too would be a third thing
  // travelling in a picture meant to be followed by eye.
  for (const gone of from.nodes) {
    if (will.has(gone.id)) continue
    nodes.push({ ...gone, opacity: exitOpacity })
  }

  return nodes
}

/** Every edge of both arrangements, routed against the nodes as they now stand. */
function tweenEdges(
  from: PlexFrame,
  to: PlexFrame,
  byId: ReadonlyMap<string, PlacedNode>,
  routing: Routing,
  entryOpacity: number,
  exitOpacity: number,
): PlacedEdge[] {
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
  const settled = new Map<string, Title>()
  for (const edge of from.edges) settled.set(edgeKey(edge), titleOf(edge))
  for (const edge of to.edges) settled.set(edgeKey(edge), titleOf(edge))

  return routeEdges(both, byId, routing, edgeOpacity).map((edge) => ({
    ...edge,
    ...settled.get(edgeKey(edge)),
  }))
}

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
  const { arriveAfter, leaveBefore } = resolved.motion
  const arriving = clamp01((t - arriveAfter) / (1 - arriveAfter))
  const leaving = 1 - clamp01(t / leaveBefore)

  const nodes = tweenNodes(from, to, t, arriving, leaving)
  const byId = new Map(nodes.map((node) => [node.id, node]))

  return {
    nodes,
    edges: tweenEdges(from, to, byId, routingFor(resolved), arriving, leaving),
    extent: lerpExtent(from.extent, to.extent, t),
    overflow: to.overflow,
  }
}
