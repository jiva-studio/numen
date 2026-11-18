/** The expanded box of a note of the frame, worked out when requested and kept until the frame or the sizes change. */
import { computed } from 'vue'
import type { PlacedNode } from '../lib/node'
import type { ExpandedBox } from './sizing/expansion'

export interface NodeSizesSource {
  readonly nodes: readonly PlacedNode[]
  readonly widen?: ((node: PlacedNode) => ExpandedBox | null) | undefined
}

export type NodeSizes = ReturnType<typeof useNodeSizes>

export function useNodeSizes(source: () => NodeSizesSource) {
  const cache = computed(() => ({
    ...source(),
    expandedBoxesById: new Map<string, ExpandedBox | null>(),
  }))

  return {
    getExpandedBoxOf: (node: PlacedNode): ExpandedBox | null => {
      const { expandedBoxesById, widen } = cache.value
      if (!expandedBoxesById.has(node.id)) expandedBoxesById.set(node.id, widen?.(node) ?? null)
      return expandedBoxesById.get(node.id) ?? null
    },
  }
}
