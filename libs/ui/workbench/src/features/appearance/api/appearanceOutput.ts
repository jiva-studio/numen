import { doc } from '@numen/ui'
import { SCHEMES, type Mode, type Sizes } from '../lib/appearance'
import { getStyleElements, getSizesCss, addAfter, IS_SIZES } from './head'

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
