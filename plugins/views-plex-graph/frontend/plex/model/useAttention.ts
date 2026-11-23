/** The node the pointer is over: it brings the links of that node forward. */
import { reactive, watch } from 'vue'
import type { PlexFrame } from '../lib/frame'

export interface Attention {
  node: string | null
}

/**
 * The attention over the drawing. A node that leaves the frame holds no attention.
 */
export function useAttention(frame: () => PlexFrame) {
  const attention = reactive<Attention>({ node: null })

  const setAttention = (id: string, isAttended: boolean) => {
    if (isAttended) attention.node = id
    else if (attention.node === id) attention.node = null
  }

  watch(frame, (next) => {
    const id = attention.node
    if (id !== null && !next.nodes.some((node) => node.id === id)) attention.node = null
  })

  return { attention, setAttention }
}
