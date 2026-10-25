/**
 * Page highlights and span targeting for a document.
 */
import { computed, shallowRef, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '../types'
import type { Documents, PageHighlight, Rect } from '../types'

export interface DocumentHighlightsOptions {
  pageNumber: Ref<number>
  error: Ref<string>
  onGoToPage: (page: number) => Promise<void>
  isOpen: () => boolean
}

export function useDocumentHighlights(
  documents: Documents,
  path: string,
  options: DocumentHighlightsOptions,
) {
  const { pageNumber, error, onGoToPage, isOpen } = options
  const highlights = shallowRef<readonly PageHighlight[]>([])
  const otherHighlights = shallowRef<readonly (readonly PageHighlight[])[]>([])

  const getHighlightsOn = (page: number): readonly Rect[] =>
    highlights.value.find((one) => one.page === page)?.rects ?? []

  const getOtherHighlightsOn = (page: number): readonly Rect[] =>
    otherHighlights.value.flatMap((where) => where.find((one) => one.page === page)?.rects ?? [])

  const highlighted = computed<readonly Rect[]>(() => getHighlightsOn(pageNumber.value))
  const otherHighlighted = computed<readonly Rect[]>(() => getOtherHighlightsOn(pageNumber.value))

  const applyHighlights = async (where: readonly (readonly PageHighlight[])[]) => {
    const [front = [], ...rest] = where
    highlights.value = front
    otherHighlights.value = rest
    const first = front[0] ?? rest.flat()[0]
    if (first) await onGoToPage(first.page)
  }

  const focusSpans = async (...spans: readonly Span[]) => {
    if (!isOpen() || spans.length === 0) return
    try {
      const where = await documents.getHighlights(path, spans)
      if (!isOpen()) return
      await applyHighlights(where)
    } catch (caughtError) {
      if (!isOpen()) return
      error.value = formatErrorMessage(caughtError)
    }
  }

  return {
    highlights,
    otherHighlights,
    getHighlightsOn,
    getOtherHighlightsOn,
    highlighted,
    otherHighlighted,
    applyHighlights,
    focusSpans,
  }
}
