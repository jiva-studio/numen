/**
 * Where along its line each title sits, and how much of it stands there, in the
 * plex's own coordinates.
 *
 * A title stays on its own line, clear of every box and of every other title:
 * at the middle while that is clear, slid along the curve to the nearest clear
 * place while it is not, cut to the longest clear stretch where the whole of the
 * words stand nowhere, and dropped where that stretch holds less than half.
 */
import { headingOf, lengthOf, type PlacedEdge } from '../edge'
import type { PlacedNode } from '../node'
import { boxOf, isOverlapping, getTitleBoxes, runBoxes, type Box } from './boxes'
import { cutToFit, MIDDLE, type Routing } from './routing'

/** A run of one line, as fractions of its length. */
interface Span {
  readonly from: number
  readonly to: number
}

/** How long a line is, and how long the words it carries are. */
interface TitleMetrics {
  readonly arc: number
  readonly extent: number
}

/** How long a piece of line is looked at on its own, in plex units. */
const STEP = 3

/** The least of a title worth setting on a line, as a part of the whole. */
const LEAST = 0.5

/** How much of a line's depth a title keeps clear of whatever it stands near. */
const APART = 0.25

const getSpanLength = (span: Span): number => span.to - span.from

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

  for (const { edge, at, metrics } of tightestFirst(edges, width)) {
    const boxAt = runBoxes(edge, routing.labelDepth + 2 * apart)
    const ends = (edge.arrow ? routing.arrowRoom : 0) / metrics.arc
    const clear = clearSpans(boxAt, [...boxes, ...titles], ends, STEP / metrics.arc)

    const found = settle(clear, edge.words!, metrics, width)
    if (!found) {
      settled[at] = { ...edge, words: undefined, wordsAt: MIDDLE }
      continue
    }

    const half = found.extent / 2 / metrics.arc
    titles.push(...getTitleBoxes(boxAt, found.at - half, found.at + half, STEP / metrics.arc))

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
): { edge: PlacedEdge; at: number; metrics: TitleMetrics }[] {
  const measured: { edge: PlacedEdge; at: number; metrics: TitleMetrics }[] = []

  for (const [at, edge] of edges.entries()) {
    if (!edge.words) continue
    const arc = lengthOf(edge)
    const extent = width(edge.words)
    if (arc <= 0 || extent <= 0) continue
    measured.push({ edge, at, metrics: { arc, extent } })
  }

  return measured.sort(
    (one, other) =>
      one.metrics.arc - one.metrics.extent - (other.metrics.arc - other.metrics.extent) ||
      one.at - other.at,
  )
}

/**
 * The words that stand on a line and where: the whole of them at the clear
 * place nearest the middle, else as many as the longest clear span holds,
 * set in the middle of that span. Nothing where that span holds less
 * than `LEAST` of them.
 */
function settle(
  clear: readonly Span[],
  words: string,
  metrics: TitleMetrics,
  width: (label: string) => number,
): { words: string; at: number; extent: number } | null {
  const whole = nearestPlace(clear, metrics.extent / 2 / metrics.arc)
  if (whole !== null) return { words, at: whole, extent: metrics.extent }

  const longest = clear.reduce<Span | null>(
    (widest, span) => (!widest || getSpanLength(span) > getSpanLength(widest) ? span : widest),
    null,
  )
  if (!longest) return null

  const held = getSpanLength(longest) * metrics.arc
  if (held < LEAST * metrics.extent) return null

  const cut = cutToFit(words, held, width)
  return { words: cut, at: (longest.from + longest.to) / 2, extent: width(cut) }
}

/**
 * The runs of a line with nothing in the way, `ends` of it kept clear at
 * either end for whatever is drawn there. Every step of the line is looked at
 * on its own, so a line crossing a box comes back as the spans to either
 * side of it.
 */
function clearSpans(
  boxAt: (from: number, to: number) => Box,
  boxes: readonly Box[],
  ends: number,
  step: number,
): Span[] {
  const last = 1 - ends
  if (ends >= last || step <= 0) return []

  const spans: Span[] = []
  let open: number | null = null

  for (let at = ends; at < last; at += step) {
    const box = boxAt(at, Math.min(at + step, last))
    if (boxes.some((other) => isOverlapping(other, box))) {
      if (open !== null) spans.push({ from: open, to: at })
      open = null
    } else if (open === null) {
      open = at
    }
  }
  if (open !== null) spans.push({ from: open, to: last })

  return spans
}

/**
 * Where a title of this half-length stands: the place nearest the middle of
 * the line that leaves the whole of the words inside one clear span.
 * Nothing where no span is long enough to hold them.
 */
function nearestPlace(clear: readonly Span[], half: number): number | null {
  let nearest: number | null = null

  for (const span of clear) {
    if (getSpanLength(span) < 2 * half) continue
    const at = Math.min(Math.max(MIDDLE, span.from + half), span.to - half)
    if (nearest === null || Math.abs(at - MIDDLE) < Math.abs(nearest - MIDDLE)) {
      nearest = at
    }
  }

  return nearest
}
