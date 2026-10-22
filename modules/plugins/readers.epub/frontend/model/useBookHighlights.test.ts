/**
 * The runs of a book lit where they stand.
 */
import { describe, expect, it, vi } from 'vitest'
import { useBookHighlights } from './useBookHighlights'

const create = (open = true) => {
  const goToOffset = vi.fn<(offset: number) => Promise<void>>(async () => {})
  const highlights = useBookHighlights(goToOffset, () => open)
  return { goToOffset, highlights }
}

describe('a book before anything is lit', () => {
  it('holds no run at all', () => {
    const { highlights } = create()

    expect(highlights.highlights.value).toStrictEqual([])
    expect(highlights.otherHighlights.value).toStrictEqual([])
  })
})

describe('a passage reached', () => {
  it('lights the first run, names the rest, and remains at the first', async () => {
    const { highlights, goToOffset } = create()

    await highlights.focusSpans({ from: 10, to: 20 }, { from: 50, to: 60 })

    expect(highlights.highlights.value).toStrictEqual([{ from: 10, to: 20 }])
    expect(highlights.otherHighlights.value).toStrictEqual([{ from: 50, to: 60 }])
    expect(goToOffset).toHaveBeenCalledWith(10)
  })

  it('leaves what is lit as it was when nothing was requested about', async () => {
    const { highlights, goToOffset } = create()
    await highlights.focusSpans({ from: 10, to: 20 })
    goToOffset.mockClear()

    await highlights.focusSpans()

    expect(highlights.highlights.value).toStrictEqual([{ from: 10, to: 20 }])
    expect(goToOffset).not.toHaveBeenCalled()
  })

  it('lights nothing and goes nowhere once the tab is closed', async () => {
    const { highlights, goToOffset } = create(false)

    await highlights.focusSpans({ from: 10, to: 20 })

    expect(highlights.highlights.value).toStrictEqual([])
    expect(goToOffset).not.toHaveBeenCalled()
  })
})
