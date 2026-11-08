/** The CSS of the window frame: the marker Wails reads to resize the window, and the bar height. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { TITLE_BAR_HEIGHT } from './lib/sizes'

const read = (file: string): string => readFileSync(resolve(import.meta.dirname, file), 'utf8')

const readRule = (css: string, selector: string): string => {
  const start = css.indexOf(`${selector} {`)
  return start < 0 ? '' : css.slice(start, css.indexOf('}', start))
}

describe('the stylesheet of the frame', () => {
  it('lets Windows and Linux resize the frameless window by every edge', () => {
    expect(readRule(read('ui/windowFrame.css'), 'body')).toMatch(/--wails-resize:\s*all;/)
  })

  it('holds the height of a title bar as a token', () => {
    const tokens = readRule(read('tokens/frame.css'), ':root')

    expect(tokens).toContain(`--numen-title-bar-height: ${TITLE_BAR_HEIGHT}px;`)
  })
})
