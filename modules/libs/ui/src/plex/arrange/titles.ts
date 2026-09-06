/**
 * Where along its line each title sits, and how much of it stands there, in the
 * plex's own coordinates.
 *
 * A title stays on its own line, clear of every box and of every other title:
 * at the middle while that is clear, slid along the curve to the nearest clear
 * place while it is not, cut to the longest clear stretch where the whole of the
 * words stand nowhere, and dropped where that stretch holds less than half.
 */
import { headingOf, lengthOf, rulerOf, type PlacedEdge } from '../edge'
import type { PlacedNode, Position } from '../node'
import { cutToFit, MIDDLE, type Routing } from './routing'

/** An upright box in the plex's own coordinates. */
interface Box {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** A run of one line, as fractions of its length. */
interface Stretch {
  readonly from: number
  readonly to: number
}

/** What a line offers the title it carries. */
interface Room {
  readonly arc: number
  readonly extent: number
}

/** How long a piece of line is looked at on its own, in plex units. */
const STEP = 3

/** The least of a title worth setting on a line, as a part of the whole. */
const LEAST = 0.5

/** How much of a line's depth a title keeps clear of whatever it stands near. */
const APART = 0.25

/** How many points along a run the box around it is drawn from. */
const RUN_SAMPLES = 8

const meets = (one: Box, other: Box): boolean =>
  one.minX < other.maxX &&
  other.minX < one.maxX &&
  one.minY < other.maxY &&
  other.minY < one.maxY

const boxOf = (node: PlacedNode, apart: number): Box => ({
  minX: node.x - node.width / 2 - apart,
  minY: node.y - node.height / 2 - apart,
  maxX: node.x + node.width / 2 + apart,
  maxY: node.y + node.height / 2 + apart,
})

const spanOf = (stretch: Stretch): number => stretch.to - stretch.from

/**
 * Give every title a place on its line. Each one is kept clear of the boxes
 * and of every title already settled, so a fan of lines out of one node reads
 * as a list.
 *
 * The tightest line goes first: a line barely longer than its words has one
 * place to put them, and a roomy one takes what is left.
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

  const apart = APART * routing.labelDepth
  const boxes = nodes.map((node) => boxOf(node, apart))
  const titles: Box[] = []
  const settled = [...edges]

  for (const { edge, at, room } of tightestFirst(edges, width)) {
    const boxAt = runBoxes(edge, routing.labelDepth + 2 * apart)
    const ends = (edge.arrow ? routing.arrowRoom : 0) / room.arc
    const clear = clearStretches(boxAt, [...boxes, ...titles], ends, STEP / room.arc)

    const found = settle(clear, edge.words!, room, width)
    if (!found) {
      settled[at] = { ...edge, words: undefined, wordsAt: MIDDLE }
      continue
    }

    const half = found.extent / 2 / room.arc
    titles.push(...ribbonOf(boxAt, found.at - half, found.at + half, STEP / room.arc))

    // The reading direction is the tangent where the words end up, and the
    // words of a curve taken the other way round are read from its far end.
    const heading = headingOf(edge, found.at)
    settled[at] = {
      ...edge,
      words: found.words,
      heading,
      wordsAt: heading === 'against' ? 1 - found.at : found.at,
    }
  }

  return settled
}

/**
 * The edges carrying measurable words, the least room first, each with the
 * place it holds in the picture. A line as long as its words comes before one
 * with room to spare, and edges alike in that keep the order they arrived in.
 */
function tightestFirst(
  edges: readonly PlacedEdge[],
  width: (label: string) => number,
): { edge: PlacedEdge; at: number; room: Room }[] {
  const measured: { edge: PlacedEdge; at: number; room: Room }[] = []

  for (const [at, edge] of edges.entries()) {
    if (!edge.words) continue
    const arc = lengthOf(edge)
    const extent = width(edge.words)
    if (arc <= 0 || extent <= 0) continue
    measured.push({ edge, at, room: { arc, extent } })
  }

  return measured.sort(
    (one, other) =>
      one.room.arc - one.room.extent - (other.room.arc - other.room.extent) ||
      one.at - other.at,
  )
}

/**
 * The words that stand on a line and where: the whole of them at the clear
 * place nearest the middle, else as many as the longest clear stretch holds,
 * set in the middle of that stretch. Nothing where that stretch holds less
 * than `LEAST` of them.
 */
function settle(
  clear: readonly Stretch[],
  words: string,
  room: Room,
  width: (label: string) => number,
): { words: string; at: number; extent: number } | null {
  const whole = nearestPlace(clear, room.extent / 2 / room.arc)
  if (whole !== null) return { words, at: whole, extent: room.extent }

  const longest = clear.reduce<Stretch | null>(
    (widest, stretch) => (!widest || spanOf(stretch) > spanOf(widest) ? stretch : widest),
    null,
  )
  if (!longest) return null

  const held = spanOf(longest) * room.arc
  if (held < LEAST * room.extent) return null

  const cut = cutToFit(words, held, width)
  return { words: cut, at: (longest.from + longest.to) / 2, extent: width(cut) }
}

/**
 * The room a title takes up, step by step along the run it is set on.
 *
 * A title set across the picture is a ribbon and not a rectangle: the box
 * around the whole of a diagonal run stands over most of a quarter of the
 * picture, and a line crossing anywhere near it would find nowhere to be.
 */
function ribbonOf(
  boxAt: (from: number, to: number) => Box,
  from: number,
  to: number,
  step: number,
): Box[] {
  const boxes: Box[] = []
  for (let at = from; at < to; at += step) {
    boxes.push(boxAt(at, Math.min(at + step, to)))
  }
  return boxes
}

/**
 * The runs of a line with nothing in the way, `ends` of it kept clear at
 * either end for whatever is drawn there. Every step of the line is looked at
 * on its own, so a line crossing a box comes back as the stretches to either
 * side of it.
 */
function clearStretches(
  boxAt: (from: number, to: number) => Box,
  placed: readonly Box[],
  ends: number,
  step: number,
): Stretch[] {
  const last = 1 - ends
  if (ends >= last || step <= 0) return []

  const stretches: Stretch[] = []
  let open: number | null = null

  for (let at = ends; at < last; at += step) {
    const box = boxAt(at, Math.min(at + step, last))
    if (placed.some((other) => meets(other, box))) {
      if (open !== null) stretches.push({ from: open, to: at })
      open = null
    } else if (open === null) {
      open = at
    }
  }
  if (open !== null) stretches.push({ from: open, to: last })

  return stretches
}

/**
 * Where a title of this half-length stands: the place nearest the middle of
 * the line that leaves the whole of the words inside one clear stretch.
 * Nothing where no stretch is long enough to hold them.
 */
function nearestPlace(clear: readonly Stretch[], half: number): number | null {
  let nearest: number | null = null

  for (const stretch of clear) {
    if (spanOf(stretch) < 2 * half) continue
    const at = Math.min(Math.max(MIDDLE, stretch.from + half), stretch.to - half)
    if (nearest === null || Math.abs(at - MIDDLE) < Math.abs(nearest - MIDDLE)) {
      nearest = at
    }
  }

  return nearest
}

/**
 * The box a run of the line fills. The words follow the line, so the run is
 * sampled along it and each sample carries the depth of a line of type across
 * the line, which is where the letters stand.
 */
function runBoxes(edge: PlacedEdge, depth: number): (from: number, to: number) => Box {
  const along = rulerOf(edge)
  const deep = depth / 2

  return (from, to) => {
    const run: Position[] = []
    for (let sample = 0; sample <= RUN_SAMPLES; sample += 1) {
      run.push(along(from + ((to - from) * sample) / RUN_SAMPLES))
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
