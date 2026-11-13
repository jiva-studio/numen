import { computed, ref, watch } from 'vue'
import type { PlexDragState } from '../../model/interaction/drag'
import type { Gesture } from '../../model/interaction/gesture'
import { createArrowTransform, createPath, createReadingPath } from '../../lib/arrange'
import { edgeKey, type PlacedEdge } from '../../lib/edge'
import type { PlexFrame } from '../../lib/frame'
import type { Attention } from '../../model/useAttention'
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

/** What the lines are drawn from: the frame, and whether the note the pointer is over highlights its lines. */
export interface EdgeLinesProps {
  readonly frame: PlexFrame
  readonly highlightLinks: boolean
  readonly gesture?: Gesture | undefined
  readonly dragging?: PlexDragState | undefined
}

/**
 * The edges of the frame as lines, and which of them are raised: the hovered one, and the ones
 * joining the attended note while the setting is on and nothing is dragged.
 *
 * The layers that draw the lines read `isRaised`, so a change of attention redraws those layers
 * and not the component that holds the lines.
 */
export function useEdgeLines(props: EdgeLinesProps, uid: string, attention: Attention) {
  const isIdle = computed(
    () =>
      (props.gesture?.from.value ?? null) === null && (props.dragging?.at.value ?? null) === null,
  )
  const getNodeToHighlight = () => (props.highlightLinks && isIdle.value ? attention.node : null)
  const frame = () => props.frame
  const lines = computed(() => getLines(frame().edges, uid))
  const overKey = ref<string | null>(null)

  watch(frame, () => {
    overKey.value = null
  })

  return {
    lines,
    isRaised: (line: EdgeLine) => {
      const attended = getNodeToHighlight()
      return (
        line.pair === overKey.value ||
        (attended !== null && (line.edge.from === attended || line.edge.to === attended))
      )
    },
    setOver: (pair: string | null) => {
      overKey.value = pair
    },
  }
}
