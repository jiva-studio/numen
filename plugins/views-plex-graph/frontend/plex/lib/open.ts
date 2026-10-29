/**
 * The window on what a node hangs: how far the headings have come out from under
 * its box, which of them stand in the window, and how far a wheel scrolls it.
 */
import { clamp01, easeOut, lerp } from './arrange'
import type { HungHeading, NodeHeadings } from './inside'
import type { Position } from './node'

/**
 * How far behind the one above it each heading sets off, as a fraction of the
 * opening.
 */
const LEAD = 0.12

/**
 * How much of the opening the leads take between them. However many headings
 * stand, the last of them sets off with this much of the opening left to run
 * in, and a wide window leads by less than a narrow one.
 */
const SPREAD = 0.6

/** How far below its place a heading sets off, as a fraction of its own height. */
const RISE = 0.7

/** Width and depth of an arrow at an edge of the panel. */
const ARROW = { wide: 4, deep: 2.5 }

/** Which way an arrow points down the page: towards what is below, away from what is above. */
const ARROW_DIRECTION = { above: -1, below: 1 } as const

/** One heading at a moment of the opening. */
export interface DrawnHeading extends HungHeading {
  /** at its current position now, from the top of what is hung. */
  readonly y: number
  readonly opacity: number
}

/** What a node hangs, partway open. */
export interface OpenHeadings {
  /** Height of the panel behind the headings. */
  readonly height: number
  /** Opacity of that panel. */
  readonly opacity: number
  /** The headings standing in the window, in the order they stand. */
  readonly headings: readonly DrawnHeading[]
  /** Which heading the window relies on, counted from the first. */
  readonly first: number
  /** Whether the window has headings above it, and headings below it. */
  readonly isAbove: boolean
  readonly isBelow: boolean
  /** The arrows at either edge, one per direction there is more to scroll to. */
  readonly arrows: readonly Arrow[]
}

/** An arrow at an edge of the panel, indicating the direction with more headings. */
export interface Arrow {
  readonly at: 'above' | 'below'
  /** The three corners it is drawn through, from the middle of the node. */
  readonly points: readonly Position[]
}

/**
 * How far a wheel scrolls the window, in whole headings, and the remainder.
 *
 * The caller carries the remainder into the next wheel event, so a trackpad
 * reporting a few pixels at a time accumulates to whole headings. Wheel deltas
 * may be in lines or pages as well as pixels, and only pixels can be compared
 * with a heading height.
 */
export function scrollBy(
  suspended: NodeHeadings,
  wheel: { readonly delta: number; readonly mode: number },
  leftover: number,
): { by: number; left: number } {
  const pixels = leftover + wheel.delta * getStride(suspended, wheel.mode)
  const lineCount = Math.trunc(pixels / suspended.headingHeight)
  return { by: lineCount, left: pixels - lineCount * suspended.headingHeight }
}

/** What one of a wheel's own units evaluates to in pixels: a line, or a window. */
function getStride(suspended: NodeHeadings, mode: number): number {
  if (mode === LINES) return suspended.headingHeight
  if (mode === WINDOWS) return suspended.visibleCount * suspended.headingHeight
  return 1
}

/** The wheel delta units, numbered as the browser's `deltaMode` numbers them. */
const LINES = 1
const WINDOWS = 2

/** The getFurthest the window on the headings may be scrolled down, counted in headings. */
export const getFurthest = (suspended: NodeHeadings): number =>
  Math.max(0, suspended.headings.length - suspended.visibleCount)

/**
 * What a node hangs at a moment of the opening.
 *
 * The panel opens to the depth it keeps, and each heading rises into place from
 * slightly below it, staggered by a lead behind the heading above. Headings
 * rise from inside the panel and never from outside it, so none is drawn where
 * the panel is absent.
 *
 * Returns null while the node is closed, since a closed node draws nothing.
 */
export function getOpenHeadings(
  suspended: NodeHeadings,
  open: number,
  scrollOffset = 0,
): OpenHeadings | null {
  const openProgress = clamp01(open)
  if (openProgress <= 0) return null

  // The window covers whole headings and scrolls one heading at a time.
  const first = Math.min(Math.max(Math.round(scrollOffset), 0), getFurthest(suspended))
  const visibleHeadings = suspended.headings.slice(first, first + suspended.visibleCount)

  // Each heading sets off a lead behind the one above it, and the leads together
  // take the same share of the opening whatever number of headings stand. What is
  // left for any one of them to run in is the opening less every lead before it.
  const many = visibleHeadings.length
  const lead = Math.min(LEAD, SPREAD / (many - 1))
  const runs = 1 - lead * (many - 1)

  /** The lowest start position of a heading that still lies within the panel. */
  const floor = suspended.height - 2 * suspended.pad - suspended.headingHeight

  const headings = visibleHeadings.map((heading, index) => {
    const own = easeOut(clamp01((openProgress - index * lead) / runs))
    const rests = index * suspended.headingHeight
    const from = Math.min(rests + RISE * suspended.headingHeight, floor)
    return { ...heading, at: rests, y: lerp(from, rests, own), opacity: own }
  })

  const isAbove = first > 0
  const isBelow = first < getFurthest(suspended)
  return {
    height: suspended.height,
    opacity: easeOut(openProgress),
    headings,
    first,
    isAbove,
    isBelow,
    arrows: [
      ...(isAbove ? [createArrow(suspended, suspended.pad / 2, 'above')] : []),
      ...(isBelow ? [createArrow(suspended, suspended.height - suspended.pad / 2, 'below')] : []),
    ],
  }
}

/**
 * An arrow at one edge of the panel, pointing in the direction with more to
 * scroll to. It is drawn near the middle of the node's headings.
 */
function createArrow(suspended: NodeHeadings, down: number, edge: Arrow['at']): Arrow {
  const direction = ARROW_DIRECTION[edge]
  const middle = suspended.offset
  const y = suspended.top + down
  return {
    at: edge,
    points: [
      { x: middle - ARROW.wide, y: y - direction * ARROW.deep },
      { x: middle, y: y + direction * ARROW.deep },
      { x: middle + ARROW.wide, y: y - direction * ARROW.deep },
    ],
  }
}
