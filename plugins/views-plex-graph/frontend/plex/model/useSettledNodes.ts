/** What each note of the frame is settled to: its widened box and its headings, kept as they are until the frame or the sizes change. */
import { computed } from 'vue'
import type { NodeHeadings } from '../lib/inside'
import type { PlacedNode } from '../lib/node'
import type { WideBox } from './sizing/expansion'

export interface SettledNodesSource {
  readonly nodes: readonly PlacedNode[]
  readonly widen?: ((node: PlacedNode) => WideBox | null) | undefined
  readonly hung?: ((node: PlacedNode) => NodeHeadings | null) | undefined
}

export function useSettledNodes(source: () => SettledNodesSource) {
  return {
    wides: computed(
      () => new Map(source().nodes.map((node) => [node.id, source().widen?.(node) ?? null])),
    ),
    hungs: computed(
      () => new Map(source().nodes.map((node) => [node.id, source().hung?.(node) ?? null])),
    ),
  }
}
