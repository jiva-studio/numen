/**
 * Viewport scaling and image URLs for a document.
 */
import { computed, ref, type Ref, type ShallowRef } from 'vue'
import type { Documents, Page } from '../types'

const WIDEST = 4096

export function useDocumentViewport(
  documents: Documents,
  path: string,
  pages: ShallowRef<readonly Page[]>,
  pageNumber: Ref<number>,
  fingerprint: Ref<string>,
  isOpen: () => boolean,
) {
  const wide = ref(0)

  const widen = (pixels: number) => {
    if (!isOpen()) return
    wide.value = Math.min(Math.max(Math.round(pixels), 0), WIDEST)
  }

  const getPageImageUrl = (page: number): string =>
    pages.value.length > 0 && wide.value > 0
      ? documents.getPageUrl(path, page, wide.value, fingerprint.value)
      : ''

  const picture = computed(() => getPageImageUrl(pageNumber.value))

  return {
    wide,
    widen,
    getPageImageUrl,
    picture,
  }
}
