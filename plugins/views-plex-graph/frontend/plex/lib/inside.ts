/**
 * The headings of a node, laid out below its box. While the attention rests on
 * the node they expand from under it, each offset slightly from the one above,
 * and selecting one identifies it.
 *
 * They are anchored to the box as drawn, so nothing here computes a width.
 */
import type { PlexOptions, Size } from './arrange'
import type { PlacedNode } from './node'
import type { PlexHeading } from '../../types'

/** How far in a heading is ever set, counted in levels. */
const DEEPEST = 3

/** The clearance around the headings, as a fraction of a heading's height. */
const PAD = 0.25

/** One heading where it hangs. */
export interface HungHeading {
  readonly id: string
  readonly text: string
  readonly indent: number
  readonly at: number
}

/** What a node hangs, settled. */
export interface NodeHeadings {
  readonly top: number
  readonly headingHeight: number
  readonly pad: number
  readonly visibleCount: number
  readonly width: number
  readonly offset: number
  readonly headings: readonly HungHeading[]
  readonly height: number
}

/** What settling where and how wide the headings are drawn needs to know. */
export interface HeadingsDeps {
  readonly measure?: ((text: string) => number) | undefined
  readonly viewport: Size
  readonly margin: number
}

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

  // They hang below the node and stay inside the window, so the number drawn is
  // the number the remaining depth below the node can hold. A node with no
  // space for any draws none.
  const depth = deps.viewport.height / 2 - deps.margin - (node.y + top)
  const rows = Math.min(maxHeadings + 1, Math.floor((depth - 2 * pad) / headingHeight))
  if (rows < 1) return null

  const headingCount = Math.min(maxHeadings, rows, headings.length)
  const setIn = getIndents(headings, headingIndent)

  const suspended: HungHeading[] = headings.map((heading, index) => ({
    id: heading.id,
    text: heading.text,
    indent: setIn.get(heading.level) ?? 0,
    at: index * headingHeight,
  }))

  return {
    top,
    headingHeight,
    pad,
    visibleCount: headingCount,
    ...getHeadingsWidth(node, suspended, { pad, deps }),
    headings: suspended,
    height: headingCount * headingHeight + 2 * pad,
  }
}

interface HeadingsWidthContext {
  readonly pad: number
  readonly deps: HeadingsDeps
}

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
