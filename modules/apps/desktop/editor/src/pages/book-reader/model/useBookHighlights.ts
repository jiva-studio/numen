/**
 * The runs of a book lit where they stand: the one the reader was opened at,
 * and the others named with it.
 */
import { shallowRef } from 'vue'
import type { Span } from '@/shared/span'

export function useBookHighlights(
  goToOffset: (offset: number) => Promise<void>,
  isOpen: () => boolean,
) {
  const highlights = shallowRef<readonly Span[]>([])
  const otherHighlights = shallowRef<readonly Span[]>([])

  const focusSpans = async (...spans: readonly Span[]) => {
    if (!isOpen() || spans.length === 0) return
    const [front, ...rest] = spans
    if (!front) return
    highlights.value = [front]
    otherHighlights.value = rest
    await goToOffset(front.from)
  }

  return { highlights, otherHighlights, focusSpans }
}
