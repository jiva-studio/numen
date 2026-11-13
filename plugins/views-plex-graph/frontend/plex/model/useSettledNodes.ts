/** What a note of the frame is settled to: its widened box and its headings, worked out when asked for and kept until the frame or the sizes change. */
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
  const settled = computed(() => ({
    ...source(),
    wides: new Map<string, WideBox | null>(),
    hungs: new Map<string, NodeHeadings | null>(),
  }))

  return {
    getWide: (node: PlacedNode): WideBox | null => {
      const { wides, widen } = settled.value
      if (!wides.has(node.id)) wides.set(node.id, widen?.(node) ?? null)
      return wides.get(node.id) ?? null
    },
    getHung: (node: PlacedNode): NodeHeadings | null => {
      const { hungs, hung } = settled.value
      if (!hungs.has(node.id)) hungs.set(node.id, hung?.(node) ?? null)
      return hungs.get(node.id) ?? null
    },
  }
}
