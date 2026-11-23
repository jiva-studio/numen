import { createArrowTransform, createPath, createReadingPath } from './arrange'
import { buildEdgeKey, type PlacedEdge } from './edge'
import type { EdgeArrow } from '../../types'

/** Rendered edge line with its SVG path, arrowheads, and titles. */
export interface EdgeLine {
  readonly edge: PlacedEdge
  readonly key: string
  readonly pair: string
  readonly d: string
  readonly arrows: readonly { readonly end: EdgeArrow; readonly transform: string }[]
  readonly titles: readonly TitleLine[]
}

/** Positioned title along an edge line. */
interface TitleLine {
  readonly words: string
  readonly path: string
  readonly line: string
  readonly at: string
  readonly across: number
}

const getName = (edge: PlacedEdge): string =>
  `${edge.from}->${edge.to}`.replace(
    /[^A-Za-z0-9_-]/g,
    (mark) => `_${mark.codePointAt(0)!.toString(16)}_`,
  )

/** Builds the rendered edge lines for the given placed edges. */
export const getLines = (edges: readonly PlacedEdge[], uid: string): readonly EdgeLine[] =>
  edges.map((edge) => ({
    edge,
    key: `${edge.from}->${edge.to}`,
    pair: buildEdgeKey(edge),
    d: createPath(edge),
    arrows: (edge.arrowheads ?? []).map((head) => ({
      end: head.end,
      transform: createArrowTransform(head),
    })),
    titles: edge.titles.map((title) => ({
      words: title.words,
      path: `${uid}-title-${getName(edge)}-${title.record}`,
      line: createReadingPath(edge, title.heading),
      at: `${100 * title.wordsAt}%`,
      across: (title.heading === 'against' ? -title.shift : title.shift) + title.rise,
    })),
  }))
