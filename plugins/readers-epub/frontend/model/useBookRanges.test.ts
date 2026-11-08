import { describe, expect, it, vi } from 'vitest'
import { nextTick, reactive, ref } from 'vue'
import type { TextRange } from '../types'
import { useBookRanges, type RangeTab } from './useBookRanges'

const createReader = (isLoading = false) => {
  const focusCalls: (readonly TextRange[])[] = []
  const reader = {
    isLoading: ref(isLoading),
    focusSpans: async (...spans: readonly TextRange[]) => {
      focusCalls.push(spans)
    },
    reportFailure: vi.fn(),
  }
  return { reader, focusCalls }
}

const createTab = (ranges: readonly TextRange[] | null = null) =>
  reactive({ ranges }) as { ranges: readonly TextRange[] | null } & RangeTab

const RANGES: readonly TextRange[] = [
  { start: 6_500, end: 6_520 },
  { start: 100, end: 110 },
]

describe('the ranges of a book tab', () => {
  it('highlights the ranges the tab opens with once the book is loaded', async () => {
    const { reader, focusCalls } = createReader(true)
    const tab = createTab(RANGES)
    useBookRanges(reader, () => tab)

    await nextTick()
    expect(focusCalls).toStrictEqual([])
    reader.isLoading.value = false
    await nextTick()

    expect(focusCalls).toStrictEqual([RANGES])
  })

  it('highlights the ranges set on the tab later, in their order', async () => {
    const { reader, focusCalls } = createReader()
    const tab = createTab()
    useBookRanges(reader, () => tab)

    tab.ranges = RANGES
    await nextTick()

    expect(focusCalls).toStrictEqual([RANGES])
  })

  it('highlights the same ranges again when the window sets them again', async () => {
    const { reader, focusCalls } = createReader()
    const tab = createTab()
    useBookRanges(reader, () => tab)

    tab.ranges = RANGES
    await nextTick()
    tab.ranges = [...RANGES]
    await nextTick()

    expect(focusCalls).toStrictEqual([RANGES, RANGES])
  })

  it('reports the failure of highlighting the ranges through the reader', async () => {
    const failure = new Error('the chapter at offset 6500 is not in the book')
    const reportFailure = vi.fn()
    const reader = {
      isLoading: ref(false),
      focusSpans: () => Promise.reject(failure),
      reportFailure,
    }
    useBookRanges(reader, () => createTab(RANGES))

    await nextTick()
    await Promise.resolve()

    expect(reportFailure).toHaveBeenCalledWith(failure)
  })
})
