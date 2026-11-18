/** The width of an element as it is drawn, border included, followed while it is there. Zero until it has been measured. */
import { onScopeDispose, ref, watch, type Ref } from 'vue'

export function useElementWidth(element: Readonly<Ref<HTMLElement | null>>): Readonly<Ref<number>> {
  const width = ref(0)
  if (typeof ResizeObserver === 'undefined') return width

  const observer = new ResizeObserver(([entry]) => {
    if (entry) width.value = entry.borderBoxSize[0]?.inlineSize ?? entry.contentRect.width
  })
  watch(
    element,
    (found) => {
      observer.disconnect()
      width.value = 0
      if (found) observer.observe(found)
    },
    { flush: 'post', immediate: true },
  )
  onScopeDispose(() => observer.disconnect())

  return width
}
