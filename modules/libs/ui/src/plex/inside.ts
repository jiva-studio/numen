/**
 * The parts of a node, hung under its box.
 *
 * A node stands for something divided into named parts. While the attention
 * rests on the node they come out from under its box, each one setting off a
 * little behind the one above it, and choosing one says which it was.
 *
 * They hang from the box as it is drawn, so nothing here works out a width: a
 * part too long for the box is cut short in it, as a title is.
 */
import { clamp01, easeOut, lerp } from './arrange'
import type { PlexOptions, Size } from './arrange'
import type { PlacedNode, Point } from './model'

/**
 * How far behind the one above it each part sets off, as a fraction of the
 * opening.
 */
const LEAD = 0.12

/** How far in a part is ever set, counted in levels. */
const DEEPEST = 3

/** How far below its place a part sets off, as a fraction of its own height. */
const RISE = 0.7

/** How wide a mark at an edge of the ground is drawn, and how deep. */
const MARK = { wide: 4, deep: 2.5 }

/** The ground kept clear around the parts, as a fraction of a part's height. */
const PAD = 0.25

/**
 * One part of a node, as the caller describes it. The identifier is opaque:
 * the plex has no way to ask what it addresses.
 */
export interface PlexPart {
  readonly id: string
  readonly text: string
  /** How deep it sits. Only the order of the levels present is read. */
  readonly level: number
}

/** One part where it hangs. */
export interface HungPart {
  readonly id: string
  readonly text: string
  /** How far its words are set in from the edge. */
  readonly indent: number
  /** Where it rests, from the top of what is hung. */
  readonly at: number
}

/** What a node hangs, settled. */
export interface HungParts {
  /** The edge they come out from, from the middle of the node. */
  readonly top: number
  readonly partHeight: number
  /** Ground kept clear around them, so none stands flush against an edge. */
  readonly pad: number
  /** How many stand in the window at once. The rest are wound to. */
  readonly shown: number
  /** As wide as the longest of them asks for, held inside the window. */
  readonly width: number
  /** How far their middle stands from where the node is placed. */
  readonly offset: number
  readonly parts: readonly HungPart[]
  /** How deep they stand once they are all the way open. */
  readonly height: number
}

/** What settling how wide the parts are drawn needs to know. */
export interface Room {
  /**
   * The width one part's box needs for its words, padding included. Where
   * there is nothing to measure text with, the parts take the node's own box.
   */
  readonly measure?: ((text: string) => number) | undefined
  readonly viewport: Size
  /** Kept clear of the window edge, so nothing sits flush against it. */
  readonly margin: number
}

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
  /** The marks at either edge, one per direction there is more to wind to. */
  readonly marks: readonly Mark[]
}

/** A mark at an edge of the ground, saying which way there is more. */
export interface Mark {
  readonly at: 'above' | 'below'
  /** The three corners it is drawn through, from the middle of the node. */
  readonly points: readonly Point[]
}

/**
 * How far in each level present is set, by where it stands among the others: a
 * note whose parts begin at the second level is set in from the edge as one
 * beginning at the first.
 */
function indents(parts: readonly PlexPart[], step: number): Map<number, number> {
  const levels = [...new Set(parts.map((part) => part.level))].sort((a, b) => a - b)
  return new Map(levels.map((level, rank) => [level, Math.min(rank, DEEPEST) * step]))
}

/**
 * What one node hangs: every part it holds, in the order they were given, and
 * how many of them stand in the window at once.
 *
 * Nothing for a node with no parts.
 */
export function hangParts(
  node: PlacedNode,
  parts: readonly PlexPart[],
  options: Pick<PlexOptions, 'partHeight' | 'partIndent' | 'maxParts'>,
  room: Room,
): HungParts | null {
  if (parts.length === 0) return null

  const { partHeight, partIndent, maxParts } = options
  const pad = Math.round(PAD * partHeight)
  const top = node.height / 2

  // They hang below the node and stay inside the window, so how many of them
  // are drawn is how many the depth left under it holds. A node with room for
  // none hangs nothing.
  const depth = room.viewport.height / 2 - room.margin - (node.y + top)
  const rows = Math.min(maxParts + 1, Math.floor((depth - 2 * pad) / partHeight))
  if (rows < 1) return null

  const shown = Math.min(maxParts, rows, parts.length)
  const setIn = indents(parts, partIndent)

  const hung: HungPart[] = parts.map((part, at) => ({
    id: part.id,
    text: part.text,
    indent: setIn.get(part.level) ?? 0,
    at: at * partHeight,
  }))

  return {
    top,
    partHeight,
    pad,
    shown,
    ...across(node, hung, pad, room),
    parts: hung,
    height: shown * partHeight + 2 * pad,
  }
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
 * How wide the parts are drawn and where that width sits: the room the longest
 * of them asks for, never narrower than the node's own box and never wider
 * than the window. It grows about the node's middle and slides back inside the
 * window where the middle leaves it no room to grow.
 */
function across(
  node: PlacedNode,
  hung: readonly HungPart[],
  pad: number,
  room: Room,
): { width: number; offset: number } {
  const measure = room.measure
  const asked = measure
    ? Math.max(...hung.map((part) => measure(part.text) + part.indent))
    : 0
  const width = Math.min(
    Math.max(asked + 2 * pad, node.width),
    room.viewport.width - 2 * room.margin,
  )

  const furthest = room.viewport.width / 2 - room.margin - width / 2
  const middle = Math.min(Math.max(node.x, -furthest), furthest)
  return { width, offset: middle - node.x }
}

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
export function openedTo(hung: HungParts, open: number, wound = 0): OpenParts | null {
  const opened = clamp01(open)
  if (opened <= 0) return null

  // The window stands whole on the parts, wound by one at a time.
  const first = Math.min(Math.max(Math.round(wound), 0), furthest(hung))
  const standing = hung.parts.slice(first, first + hung.shown)

  // Each part sets off a lead behind the one above it, so what is left for any
  // one of them to run in is the opening less every lead before it.
  const runs = Math.max(1 - LEAD * (standing.length - 1), LEAD)

  /** The deepest a part may set off from and still stand on the ground. */
  const floor = hung.height - 2 * hung.pad - hung.partHeight

  const parts = standing.map((part, at) => {
    const own = easeOut(clamp01((opened - at * LEAD) / runs))
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
    marks: [
      ...(above ? [markAt(hung, hung.pad / 2, -1)] : []),
      ...(below ? [markAt(hung, hung.height - hung.pad / 2, 1)] : []),
    ],
  }
}

/**
 * A mark at one edge of the ground, pointing the way there is more to wind to.
 * It is drawn about the middle of what is hung, which is where the eye is.
 */
function markAt(hung: HungParts, down: number, facing: 1 | -1): Mark {
  const middle = hung.offset
  const y = hung.top + down
  return {
    at: facing > 0 ? 'below' : 'above',
    points: [
      { x: middle - MARK.wide, y: y - facing * MARK.deep },
      { x: middle, y: y + facing * MARK.deep },
      { x: middle + MARK.wide, y: y - facing * MARK.deep },
    ],
  }
}
