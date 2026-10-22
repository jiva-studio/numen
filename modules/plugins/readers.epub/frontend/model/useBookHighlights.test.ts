/**
 * The runs of a book lit where they stand.
 */
import { describe, expect, it, vi } from 'vitest'
import { useBookHighlights } from './useBookHighlights'

const create = (open = true) => {
  const goToOffset = vi.fn<(offset: number) => Promise<void>>(async () => {})
  const lit = useBookHighlights(goToOffset, () => open)
  return { goToOffset, lit }
}

describe('a book before anything is lit', () => {
  it('holds no run at all', () => {
    const { lit } = create()

    expect(lit.highlights.value).toStrictEqual([])
    expect(lit.otherHighlights.value).toStrictEqual([])
  })
})

describe('a passage reached', () => {
  it('lights the first run, names the rest, and remains at the first', async () => {
    const { lit, goToOffset } = create()

    await lit.focusSpans({ from: 10, to: 20 }, { from: 50, to: 60 })

    expect(lit.highlights.value).toStrictEqual([{ from: 10, to: 20 }])
    expect(lit.otherHighlights.value).toStrictEqual([{ from: 50, to: 60 }])
    expect(goToOffset).toHaveBeenCalledWith(10)
  })

  it('leaves what is lit as it was when nothing was requested about', async () => {
    const { lit, goToOffset } = create()
    await lit.focusSpans({ from: 10, to: 20 })
    goToOffset.mockClear()

    await lit.focusSpans()

    expect(lit.highlights.value).toStrictEqual([{ from: 10, to: 20 }])
    expect(goToOffset).not.toHaveBeenCalled()
  })

  it('lights nothing and goes nowhere once the tab is closed', async () => {
    const { lit, goToOffset } = create(false)

    await lit.focusSpans({ from: 10, to: 20 })

    expect(lit.highlights.value).toStrictEqual([])
    expect(goToOffset).not.toHaveBeenCalled()
  })
})
