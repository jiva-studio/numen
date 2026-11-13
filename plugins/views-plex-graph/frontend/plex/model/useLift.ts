/** Whether the parts of a node are drawn in the layer over the others, which is while it is open, and the focus kept on the node when they move there. */
import { computed, nextTick, watch } from 'vue'

export interface LiftSite {
  readonly layer: () => Element | null
  readonly open: () => number
  readonly group: () => SVGElement | null
  readonly lifted: () => Element | null
}

export function useLift(site: LiftSite) {
  const isLifted = computed(() => site.layer() !== null && site.open() > 0)

  // The handle is made again where it moves to, and a focus on it goes with the old one.
  watch(isLifted, async () => {
    const group = site.group()
    const active = group?.ownerDocument.activeElement
    const hasFocus =
      !!active && active !== group && !!(group?.contains(active) || site.lifted()?.contains(active))
    if (!hasFocus) return
    await nextTick()
    group?.focus()
  })

  return { isLifted }
}
