/** The node the pointer is over and the node whose names it is over: either brings the names of that node forward. */
import { reactive, watch } from 'vue'
import type { PlexFrame } from '../lib/frame'

export interface Attention {
  node: string | null
  names: string | null
}

export type AttentionSlot = keyof Attention

/**
 * The attention over the drawing. Only the names read it, so a move of the pointer redraws them and not the nodes.
 * A node that leaves the frame holds no attention, and neither does a node whose name was chosen.
 */
export function useAttention(frame: () => PlexFrame) {
  const attention = reactive<Attention>({ node: null, names: null })

  const attend = (slot: AttentionSlot, id: string, isAttended: boolean) => {
    if (isAttended) attention[slot] = id
    else if (attention[slot] === id) attention[slot] = null
  }

  watch(frame, (next) => {
    const ids = new Set(next.nodes.map((node) => node.id))
    for (const slot of ['node', 'names'] as const) {
      const id = attention[slot]
      if (id !== null && !ids.has(id)) attention[slot] = null
    }
  })

  return { attention, attend }
}
