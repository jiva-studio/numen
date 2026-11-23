import { computed, ref, watch } from 'vue'
import type { PlexDragState } from '../../model/interaction/drag'
import type { Gesture } from '../../model/interaction/gesture'
import { createEdgeLabel, buildEdgeKey } from '../../lib/edge'
import { getLines, type EdgeLine } from '../../lib/lines'
import type { PlexFrame } from '../../lib/frame'
import type { Attention } from '../../model/useAttention'

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
  const getFrame = () => props.frame
  const lines = computed(() => getLines(getFrame().edges, uid))
  const overKey = ref<string | null>(null)

  watch(getFrame, (next) => {
    if (next.edges.some((edge) => buildEdgeKey(edge) === overKey.value)) return
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
    getLabel: (line: EdgeLine): string =>
      createEdgeLabel(
        line.edge,
        (id) => getFrame().nodes.find((node) => node.id === id)?.title || 'Untitled',
      ),
  }
}

/** The lines of one layer of the drawing: the ones raised, and the ones at rest. */
export function useLayerLines(
  source: () => {
    readonly lines: readonly EdgeLine[]
    readonly isRaised: (line: EdgeLine) => boolean
  },
) {
  return {
    raised: computed(() => source().lines.filter(source().isRaised)),
    resting: computed(() => source().lines.filter((line) => !source().isRaised(line))),
  }
}
