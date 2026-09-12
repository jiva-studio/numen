/**
 * Page highlights and span targeting for a document.
 */
import { computed, shallowRef, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '../../../shared/core'
import type { Documents, PageHighlight, Rect } from '../types'

export function useDocumentHighlights(
  documents: Documents,
  path: string,
  pageNumber: Ref<number>,
  error: Ref<string>,
  onGoToPage: (page: number) => Promise<void>,
  isOpen: () => boolean,
) {
  const highlights = shallowRef<readonly PageHighlight[]>([])
  const others = shallowRef<readonly (readonly PageHighlight[])[]>([])

  const highlightedOn = (page: number): readonly Rect[] =>
    highlights.value.find((one) => one.page === page)?.rects ?? []

  const alsoOn = (page: number): readonly Rect[] =>
    others.value.flatMap((where) => where.find((one) => one.page === page)?.rects ?? [])

  const highlighted = computed<readonly Rect[]>(() => highlightedOn(pageNumber.value))
  const also = computed<readonly Rect[]>(() => alsoOn(pageNumber.value))

  const applyHighlights = async (where: readonly (readonly PageHighlight[])[]) => {
    const [front = [], ...rest] = where
    highlights.value = front
    others.value = rest
    const first = front[0] ?? rest.flat()[0]
    if (first) await onGoToPage(first.page)
  }

  const focusSpans = async (...spans: readonly Span[]) => {
    if (!isOpen() || spans.length === 0) return
    try {
      const where = await documents.getHighlights(path, spans)
      if (!isOpen()) return
      await applyHighlights(where)
    } catch (thrown) {
      if (!isOpen()) return
      error.value = formatErrorMessage(thrown)
    }
  }

  return {
    highlights,
    others,
    highlightedOn,
    alsoOn,
    highlighted,
    also,
    applyHighlights,
    focusSpans,
  }
}
