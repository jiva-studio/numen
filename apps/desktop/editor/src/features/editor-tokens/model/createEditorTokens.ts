/**
 * The window follows the editor settings: it writes the font size, the line height and the font
 * family as tokens of the page, last in the head so a theme cannot change them.
 */
import { watch } from 'vue'
import { win } from '@numen/ui'
import type { SettingsStore } from '@/entities/settings'
import { getEditorCss } from '../lib/editorCss'

const MARKER = 'data-appearance'
const IS_EDITOR = 'editor'

/** The values of the page where the settings could not be read. */
const DEFAULT_FONT_SIZE = 14
const DEFAULT_LINE_HEIGHT = 1.6

export function createEditorTokens(
  store: Pick<SettingsStore, 'read'>,
  sheet: Document = win.document,
): { start(): () => void } {
  const readNumber = (key: string, fallback: number): number => {
    const value = store.read.getCommittedSetting(['app', 'editor', key])
    return typeof value === 'number' ? value : fallback
  }

  const readCss = (): string => {
    const family = store.read.getCommittedSetting(['app', 'editor', 'fontFamily'])
    return getEditorCss({
      fontSize: readNumber('fontSize', DEFAULT_FONT_SIZE),
      lineHeight: readNumber('lineHeight', DEFAULT_LINE_HEIGHT),
      fontFamily: typeof family === 'string' ? family : '',
    })
  }

  const getStyle = (): HTMLStyleElement => {
    const found = sheet.head.querySelector<HTMLStyleElement>(`style[${MARKER}="${IS_EDITOR}"]`)
    const style = found ?? sheet.createElement('style')
    style.setAttribute(MARKER, IS_EDITOR)
    sheet.head.appendChild(style)
    return style
  }

  return {
    start: () => {
      const style = getStyle()
      return watch(
        readCss,
        (css) => {
          style.textContent = css
        },
        { immediate: true },
      )
    },
  }
}
