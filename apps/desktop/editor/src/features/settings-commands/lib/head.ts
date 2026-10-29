/**
 * The style elements the page is styled with, and what is written into them.
 *
 * The page is served with three at the end of the head, each marked with which of the
 * three it is: the mode, the theme's own file, and the two multipliers. The theme's
 * follows the mode's and the sizes follow both.
 */
import type { Sizes } from '@/entities/settings'

/** The elements the page carries: the mode's, the theme's, and the sizes'. */
export interface StyleElements {
  readonly mode: HTMLStyleElement
  readonly theme: HTMLStyleElement
  /** Nothing for a page served at no size of its own, until one is written. */
  sizes: HTMLStyleElement | undefined
}

/**
 * The attribute each of the three carries, and which element it identifies.
 * The application writes these when it styles the page.
 */
export const MARKER = 'data-appearance'
export const IS_MODE = 'mode'
export const IS_THEME = 'theme'
export const IS_SIZES = 'sizes'

/** The element the page was served marked as one of the three, and nothing where it carries none. */
const findStyle = (sheet: Document, is: string): HTMLStyleElement | null =>
  sheet.head.querySelector<HTMLStyleElement>(`style[${MARKER}="${is}"]`)

/**
 * The elements the head ends with. A page served by something that dresses it
 * in nothing is given a mode's and a theme's of its own, in that order.
 */
export const getStyleElements = (sheet: Document): StyleElements => {
  const mode = findStyle(sheet, IS_MODE) ?? sheet.head.appendChild(createStyle(IS_MODE, sheet))
  const theme = findStyle(sheet, IS_THEME) ?? addAfter(mode, IS_THEME, sheet)
  return { mode, theme, sizes: findStyle(sheet, IS_SIZES) ?? undefined }
}

/** One of the three, marked as which of them it is. */
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
