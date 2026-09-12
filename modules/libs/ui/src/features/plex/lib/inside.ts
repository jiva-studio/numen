/**
 * The parts of a node, hung under its box. While the attention rests on the
 * node they come out from under it, each setting off a little behind the one
 * above, and choosing one says which it was.
 *
 * They hang from the box as it is drawn, so nothing here works out a width.
 */
import type { PlexOptions, Size } from './arrange'
import type { PlacedNode } from './node'

/** How far in a part is ever set, counted in levels. */
const DEEPEST = 3

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

/** What settling where and how wide the parts are drawn needs to know. */
export interface PartsDeps {
  /**
   * The width one part's box needs for its words, padding included. Where
   * there is nothing to measure text with, the parts take the node's own box.
   */
  readonly measure?: ((text: string) => number) | undefined
  readonly viewport: Size
  /** Kept clear of the window edge, so nothing sits flush against it. */
  readonly margin: number
}

/**
 * How far in each level present is set, by where it stands among the others: a
 * note whose parts begin at the second level is set in from the edge as one
 * beginning at the first.
 */
function getIndents(parts: readonly PlexPart[], step: number): Map<number, number> {
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
  deps: PartsDeps,
): HungParts | null {
  if (parts.length === 0) return null

  const { partHeight, partIndent, maxParts } = options
  const pad = Math.round(PAD * partHeight)
  const top = node.height / 2

  // They hang below the node and stay inside the window, so how many of them
  // are drawn is how many the depth left under it holds. A node with room for
  // none hangs nothing.
  const depth = deps.viewport.height / 2 - deps.margin - (node.y + top)
  const rows = Math.min(maxParts + 1, Math.floor((depth - 2 * pad) / partHeight))
  if (rows < 1) return null

  const shown = Math.min(maxParts, rows, parts.length)
  const setIn = getIndents(parts, partIndent)

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
    ...across(node, hung, pad, deps),
    parts: hung,
    height: shown * partHeight + 2 * pad,
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
  deps: PartsDeps,
): { width: number; offset: number } {
  const measure = deps.measure
  const asked = measure
    ? Math.max(...hung.map((part) => measure(part.text) + part.indent))
    : 0
  const width = Math.min(
    Math.max(asked + 2 * pad, node.width),
    deps.viewport.width - 2 * deps.margin,
  )

  const furthest = deps.viewport.width / 2 - deps.margin - width / 2
  const middle = Math.min(Math.max(node.x, -furthest), furthest)
  return { width, offset: middle - node.x }
}
