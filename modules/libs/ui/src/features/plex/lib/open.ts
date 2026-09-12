/**
 * The window on what a node hangs: how far the parts have come out from under
 * its box, which of them stand in the window, and how far a wheel winds it.
 */
import { clamp01, easeOut, lerp } from './arrange'
import type { HungPart, HungParts } from './inside'
import type { Position } from './node'

/**
 * How far behind the one above it each part sets off, as a fraction of the
 * opening.
 */
const LEAD = 0.12

/**
 * How much of the opening the leads take between them. However many parts
 * stand, the last of them sets off with this much of the opening left to run
 * in, and a wide window leads by less than a narrow one.
 */
const SPREAD = 0.6

/** How far below its place a part sets off, as a fraction of its own height. */
const RISE = 0.7

/** How wide an arrow at an edge of the ground is drawn, and how deep. */
const ARROW = { wide: 4, deep: 2.5 }

/** One part at a moment of the opening. */
export interface DrawnPart extends HungPart {
  /** Where it stands now, from the top of what is hung. */
  readonly y: number
  readonly opacity: number
}

/** What a node hangs, partway open. */
export interface OpenParts {
  /** How deep the ground under them stands. */
  readonly height: number
  /** How far that ground has come up. */
  readonly opacity: number
  /** The parts standing in the window, in the order they stand. */
  readonly parts: readonly DrawnPart[]
  /** Which part the window stands on, counted from the first. */
  readonly first: number
  /** Whether the window has parts above it, and parts below it. */
  readonly above: boolean
  readonly below: boolean
  /** The arrows at either edge, one per direction there is more to wind to. */
  readonly arrows: readonly Arrow[]
}

/** An arrow at an edge of the ground, saying which way there is more. */
export interface Arrow {
  readonly at: 'above' | 'below'
  /** The three corners it is drawn through, from the middle of the node. */
  readonly points: readonly Position[]
}

/**
 * How far a wheel winds the window, in whole parts, and what is left over.
 *
 * A hand carries the leftover back into the next wheel, so a trackpad giving a
 * few pixels at a time winds as far as those pixels come to. A wheel says how
 * far it moved in lines or in windows as readily as in pixels, and only pixels
 * can be measured against a part.
 */
export function woundBy(
  hung: HungParts,
  wheel: { readonly delta: number; readonly mode: number },
  carried: number,
): { by: number; left: number } {
  const pixels = carried + wheel.delta * stride(hung, wheel.mode)
  const by = Math.trunc(pixels / hung.partHeight)
  return { by, left: pixels - by * hung.partHeight }
}

/** What one of a wheel's own units comes to in pixels: a line, or a window. */
function stride(hung: HungParts, mode: number): number {
  if (mode === LINES) return hung.partHeight
  if (mode === WINDOWS) return hung.shown * hung.partHeight
  return 1
}

/** The units a wheel says how far it moved in, as a browser numbers them. */
const LINES = 1
const WINDOWS = 2

/** The furthest the window on the parts may be wound down, counted in parts. */
export const furthest = (hung: HungParts): number =>
  Math.max(0, hung.parts.length - hung.shown)

/**
 * What a node hangs at a moment of the opening.
 *
 * The ground comes up at the depth it will keep, and each part rises onto its
 * place from a little below it, a lead behind the one above. A part rises from
 * inside the ground and never from beyond it, so none of them is ever drawn
 * where there is nothing under it.
 *
 * Nothing while it is shut, which is what a node draws nothing at all under.
 */
export function getOpenParts(hung: HungParts, open: number, wound = 0): OpenParts | null {
  const opened = clamp01(open)
  if (opened <= 0) return null

  // The window stands whole on the parts, wound by one at a time.
  const first = Math.min(Math.max(Math.round(wound), 0), furthest(hung))
  const shown = hung.parts.slice(first, first + hung.shown)

  // Each part sets off a lead behind the one above it, and the leads together
  // take the same share of the opening whatever number of parts stand. What is
  // left for any one of them to run in is the opening less every lead before it.
  const many = shown.length
  const lead = many > 1 ? Math.min(LEAD, SPREAD / (many - 1)) : 0
  const runs = 1 - lead * (many - 1)

  /** The deepest a part may set off from and still stand on the ground. */
  const floor = hung.height - 2 * hung.pad - hung.partHeight

  const parts = shown.map((part, at) => {
    const own = easeOut(clamp01((opened - at * lead) / runs))
    const rests = at * hung.partHeight
    const from = Math.min(rests + RISE * hung.partHeight, floor)
    return { ...part, at: rests, y: lerp(from, rests, own), opacity: own }
  })

  const above = first > 0
  const below = first < furthest(hung)
  return {
    height: hung.height,
    opacity: easeOut(opened),
    parts,
    first,
    above,
    below,
    arrows: [
      ...(above ? [arrowAt(hung, hung.pad / 2, -1)] : []),
      ...(below ? [arrowAt(hung, hung.height - hung.pad / 2, 1)] : []),
    ],
  }
}

/**
 * An arrow at one edge of the ground, pointing the way there is more to wind
 * to. It is drawn about the middle of what is hung, which is where the eye is.
 */
function arrowAt(hung: HungParts, down: number, facing: 1 | -1): Arrow {
  const middle = hung.offset
  const y = hung.top + down
  return {
    at: facing > 0 ? 'below' : 'above',
    points: [
      { x: middle - ARROW.wide, y: y - facing * ARROW.deep },
      { x: middle, y: y + facing * ARROW.deep },
      { x: middle + ARROW.wide, y: y - facing * ARROW.deep },
    ],
  }
}
