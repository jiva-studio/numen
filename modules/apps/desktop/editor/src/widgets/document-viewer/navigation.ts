/**
 * Navigation state and controls for a document.
 */
import { ref, type ShallowRef } from 'vue'
import type { Page } from './types'

export function useDocumentNavigation(pages: ShallowRef<readonly Page[]>) {
  const at = ref(0)

  const go = async (page: number) => {
    if (pages.value.length === 0) return
    at.value = Math.min(Math.max(Math.trunc(page), 0), pages.value.length - 1)
  }

  const next = () => go(at.value + 1)
  const back = () => go(at.value - 1)

  return {
    at,
    go,
    next,
    back,
  }
}
