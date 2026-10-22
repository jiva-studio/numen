/**
 * Viewport scaling and image URLs for a document.
 */
import { computed, ref, type Ref, type ShallowRef } from 'vue'
import type { Documents, Page } from '../types'

const WIDEST = 4096

export interface DocumentViewportOptions {
  pages: ShallowRef<readonly Page[]>
  pageNumber: Ref<number>
  fingerprint: Ref<string | undefined>
  isOpen: () => boolean
}

export function useDocumentViewport(
  documents: Documents,
  path: string,
  options: DocumentViewportOptions,
) {
  const { pages, pageNumber, fingerprint, isOpen } = options
  const width = ref(0)

  const expandWidth = (pixels: number) => {
    if (!isOpen()) return
    width.value = Math.min(Math.max(Math.round(pixels), 0), WIDEST)
  }

  const getPageImageUrl = (page: number): string =>
    pages.value.length > 0 && width.value > 0
      ? documents.getPageUrl(path, page, width.value, fingerprint.value!)
      : ''

  const picture = computed(() => getPageImageUrl(pageNumber.value))

  return {
    width,
    widen: expandWidth,
    getPageImageUrl,
    picture,
  }
}
