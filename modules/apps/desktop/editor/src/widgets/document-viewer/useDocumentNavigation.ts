/**
 * Navigation state and controls for a document.
 */
import { ref, type ShallowRef } from 'vue'
import type { Page } from './types'

export function useDocumentNavigation(pages: ShallowRef<readonly Page[]>) {
  const pageNumber = ref(0)

  const goToPage = async (page: number) => {
    if (pages.value.length === 0) return
    pageNumber.value = Math.min(Math.max(Math.trunc(page), 0), pages.value.length - 1)
  }

  const nextPage = () => goToPage(pageNumber.value + 1)
  const prevPage = () => goToPage(pageNumber.value - 1)

  return {
    pageNumber,
    goToPage,
    nextPage,
    prevPage,
  }
}
