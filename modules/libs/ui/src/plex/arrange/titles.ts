/**
 * Where along its line each title sits.
 *
 * A title stays on its own line: at the middle of it while the middle is free,
 * and slid along the same curve to the nearest free place while it is not.
 * Pure, and worked out in the plex's own coordinates — two titles far apart
 * along their curves can still be one on top of the other in the picture.
 */
import {
  lengthOf,
  rulerOf,
  type PlacedEdge,
  type PlacedNode,
  type Point,
} from '../model'
import { MIDDLE, type Routing } from './routing'

/** An upright box in the plex's own coordinates. */
interface Box {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** How far one try along a line stands from the last, in plex units. */
const STEP = 3

/** How many points along a title's run the box around it is drawn from. */
const RUN_SAMPLES = 8

const meets = (one: Box, other: Box): boolean =>
  one.minX < other.maxX &&
  other.minX < one.maxX &&
  one.minY < other.maxY &&
  other.minY < one.maxY

const boxOf = (node: PlacedNode): Box => ({
  minX: node.x - node.width / 2,
  minY: node.y - node.height / 2,
  maxX: node.x + node.width / 2,
  maxY: node.y + node.height / 2,
})

/**
 * Give every title a place on its line, in the order the edges arrive. Each
 * one is kept clear of the boxes and of every title already settled, so a fan
 * of lines out of one node reads as a list.
 *
 * Where nothing measured the words there is no extent to keep clear of, and
 * every title stays where it was put.
 */
export function settleTitles(
  edges: readonly PlacedEdge[],
  nodes: readonly PlacedNode[],
  routing: Routing,
): PlacedEdge[] {
  const width = routing.labelWidth
  if (!width) return [...edges]

  const boxes = nodes.map(boxOf)
  const titles: Box[] = []

  return edges.map((edge) => {
    if (!edge.words) return edge

    const arc = lengthOf(edge)
    const extent = width(edge.words)
    if (arc <= 0 || extent <= 0) return edge

    const boxAt = titleBoxes(edge, arc, extent, routing.labelHeight)
    const ends = (extent / 2 + (edge.arrow ? routing.arrowRoom : 0)) / arc
    const at = freePlace(boxAt, boxes, titles, ends, STEP / arc)

    // A title with nowhere clear still stands somewhere, and the next one along
    // keeps off it.
    titles.push(boxAt(at))

    // The words of a curve taken the other way round are read from its far end.
    return { ...edge, wordsAt: edge.heading === 'against' ? 1 - at : at }
  })
}

/**
 * The box a title fills where it is set at a fraction of the curve. The words
 * follow the line, so the run is sampled along it and each sample carries the
 * depth of a line of type across the line, which is where the letters stand.
 */
function titleBoxes(
  edge: PlacedEdge,
  arc: number,
  extent: number,
  height: number,
): (at: number) => Box {
  const along = rulerOf(edge)
  const half = extent / 2 / arc
  const deep = height / 2

  return (at) => {
    const run: Point[] = []
    for (let sample = 0; sample <= RUN_SAMPLES; sample += 1) {
      run.push(along(at - half + (2 * half * sample) / RUN_SAMPLES))
    }

    let minX = Infinity
    let minY = Infinity
    let maxX = -Infinity
    let maxY = -Infinity

    for (const [sample, point] of run.entries()) {
      const back = run[Math.max(sample - 1, 0)]!
      const on = run[Math.min(sample + 1, run.length - 1)]!
      const span = Math.hypot(on.x - back.x, on.y - back.y) || 1
      const acrossX = (Math.abs(on.y - back.y) / span) * deep
      const acrossY = (Math.abs(on.x - back.x) / span) * deep

      minX = Math.min(minX, point.x - acrossX)
      minY = Math.min(minY, point.y - acrossY)
      maxX = Math.max(maxX, point.x + acrossX)
      maxY = Math.max(maxY, point.y + acrossY)
    }

    return { minX, minY, maxX, maxY }
  }
}

/**
 * The fraction of the curve a title is set at: the middle, else the nearest
 * place to it on either side, stepping outwards for as long as the whole of
 * the words still stands on the line.
 *
 * A line offering nowhere clear of everything gives up the boxes before it
 * gives up its neighbours, since two titles set one over the other can be read
 * as neither. A line offering nowhere at all keeps its title at the middle.
 */
function freePlace(
  boxAt: (at: number) => Box,
  boxes: readonly Box[],
  titles: readonly Box[],
  ends: number,
  step: number,
): number {
  if (ends > MIDDLE || step <= 0) return MIDDLE

  const tries: number[] = []
  for (let away = 0; away <= MIDDLE - ends; away += step) {
    if (away === 0) tries.push(MIDDLE)
    else tries.push(MIDDLE + away, MIDDLE - away)
  }

  const clearOf = (standing: readonly Box[]) => (at: number) =>
    !standing.some((box) => meets(box, boxAt(at)))

  return (
    tries.find(clearOf([...boxes, ...titles])) ?? tries.find(clearOf(titles)) ?? MIDDLE
  )
}
