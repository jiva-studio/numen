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
import type { PlacedNode } from './model'

/** How many parts are drawn under a node. */
export const MOST = 6

/** What stands in place of the ones that did not fit. */
export const REST = '…'

/**
 * How far behind the one above it each part sets off, as a fraction of the
 * opening.
 */
const LEAD = 0.12

/** How far in a part is ever set, counted in levels. */
const DEEPEST = 3

/** How far below its place a part sets off, as a fraction of its own height. */
const RISE = 0.7

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

/**
 * One part where it hangs. A part with no identifier stands for the ones that
 * did not fit, and cannot be chosen.
 */
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
  readonly parts: readonly DrawnPart[]
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
 * What one node hangs: the parts it holds, in the order they were given, and a
 * last one for those past the ceiling.
 *
 * Nothing for a node with no parts.
 */
export function hangParts(
  node: PlacedNode,
  parts: readonly PlexPart[],
  options: Pick<PlexOptions, 'partHeight' | 'partIndent'>,
  room: Room,
): HungParts | null {
  if (parts.length === 0) return null

  const { partHeight, partIndent } = options
  const drawn = parts.slice(0, MOST)
  const setIn = indents(drawn, partIndent)

  const hung: HungPart[] = drawn.map((part, at) => ({
    id: part.id,
    text: part.text,
    indent: setIn.get(part.level) ?? 0,
    at: at * partHeight,
  }))

  if (parts.length > drawn.length) {
    hung.push({ id: '', text: REST, indent: 0, at: hung.length * partHeight })
  }

  const pad = Math.round(PAD * partHeight)
  return {
    top: node.height / 2,
    partHeight,
    pad,
    ...across(node, hung, pad, room),
    parts: hung,
    height: hung.length * partHeight + 2 * pad,
  }
}

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
export function openedTo(hung: HungParts, open: number): OpenParts | null {
  const opened = clamp01(open)
  if (opened <= 0) return null

  // Each part sets off a lead behind the one above it, so what is left for any
  // one of them to run in is the opening less every lead before it.
  const runs = Math.max(1 - LEAD * (hung.parts.length - 1), LEAD)

  /** The deepest a part may set off from and still stand on the ground. */
  const floor = hung.height - 2 * hung.pad - hung.partHeight

  const parts = hung.parts.map((part, at) => {
    const own = easeOut(clamp01((opened - at * LEAD) / runs))
    const from = Math.min(part.at + RISE * hung.partHeight, floor)
    return { ...part, y: lerp(from, part.at, own), opacity: own }
  })

  return { height: hung.height, opacity: easeOut(opened), parts }
}
