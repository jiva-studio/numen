import { describe, expect, it } from 'vitest'
import { nextTick, reactive, ref } from 'vue'
import type { TextRange } from '../types'
import { useBookRanges, type RangedTab } from './useBookRanges'

/** A reader that records every call that lights ranges. */
const createReader = (isLoading = false) => {
  const focusCalls: (readonly TextRange[])[] = []
  const reader = {
    isLoading: ref(isLoading),
    focusSpans: async (...spans: readonly TextRange[]) => {
      focusCalls.push(spans)
    },
  }
  return { reader, focusCalls }
}

const createTab = (ranges: readonly TextRange[] | null = null) =>
  reactive({ ranges }) as { ranges: readonly TextRange[] | null } & RangedTab

const RANGES: readonly TextRange[] = [
  { start: 6_500, end: 6_520 },
  { start: 100, end: 110 },
]

describe('the ranges of a book tab', () => {
  it('lights the ranges the tab opens with once the book is loaded', async () => {
    const { reader, focusCalls } = createReader(true)
    const tab = createTab(RANGES)
    useBookRanges(reader, () => tab)

    await nextTick()
    expect(focusCalls).toStrictEqual([])
    reader.isLoading.value = false
    await nextTick()

    expect(focusCalls).toStrictEqual([RANGES])
  })

  it('lights the ranges set on the tab later, in their order', async () => {
    const { reader, focusCalls } = createReader()
    const tab = createTab()
    useBookRanges(reader, () => tab)

    tab.ranges = RANGES
    await nextTick()

    expect(focusCalls).toStrictEqual([RANGES])
  })

  it('lights the same ranges again when the window sets them anew', async () => {
    const { reader, focusCalls } = createReader()
    const tab = createTab()
    useBookRanges(reader, () => tab)

    tab.ranges = RANGES
    await nextTick()
    tab.ranges = [...RANGES]
    await nextTick()

    expect(focusCalls).toStrictEqual([RANGES, RANGES])
  })
})
