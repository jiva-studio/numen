import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { Ref } from 'vue'

export const HEADER_LEVELS = ['full', 'icons', 'menu'] as const

export type HeaderLevel = (typeof HEADER_LEVELS)[number]

/** The header is crowded when its content spills over it or a label shows an ellipsis. */
function isCrowded(element: HTMLElement): boolean {
  if (element.scrollWidth > element.clientWidth) return true
  return [...element.querySelectorAll<HTMLElement>('.card-action__label')].some(
    (label) => label.scrollWidth > label.clientWidth,
  )
}

/** The fullest level at which the header is not crowded, tried again at every size change of the header. */
export function useHeaderFit(header: Readonly<Ref<HTMLElement | null>>): Readonly<Ref<HeaderLevel>> {
  const level = ref<HeaderLevel>('full')
  let watcher: ResizeObserver | undefined

  async function fit(): Promise<void> {
    const element = header.value
    if (element === null) return
    for (const next of HEADER_LEVELS) {
      level.value = next
      await nextTick()
      if (!isCrowded(element)) return
    }
  }

  onMounted(() => {
    fit().catch(() => undefined)
    if (typeof ResizeObserver === 'undefined' || header.value === null) return
    watcher = new ResizeObserver(() => fit().catch(() => undefined))
    watcher.observe(header.value)
  })

  onBeforeUnmount(() => watcher?.disconnect())
  return level
}
