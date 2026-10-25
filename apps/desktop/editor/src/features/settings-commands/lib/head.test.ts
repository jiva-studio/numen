import { describe, expect, it } from 'vitest'
import { IS_MODE, IS_SIZES, IS_THEME, MARKER, addAfter, getStyleElements } from './head'

const createSheet = () => document.implementation.createHTMLDocument('numen')

describe('the elements a page is dressed in', () => {
  it('are marked as the one each is, on a page served in nothing', () => {
    const sheet = createSheet()

    const { mode, theme } = getStyleElements(sheet)

    expect(mode.getAttribute(MARKER)).toBe(IS_MODE)
    expect(theme.getAttribute(MARKER)).toBe(IS_THEME)
  })

  it('are found again by their mark, and none is made twice', () => {
    const sheet = createSheet()
    const first = getStyleElements(sheet)

    const again = getStyleElements(sheet)

    expect(again.mode).toBe(first.mode)
    expect(again.theme).toBe(first.theme)
    expect(sheet.head.querySelectorAll('style')).toHaveLength(2)
  })

  it('stand mode first, then theme, and the sizes follow the theme once added', () => {
    const sheet = createSheet()
    const { theme } = getStyleElements(sheet)

    const sizes = addAfter(theme, IS_SIZES, sheet)

    expect(sizes.getAttribute(MARKER)).toBe(IS_SIZES)
    expect(
      [...sheet.head.querySelectorAll('style')].map((one) => one.getAttribute(MARKER)),
    ).toEqual([IS_MODE, IS_THEME, IS_SIZES])
    expect(getStyleElements(sheet).sizes).toBe(sizes)
  })
})
