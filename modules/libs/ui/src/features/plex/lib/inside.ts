/**
 * The headings of a node, hung under its box. While the attention rests on the
 * node they come out from under it, each setting off a little behind the one
 * above, and choosing one says which it was.
 *
 * They hang from the box as it is drawn, so nothing here works out a width.
 */
import type { PlexOptions, Size } from './arrange'
import type { PlacedNode } from './node'

/** How far in a heading is ever set, counted in levels. */
const DEEPEST = 3

/** The ground kept clear around the headings, as a fraction of a heading's height. */
const PAD = 0.25

/**
 * One heading of a node, as the caller describes it. The identifier is opaque:
 * the plex has no way to ask what it addresses.
 */
export interface PlexHeading {
  readonly id: string
  readonly text: string
  /** How deep it sits. Only the order of the levels present is read. */
  readonly level: number
}

/** One heading where it hangs. */
export interface HungHeading {
  readonly id: string
  readonly text: string
  /** How far its words are set in from the edge. */
  readonly indent: number
  /** Where it rests, from the top of what is hung. */
  readonly at: number
}

/** What a node hangs, settled. */
export interface NodeHeadings {
  /** The edge they come out from, from the middle of the node. */
  readonly top: number
  readonly headingHeight: number
  /** Ground kept clear around them, so none stands flush against an edge. */
  readonly pad: number
  /** How many stand in the window at once. The rest are scrolled to. */
  readonly shown: number
  /** As wide as the longest of them asks for, held inside the window. */
  readonly width: number
  /** How far their middle stands from where the node is placed. */
  readonly offset: number
  readonly headings: readonly HungHeading[]
  /** How deep they stand once they are all the way open. */
  readonly height: number
}

/** What settling where and how wide the headings are drawn needs to know. */
export interface HeadingsDeps {
  /**
   * The width one heading's box needs for its words, padding included. Where
   * there is nothing to measure text with, the headings take the node's own box.
   */
  readonly measure?: ((text: string) => number) | undefined
  readonly viewport: Size
  /** Kept clear of the window edge, so nothing sits flush against it. */
  readonly margin: number
}

/**
 * How far in each level present is set, by at its current position among the others: a
 * note whose headings begin at the second level is set in from the edge as one
 * beginning at the first.
 */
function getIndents(headings: readonly PlexHeading[], step: number): Map<number, number> {
  const levels = [...new Set(headings.map((heading) => heading.level))].sort((a, b) => a - b)
  return new Map(levels.map((level, rank) => [level, Math.min(rank, DEEPEST) * step]))
}

export interface NodeHeadingsContext {
  readonly options: Pick<PlexOptions, 'headingHeight' | 'headingIndent' | 'maxHeadings'>
  readonly deps: HeadingsDeps
}

/**
 * What one node hangs: every heading it holds, in the order they were given, and
 * how many of them stand in the window at once.
 *
 * Nothing for a node with no headings.
 */
export function getNodeHeadings(
  node: PlacedNode,
  headings: readonly PlexHeading[],
  context: NodeHeadingsContext,
): NodeHeadings | null {
  if (headings.length === 0) return null

  const { options, deps } = context
  const { headingHeight, headingIndent, maxHeadings } = options
  const pad = Math.round(PAD * headingHeight)
  const top = node.height / 2

  // They hang below the node and stay inside the window, so how many of them
  // are drawn is how many the depth left under it holds. A node with room for
  // none hangs nothing.
  const depth = deps.viewport.height / 2 - deps.margin - (node.y + top)
  const rows = Math.min(maxHeadings + 1, Math.floor((depth - 2 * pad) / headingHeight))
  if (rows < 1) return null

  const headingCount = Math.min(maxHeadings, rows, headings.length)
  const setIn = getIndents(headings, headingIndent)

  const suspended: HungHeading[] = headings.map((heading, at) => ({
    id: heading.id,
    text: heading.text,
    indent: setIn.get(heading.level) ?? 0,
    at: at * headingHeight,
  }))

  return {
    top,
    headingHeight,
    pad,
    shown: headingCount,
    ...getHeadingsWidth(node, suspended, { pad, deps }),
    headings: suspended,
    height: headingCount * headingHeight + 2 * pad,
  }
}

interface HeadingsWidthContext {
  readonly pad: number
  readonly deps: HeadingsDeps
}

/**
 * How wide the headings are drawn and where that width sits: the room the longest
 * of them asks for, never narrower than the node's own box and never wider
 * than the window. It grows about the node's middle and slides back inside the
 * window where the middle leaves it no room to grow.
 */
function getHeadingsWidth(
  node: PlacedNode,
  suspended: readonly HungHeading[],
  context: HeadingsWidthContext,
): { width: number; offset: number } {
  const { pad, deps } = context
  const measure = deps.measure
  const measuredContentWidth = measure
    ? Math.max(...suspended.map((heading) => measure(heading.text) + heading.indent))
    : 0
  const width = Math.min(
    Math.max(measuredContentWidth + 2 * pad, node.width),
    deps.viewport.width - 2 * deps.margin,
  )

  const getFurthest = deps.viewport.width / 2 - deps.margin - width / 2
  const middle = Math.min(Math.max(node.x, -getFurthest), getFurthest)
  return { width, offset: middle - node.x }
}
