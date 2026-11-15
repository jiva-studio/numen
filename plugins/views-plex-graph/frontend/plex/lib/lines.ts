import { createArrowTransform, createPath, createReadingPath } from './arrange'
import { edgeKey, type PlacedEdge } from './edge'
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

/** Builds the rendered edge lines for the given placed edges. */
export const getLines = (edges: readonly PlacedEdge[], uid: string): readonly EdgeLine[] =>
  edges.map((edge, index) => ({
    edge,
    key: `${edge.from}->${edge.to}`,
    pair: edgeKey(edge),
    d: createPath(edge),
    arrows: (edge.arrowheads ?? []).map((head) => ({
      end: head.end,
      transform: createArrowTransform(head),
    })),
    titles: edge.titles.map((title, slot) => ({
      words: title.words,
      path: `${uid}-title-${index}-${slot}`,
      line: createReadingPath(edge, title.heading),
      at: `${100 * title.wordsAt}%`,
      across: (title.heading === 'against' ? -title.shift : title.shift) + title.rise,
    })),
  }))
