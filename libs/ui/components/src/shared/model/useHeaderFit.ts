import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { Ref } from 'vue'
import { browserViewport, type Viewport } from '../api/viewport'

const HEADER_LEVELS = ['full', 'icons', 'menu'] as const

type HeaderLevel = (typeof HEADER_LEVELS)[number]

function isCrowded(element: HTMLElement): boolean {
  if (element.scrollWidth > element.clientWidth) return true
  return [...element.querySelectorAll<HTMLElement>('.card-action__label')].some(
    (label) => label.scrollWidth > label.clientWidth,
  )
}

/**
 * The fullest level at which the header is not crowded. It is found again only when the width of the header changes:
 * a change of the level changes the height and the content, and a fit that answered to those would run without end.
 * Fits never overlap; a width that comes while one runs is fitted when it ends.
 */
export function useHeaderFit(
  header: Readonly<Ref<HTMLElement | null>>,
  viewport: Pick<Viewport, 'watch'> = browserViewport,
): Readonly<Ref<HeaderLevel>> {
  const level = ref<HeaderLevel>('full')
  let unwatch: (() => void) | undefined
  let fitted = 0
  const flags = { isFitting: false, isStale: false, isGone: false }

  async function fit(): Promise<void> {
    const element = header.value
    if (element === null || flags.isFitting) {
      flags.isStale = true
      return
    }
    flags.isFitting = true
    try {
      do {
        flags.isStale = false
        await fitLevel(element)
      } while (isStale())
    } finally {
      flags.isFitting = false
    }
  }

  function isStale(): boolean {
    return flags.isStale && !flags.isGone
  }

  async function fitLevel(element: HTMLElement): Promise<void> {
    for (const next of HEADER_LEVELS) {
      if (flags.isGone) return
      level.value = next
      await nextTick()
      if (!isCrowded(element)) return
    }
  }

  onMounted(() => {
    fit().catch(() => undefined)
    if (header.value === null) return
    unwatch = viewport.watch(header.value, (size) => {
      if (size.width === fitted) return
      fitted = size.width
      fit().catch(() => undefined)
    })
  })

  onBeforeUnmount(() => {
    flags.isGone = true
    unwatch?.()
  })
  return level
}
