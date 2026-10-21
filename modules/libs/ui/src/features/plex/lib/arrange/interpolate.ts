/**
 * One frame between two arrangements. Pure, so a moment partway through a
 * movement is a value with a test.
 *
 * Nodes are matched by identifier alone, and nothing here knows what a node
 * stands for.
 */
import { edgeKey, type PlacedEdge, type PlexEdge } from '../edge'
import type { PlexFrame } from '../frame'
import type { PlacedNode, Position } from '../node'
import { clamp01, lerp, lerpExtent } from './math'
import { resolveOptions, type PlexOptionsInput } from './options'
import { routeEdges, createRouting, type Routing } from './routing'

/** What a settled arrangement says about the title an edge carries. */
type Title = Pick<PlacedEdge, 'heading' | 'words' | 'wordsAt'>

const createTitle = (edge: PlacedEdge): Title => ({
  heading: edge.heading,
  words: edge.words,
  wordsAt: edge.wordsAt,
})

/** Where an arriving node unfolds from: the place the new focus started at. */
function getOrigin(to: PlexFrame, was: ReadonlyMap<string, PlacedNode>): Position {
  const newFocus = to.nodes.find((node) => node.relation === 'focus')
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

interface NodeTweenOptions {
  readonly t: number
  readonly opacity: number
}

/** A node that was not there before, unfolding from the origin. */
function openNode(target: PlacedNode, origin: Position, options: NodeTweenOptions): PlacedNode {
  return {
    ...target,
    x: lerp(origin.x, target.x, options.t),
    y: lerp(origin.y, target.y, options.t),
    opacity: options.opacity,
  }
}

interface NodeTiming {
  readonly t: number
  readonly entry: number
  readonly exit: number
}

/** Every node of both arrangements, currently partway through the movement. */
function tweenNodes(from: PlexFrame, to: PlexFrame, timing: NodeTiming): PlacedNode[] {
  const was = new Map(from.nodes.map((node) => [node.id, node]))
  const will = new Set(to.nodes.map((node) => node.id))
  const origin = getOrigin(to, was)

  const nodes: PlacedNode[] = []

  for (const target of to.nodes) {
    const start = was.get(target.id)
    nodes.push(
      start
        ? tweenNode(start, target, timing.t)
        : openNode(target, origin, { t: timing.t, opacity: timing.entry }),
    )
  }

  // Departures stay put and fade; moving them too would be a third thing
  // travelling in a picture meant to be followed by eye.
  for (const gone of from.nodes) {
    if (will.has(gone.id)) continue
    nodes.push({ ...gone, opacity: timing.exit })
  }

  return nodes
}

interface EdgeTweenContext {
  readonly byId: ReadonlyMap<string, PlacedNode>
  readonly routing: Routing
  readonly entryOpacity: number
  readonly exitOpacity: number
}

/** Every edge of both arrangements, routed against the nodes as they now stand. */
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
  const settled = new Map<string, Title>()
  for (const edge of from.edges) settled.set(edgeKey(edge), createTitle(edge))
  for (const edge of to.edges) settled.set(edgeKey(edge), createTitle(edge))

  return routeEdges(both, byId, { routing, opacity: edgeOpacity }).map((edge) => ({
    ...edge,
    ...settled.get(edgeKey(edge)),
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
  const arriving = clamp01((t - arriveAfter) / (1 - arriveAfter))
  const leaving = 1 - clamp01(t / leaveBefore)

  const nodes = tweenNodes(from, to, { t, entry: arriving, exit: leaving })
  const byId = new Map(nodes.map((node) => [node.id, node]))

  return {
    nodes,
    edges: tweenEdges(from, to, {
      byId,
      routing: createRouting(resolved),
      entryOpacity: arriving,
      exitOpacity: leaving,
    }),
    extent: lerpExtent(from.extent, to.extent, t),
    overflow: to.overflow,
  }
}
