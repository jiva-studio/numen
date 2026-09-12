/** An edge with what the drawing asks of it, worked out once for both layers. */
import { computed, ref, watch } from 'vue'
import { arrowTransformOf, pathOf, readingPathOf } from '../../lib/arrange'
import { edgeKey, type PlacedEdge } from '../../lib/edge'
import type { PlexFrame } from '../../lib/frame'

export interface EdgeLine {
  readonly edge: PlacedEdge
  /** One drawing per pair and direction, so a pair may carry two lines. */
  readonly key: string
  /** What the hand is on: two lines between one pair are one line to point at. */
  readonly pair: string
  readonly d: string
  /** The head on its end of the line, and nothing for a line carrying none. */
  readonly arrow: string | null
  /** The path the title is set along, named, and nothing for a line with no title. */
  readonly titlePath: string | null
  readonly titleLine: string
  /** How far along that path the middle of the title stands. */
  readonly titleAt: string
}

/**
 * Every edge with what the drawing asks of it. A title is always set along the
 * line it belongs to, in the words the arrangement cut for it and at the place
 * along it the arrangement chose.
 *
 * `uid` names the paths the titles are set along, so two plexes on one page
 * each name their own.
 */
export const linesOf = (edges: readonly PlacedEdge[], uid: string): readonly EdgeLine[] =>
  edges.map((edge, at) => ({
    edge,
    key: `${edge.from}->${edge.to}`,
    pair: edgeKey(edge),
    d: pathOf(edge),
    arrow: edge.arrowhead ? arrowTransformOf(edge.arrowhead) : null,
    titlePath: edge.words ? `${uid}-title-${at}` : null,
    titleLine: readingPathOf(edge),
    titleAt: `${100 * edge.wordsAt}%`,
  }))

/**
 * The lines of one frame, split by the one the hand is on: that one is drawn
 * over the boxes, and every other under them.
 *
 * `uid` names the paths the titles are set along.
 */
export function useEdgeLines(frame: () => PlexFrame, uid: string) {
  const lines = computed(() => linesOf(frame().edges, uid))

  /** The edge the hand is on, by a key that survives the re-routing of a move. */
  const over = ref<string | null>(null)

  // A new frame is a picture on its way somewhere, and the hand is on none of
  // it until it settles.
  watch(frame, () => {
    over.value = null
  })

  return {
    lines,
    lifted: computed(() => lines.value.filter((line) => line.pair === over.value)),
    resting: computed(() => lines.value.filter((line) => line.pair !== over.value)),
    setOver: (pair: string | null) => {
      over.value = pair
    },
  }
}
