/**
 * The style elements the page is styled with, and what is written into them.
 *
 * The page is served with three at the end of the head, each marked with which of the
 * three it is: the mode, the theme's own file, and the two multipliers. The theme's
 * follows the mode's and the sizes follow both.
 */
import { doc } from '@numen/ui'
import { SCHEMES, type Mode, type Sizes } from '../lib/appearance'

/** The elements the page carries: the mode's, the theme's, and the sizes'. */
export interface StyleElements {
  readonly mode: HTMLStyleElement
  readonly theme: HTMLStyleElement
  sizes: HTMLStyleElement | undefined
}

/**
 * The attribute each of the three carries, and which element it identifies.
 * The application writes these when it styles the page.
 */
const MARKER = 'data-appearance'
const IS_MODE = 'mode'
const IS_THEME = 'theme'
export const IS_SIZES = 'sizes'

const findStyle = (sheet: Document, is: string): HTMLStyleElement | null =>
  sheet.head.querySelector<HTMLStyleElement>(`style[${MARKER}="${is}"]`)

/**
 * The elements the head ends with. A page served by something that styles it
 * in nothing is given a mode's and a theme's of its own, in that order.
 */
export const getStyleElements = (sheet: Document): StyleElements => {
  const mode = findStyle(sheet, IS_MODE) ?? sheet.head.appendChild(createStyle(IS_MODE, sheet))
  const theme = findStyle(sheet, IS_THEME) ?? addAfter(mode, IS_THEME, sheet)
  return { mode, theme, sizes: findStyle(sheet, IS_SIZES) ?? undefined }
}

const createStyle = (is: string, sheet: Document): HTMLStyleElement => {
  const one = sheet.createElement('style')
  one.setAttribute(MARKER, is)
  return one
}

/** An element straight after another, which is where the next of the three goes. */
export const addAfter = (
  before: HTMLStyleElement,
  is: string,
  sheet: Document,
): HTMLStyleElement => {
  const next = createStyle(is, sheet)
  before.after(next)
  return next
}

/** The two multipliers as the page carries them. */
export const getSizesCss = (sizes: Sizes): string =>
  `:root { --numen-interface-scale: ${sizes.interfaceScale}; --numen-text-scale: ${sizes.textScale}; }`

/** Applies theme colors, mode and sizes to the window's style elements. */
export function createAppearanceOutput(sheet: Document = doc) {
  const elements = getStyleElements(sheet)
  let appliedSizesCss = ''
  return {
    servedTheme: elements.theme.textContent,
    writeMode: (mode: Mode) => {
      elements.mode.textContent = `:root { color-scheme: ${SCHEMES[mode]}; }`
    },
    writeTheme: (css: string) => {
      elements.theme.textContent = css
    },
    writeSizes: (sizes: Sizes) => {
      const css = getSizesCss(sizes)
      if (css === appliedSizesCss) return
      appliedSizesCss = css
      elements.sizes ??= addAfter(elements.theme, IS_SIZES, sheet)
      elements.sizes.textContent = css
    },
  }
}
