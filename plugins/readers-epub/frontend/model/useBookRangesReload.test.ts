import { describe, expect, it } from 'vitest'
import { nextTick, reactive, ref } from 'vue'
import type { TextRange } from '../types'
import { useBookRanges } from './useBookRanges'

describe('the ranges of a book tab across a reload', () => {
  it('does not light the same array again when the book loads again', async () => {
    const focusCalls: (readonly TextRange[])[] = []
    const reader = {
      isLoading: ref(false),
      focusSpans: async (...spans: readonly TextRange[]) => {
        focusCalls.push(spans)
      },
    }
    const tab = reactive({ ranges: [{ start: 40, end: 50 }] as readonly TextRange[] | null })
    useBookRanges(reader, () => tab)
    await nextTick()

    reader.isLoading.value = true
    await nextTick()
    reader.isLoading.value = false
    await nextTick()

    expect(focusCalls).toHaveLength(1)
  })
})
