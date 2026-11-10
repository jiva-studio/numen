import { computed, ref, watch } from 'vue'
import { createArrowTransform, createPath, createReadingPath } from '../../lib/arrange'
import { edgeKey, type PlacedEdge } from '../../lib/edge'
import type { PlexFrame } from '../../lib/frame'
import type { EdgeArrow } from '../../../types'

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

/** Splits the frame edges into resting lines and hovered raised lines. */
export function useEdgeLines(frame: () => PlexFrame, uid: string) {
  const lines = computed(() => getLines(frame().edges, uid))
  const overKey = ref<string | null>(null)

  watch(frame, () => {
    overKey.value = null
  })

  return {
    lines,
    raisedLines: computed(() => lines.value.filter((line) => line.pair === overKey.value)),
    resting: computed(() => lines.value.filter((line) => line.pair !== overKey.value)),
    setOver: (pair: string | null) => {
      overKey.value = pair
    },
  }
}
