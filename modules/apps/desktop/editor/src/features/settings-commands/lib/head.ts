/**
 * The style elements the page is dressed in, and what is written into them.
 *
 * The page is served with three at the end of the head, each marked as the one
 * of the three it is: the mode, the theme's own file, and the two multipliers.
 * The theme's stands after the mode's and the sizes after both.
 */
import type { Sizes } from '@/entities/settings/theme'

/** The elements the page carries: the mode's, the theme's, and the sizes'. */
export interface StyleElements {
  readonly mode: HTMLStyleElement
  readonly theme: HTMLStyleElement
  /** Nothing for a page served at no size of its own, until one is written. */
  sizes: HTMLStyleElement | undefined
}

/**
 * The attribute each of the three carries, and what each of them says it is.
 * The application writes these where it dresses the page.
 */
export const MARKER = 'data-appearance'
export const IS_MODE = 'mode'
export const IS_THEME = 'theme'
export const IS_SIZES = 'sizes'

/** The element the page was served marked as one of the three, and nothing where it carries none. */
const marked = (sheet: Document, is: string): HTMLStyleElement | null =>
  sheet.head.querySelector<HTMLStyleElement>(`style[${MARKER}="${is}"]`)

/**
 * The elements the head ends with. A page served by something that dresses it
 * in nothing is given a mode's and a theme's of its own, in that order.
 */
export const dressing = (sheet: Document): StyleElements => {
  const mode = marked(sheet, IS_MODE) ?? sheet.head.appendChild(styling(IS_MODE, sheet))
  const theme = marked(sheet, IS_THEME) ?? after(mode, IS_THEME, sheet)
  return { mode, theme, sizes: marked(sheet, IS_SIZES) ?? undefined }
}

/** One of the three, marked as which of them it is. */
const styling = (is: string, sheet: Document): HTMLStyleElement => {
  const one = sheet.createElement('style')
  one.setAttribute(MARKER, is)
  return one
}

/** An element straight after another, which is where the next of the three goes. */
export const after = (
  before: HTMLStyleElement,
  is: string,
  sheet: Document,
): HTMLStyleElement => {
  const next = styling(is, sheet)
  before.after(next)
  return next
}

/** The two multipliers as the page carries them. */
export const declared = (sizes: Sizes): string =>
  `:root { --numen-interface-scale: ${sizes.interfaceScale}; --numen-text-scale: ${sizes.textScale}; }`
