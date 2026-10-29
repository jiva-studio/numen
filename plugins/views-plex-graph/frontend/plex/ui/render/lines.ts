/** An edge with the properties the drawing needs, computed once for both layers. */
import { computed, ref, watch } from 'vue'
import { createArrowTransform, createPath, createReadingPath } from '../../lib/arrange'
import { edgeKey, type PlacedEdge } from '../../lib/edge'
import type { PlexFrame } from '../../lib/frame'

export interface EdgeLine {
  readonly edge: PlacedEdge
  /** One drawing per pair and direction, so a pair may carry two lines. */
  readonly key: string
  /** The edge under the pointer: two lines between one pair are one line to target. */
  readonly pair: string
  readonly d: string
  /** The head on its end of the line, and nothing for a line carrying none. */
  readonly arrow: string | null
  /** The path the title is set along, named, and nothing for a line with no title. */
  readonly titlePath: string | null
  readonly titleLine: string
  /** How far along that path the middle of the title lies. */
  readonly titleAt: string
}

/**
 * Every edge with the properties the drawing needs. A title is always set along
 * the line it belongs to, in the words the arrangement split for it and at the
 * position along the line the arrangement chose.
 *
 * `uid` names the paths the titles are set along, so two plexes on one page
 * each name their own.
 */
export const getLines = (edges: readonly PlacedEdge[], uid: string): readonly EdgeLine[] =>
  edges.map((edge, index) => ({
    edge,
    key: `${edge.from}->${edge.to}`,
    pair: edgeKey(edge),
    d: createPath(edge),
    arrow: edge.arrowhead ? createArrowTransform(edge.arrowhead) : null,
    titlePath: edge.words ? `${uid}-title-${index}` : null,
    titleLine: createReadingPath(edge),
    titleAt: `${100 * edge.wordsAt}%`,
  }))

/**
 * The lines of one frame, split by the one under the pointer: that one is drawn
 * over the boxes, and every other under them.
 *
 * `uid` names the paths the titles are set along.
 */
export function useEdgeLines(frame: () => PlexFrame, uid: string) {
  const lines = computed(() => getLines(frame().edges, uid))

  /** The edge under the pointer, by a key that survives the re-routing of a move. */
  const overKey = ref<string | null>(null)

  // A new frame is still in transition, and no edge is under the pointer until it
  // settles.
  watch(frame, () => {
    overKey.value = null
  })

  return {
    lines,
    lifted: computed(() => lines.value.filter((line) => line.pair === overKey.value)),
    resting: computed(() => lines.value.filter((line) => line.pair !== overKey.value)),
    setOver: (pair: string | null) => {
      overKey.value = pair
    },
  }
}
