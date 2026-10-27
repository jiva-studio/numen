/**
 * The runs of a book lit where they stand: the one the reader was opened at,
 * and the others named with it.
 */
import { shallowRef } from 'vue'
import type { TextRange } from '@numen/plugin-sdk'

export function useBookHighlights(
  goToOffset: (offset: number) => Promise<void>,
  isOpen: () => boolean,
) {
  const highlights = shallowRef<readonly TextRange[]>([])
  const otherHighlights = shallowRef<readonly TextRange[]>([])

  const focusSpans = async (...spans: readonly TextRange[]) => {
    if (!isOpen()) return
    const [front, ...rest] = spans
    if (!front) return
    highlights.value = [front]
    otherHighlights.value = rest
    await goToOffset(front.start)
  }

  return { highlights, otherHighlights, focusSpans }
}
