/**
 * The position of each title along its line, and how much of the title is shown
 * there, in plex coordinates.
 *
 * A title stays on its own line, clear of every box and of every other title:
 * at its own position when that position is clear (centered for a line with one
 * title, one near each end for a line with two), and moved along the curve
 * to the nearest clear position, truncated to the longest clear stretch when the whole
 * text fits nowhere, and dropped when that stretch is shorter than half.
 */
import { getLength, type EdgeTitle, type PlacedEdge } from '../../edge'
import type { PlacedNode } from '../../node'
import { getBox, isOverlapping, getTitleBoxes, runBoxes, type Box } from '../layout/boxes'
import { MIDDLE, createTitle, cutToFit, getDepth, type Routing, type TitleSetting } from './routing'

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

/** The fraction of a line's depth that a title keeps clear of adjacent shapes. */
const APART = 0.25

const getSpanLength = (span: Span): number => span.to - span.from

/**
 * Give every title a place on its line, clear of the boxes and of every title already settled, so a fan of lines out of one node reads as a list.
 * The tightest line goes first: a line barely longer than its words has one place to put them, and a roomy one takes what is left.
 * Where nothing measured the words there is no extent to keep clear of, and every title keeps its position.
 */
export function separateTitles(
  edges: readonly PlacedEdge[],
  nodes: readonly PlacedNode[],
  routing: Routing,
): PlacedEdge[] {
  const width = routing.labelWidth
  if (!width) return [...edges]

  const apart = APART * getDepth(routing.labelType)
  const boxes = nodes.map((node) => getBox(node, apart))
  const titleBoxes: Box[] = []
  const resolved: (EdgeTitle | null)[][] = edges.map((edge) => [...edge.titles])

  for (const { edge, index, title, slot, metrics } of sortTightestFirst(edges, width)) {
    const boxAt = runBoxes(edge, {
      depth: getDepth(routing.labelType) + 2 * apart,
      shift: title.shift,
    })
    const ends = (edge.arrows?.length ? routing.tuning.arrowClearance : 0) / metrics.arc
    const open = clearSpans(boxAt, [...boxes, ...titleBoxes], { ends, step: STEP / metrics.arc })
    const clear = edge.titles.length > 1 ? clipToHalf(open, title.anchor) : open

    const found = settle(clear, title.words, {
      metrics,
      width,
      target: title.anchor,
    })
    // A title of a pair keeps the reading of its record; the one title of a line reads the readable way along the line, wherever it is set.
    const setting: TitleSetting = {
      anchor: title.anchor,
      side: title.side,
      shift: title.shift,
      rise: title.rise,
      ...(edge.titles.length > 1 ? { heading: title.heading } : {}),
    }
    resolved[index]![slot] = found ? createTitle(edge, found.at, found.words, setting) : null
    if (!found) continue

    const half = found.extent / 2 / metrics.arc
    titleBoxes.push(
      ...getTitleBoxes(boxAt, { from: found.at - half, to: found.at + half }, STEP / metrics.arc),
    )
  }

  return edges.map((edge, index) => ({
    ...edge,
    titles: resolved[index]!.filter((title): title is EdgeTitle => title !== null),
  }))
}

/** One title to place: the edge it is on, where both sit in their lists, and its metrics. */
interface MeasuredTitle {
  readonly edge: PlacedEdge
  readonly index: number
  readonly title: EdgeTitle
  readonly slot: number
  readonly metrics: TitleMetrics
}

function sortTightestFirst(
  edges: readonly PlacedEdge[],
  width: (label: string) => number,
): MeasuredTitle[] {
  const measured: MeasuredTitle[] = []

  for (const [index, edge] of edges.entries()) {
    const arc = getLength(edge)
    for (const [slot, title] of edge.titles.entries()) {
      const extent = width(title.words)
      if (arc <= 0 || extent <= 0) continue
      measured.push({ edge, index, title, slot, metrics: { arc, extent } })
    }
  }

  return measured.sort(
    (one, other) =>
      one.metrics.arc - one.metrics.extent - (other.metrics.arc - other.metrics.extent) ||
      one.index - other.index ||
      one.slot - other.slot,
  )
}

/**
 * The words that stand on a line and where: the whole of them at the clear
 * place nearest the position it asks for, else as many as the longest
 * clear span holds, set in the middle of that span. Nothing where that span
 * holds less than `LEAST` of them.
 */
interface SettleOptions {
  readonly metrics: TitleMetrics
  readonly width: (label: string) => number
  readonly target: number
}

function settle(
  clear: readonly Span[],
  words: string,
  options: SettleOptions,
): { words: string; at: number; extent: number } | null {
  const { metrics, width, target } = options
  const whole = findNearestPlace(clear, metrics.extent / 2 / metrics.arc, target)
  if (whole !== null) return { words, at: whole, extent: metrics.extent }

  const longest = clear.reduce<Span | null>(
    (widest, span) => (!widest || getSpanLength(span) > getSpanLength(widest) ? span : widest),
    null,
  )
  if (!longest) return null

  const availableArcLength = getSpanLength(longest) * metrics.arc
  if (availableArcLength < LEAST * metrics.extent) return null

  const truncatedWords = cutToFit(words, availableArcLength, width)
  return {
    words: truncatedWords,
    at: (longest.from + longest.to) / 2,
    extent: width(truncatedWords),
  }
}

interface ClearSpanOptions {
  readonly ends: number
  readonly step: number
}

function clearSpans(
  boxAt: (from: number, to: number) => Box,
  boxes: readonly Box[],
  options: ClearSpanOptions,
): Span[] {
  const { ends, step } = options
  const last = 1 - ends
  if (ends >= last || step <= 0) return []

  const spans: Span[] = []
  let open: number | null = null

  for (let offset = ends; offset < last; offset += step) {
    const box = boxAt(offset, Math.min(offset + step, last))
    if (boxes.some((other) => isOverlapping(other, box))) {
      if (open !== null) spans.push({ from: open, to: offset })
      open = null
    } else if (open === null) {
      open = offset
    }
  }
  if (open !== null) spans.push({ from: open, to: last })

  return spans
}

function clipToHalf(spans: readonly Span[], anchor: number): Span[] {
  const [low, high] = anchor < MIDDLE ? [0, MIDDLE] : [MIDDLE, 1]
  return spans
    .map((span) => ({ from: Math.max(span.from, low), to: Math.min(span.to, high) }))
    .filter((span) => getSpanLength(span) > 0)
}

function findNearestPlace(clear: readonly Span[], half: number, target: number): number | null {
  let nearest: number | null = null

  for (const span of clear) {
    if (getSpanLength(span) < 2 * half) continue
    const pos = Math.min(Math.max(target, span.from + half), span.to - half)
    if (nearest === null || Math.abs(pos - target) < Math.abs(nearest - target)) {
      nearest = pos
    }
  }

  return nearest
}
